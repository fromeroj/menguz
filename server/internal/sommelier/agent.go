package sommelier

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"menguz/internal/models"
)

// systemPrompt defines the sommelier persona and hard guardrails. The model
// may ONLY cite products returned by tools, must respect live stock, and
// carries the 18+ responsibility of an alcohol retailer.
const systemPrompt = `Eres "Sofía", la sommelier digital de Menguz Vinos y Licores, una vinatería de CDMX con más de 60 años de tradición. Estilo: cálida, experta y sin pretensiones — como las asesoras de Hedonism Wines: haces preguntas inteligentes, escuchas el contexto (comida, ocasión, presupuesto, gustos) y recomiendas con criterio, no con prisa.

REGLAS ESTRICTAS (incumplir es un error grave):
1. NUNCA cites precios, existencias ni productos que no hayas obtenido de las tools buscar_vinos/ficha_vino en ESTA conversación. El catálogo cambia cada noche.
2. Llama buscar_vinos antes de cualquier recomendación concreta. Si una búsqueda sale vacía, ajusta los filtros (relájalos) o propone alternativas del resultado más cercano — di que no hay exactamente eso.
3. Si algo está agotado (existencia 0), NO lo recomendiones; menciona solo si el cliente lo pidió y ofrece alternativa.
4. Precios en MXN. Formato de botellas 750 ml salvo que se indique.
5. Venta de alcohol solo a mayores de 18 años. Si el contexto sugiere menores, declina con amabilidad.
6. Máximo 150 palabras por respuesta, en español mexicano cordial. Puedes usar 1-2 emojis como 🍷 ✨.
7. Cuando recomiendes, cierra invitando a agregar al carrito o a pedir por WhatsApp (55 1132 2231) para confirmar entrega en CDMX en 24h. Pago por transferencia.
8. Si el cliente pide un humano, comparte el WhatsApp. Horario: L-V 9:00-18:30, S 9:00-13:30.

Tu objetivo: que el cliente encuentre la botella correcta para SU momento, dentro de su presupuesto, y que exista en el inventario.`

// Answer runs one sommelier turn: tool-calling loop against DeepSeek when an
// LLM is configured, deterministic fallback otherwise. delta streams text
// chunks (may be nil). It returns the final text and the deduplicated set of
// recommended products surfaced via tools (fallback computes them directly).
func (e *Engine) Answer(ctx context.Context, history []models.ChatMessage, delta func(string)) (string, []Recomendacion, error) {
	if e.LLM == nil {
		text, recs, _ := e.fallback(history, delta)
		return text, recs, nil
	}

	msgs := toMessages(history)
	recommended := map[string]Recomendacion{}
	order := []string{} // preserve first-surface order

	for round := 0; round < e.MaxToolRounds; round++ {
		reply, err := e.LLM.Chat(ctx, systemPrompt, msgs, Tools(), delta)
		if err != nil {
			// graceful degradation: rules over the same inventory
			fbText, fbRecs, _ := e.fallback(history, nil)
			if len(fbRecs) > 0 {
				for _, r := range fbRecs {
					if _, ok := recommended[r.CveArt]; !ok {
						recommended[r.CveArt] = r
						order = append(order, r.CveArt)
					}
				}
			}
			return joinText(replyText(reply), fbText), ordered(recommended, order), nil
		}
		if len(reply.ToolCalls) == 0 {
			return reply.Text, ordered(recommended, order), nil
		}
		// assistant turn with tool calls
		msgs = append(msgs, Message{Role: "assistant", Content: reply.Text, ToolCalls: reply.ToolCalls})
		for _, tc := range reply.ToolCalls {
			result := e.execTool(tc)
			msgs = append(msgs, Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
	}
	// exhausted tool rounds: force a final plain answer
	last, err := e.LLM.Chat(ctx, systemPrompt+"\n\n(Debes responder YA con lo que tienes; no llames más tools.)", msgs, nil, delta)
	if err != nil {
		fbText, fbRecs, _ := e.fallback(history, nil)
		return fbText, fbRecs, nil
	}
	return last.Text, ordered(recommended, order), nil
}

func replyText(r *Reply) string {
	if r == nil {
		return ""
	}
	return r.Text
}

func joinText(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "\n\n" + b
}

func ordered(m map[string]Recomendacion, order []string) []Recomendacion {
	out := []Recomendacion{}
	for _, k := range order {
		if r, ok := m[k]; ok {
			out = append(out, r)
		}
	}
	return out
}

func toMessages(history []models.ChatMessage) []Message {
	out := []Message{}
	for _, m := range history {
		if m.Role == "system" {
			continue
		}
		out = append(out, Message{Role: m.Role, Content: m.Content})
	}
	return out
}

// execTool runs one tool call against the live store and returns the JSON
// string the model sees. It also records recommendations for the UI cards.
func (e *Engine) execTool(tc ToolCall) string {
	var args map[string]any
	_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
	s := func(k string) string { v, _ := args[k].(string); return v }
	f := func(k string) float64 {
		switch v := args[k].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
		return 0
	}

	switch tc.Function.Name {
	case "buscar_vinos":
		recs := e.Search(Filtro{
			Q: s("q"), Tipo: s("tipo"), Uva: s("uva"), Bodega: s("bodega"),
			Region: s("region"), Pais: s("pais"), Ocasion: s("ocasion"), Tag: s("tag"),
			PrecioMin: f("precio_min"), PrecioMax: f("precio_max"),
			Disponibles:   args["disponibles"] == true,
			MaxResultados: int(f("max_resultados")),
		})
		e.collect(recs)
		b, _ := json.Marshal(map[string]any{"resultados": recs, "total": len(recs)})
		return string(b)

	case "ficha_vino":
		ficha, ok := e.FichaVino(s("cve_art"))
		if !ok {
			return `{"error":"producto no encontrado o inactivo"}`
		}
		if rec, ok2 := fichaToRecomendacion(ficha); ok2 {
			e.collect([]Recomendacion{rec})
		}
		b, _ := json.Marshal(ficha)
		return string(b)
	}
	b, _ := json.Marshal(map[string]any{"error": "tool desconocida: " + tc.Function.Name})
	return string(b)
}

// collect feeds the OnRecommend callback with freshly surfaced products.
func (e *Engine) collect(recs []Recomendacion) {
	if e.OnRecommend == nil || len(recs) == 0 {
		return
	}
	cp := make([]Recomendacion, len(recs))
	copy(cp, recs)
	e.OnRecommend(cp)
}

func fichaToRecomendacion(f map[string]any) (Recomendacion, bool) {
	cve, _ := f["cve_art"].(string)
	if cve == "" {
		return Recomendacion{}, false
	}
	r := Recomendacion{CveArt: cve}
	r.Nombre, _ = f["nombre"].(string)
	r.Precio, _ = f["precio"].(float64)
	r.Existencia, _ = f["existencia"].(float64)
	r.Imagen, _ = f["imagen"].(string)
	r.Categoria, _ = f["categoria"].(string)
	r.Bodega, _ = f["bodega"].(string)
	r.Maridaje, _ = f["maridaje"].(string)
	r.Anada, _ = f["anada"].(string)
	if uvas, ok := f["uvas"].([]string); ok {
		r.Uvas = uvas
	}
	return r, true
}

// ---------- deterministic fallback (no DeepSeek key) ----------

// fallbackAnswer keeps the same guardrails without an LLM: it parses intent
// from the last user message, searches the same inventory, and returns a
// curated reply + product cards. OnRecommend fires for parity with the LLM path.
func (e *Engine) fallback(history []models.ChatMessage, delta func(string)) (string, []Recomendacion, error) {
	last := ""
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			last = history[i].Content
			break
		}
	}
	f, proem := parseIntent(last)
	recs := e.Search(f)
	e.collect(recs)

	var b strings.Builder
	switch {
	case len(recs) == 0:
		b.WriteString("No encontré algo que coincida exactamente con eso en el inventario actual 😔. ¿Me das más contexto — presupuesto por botella, tipo de comida u ocasión? También te atiendo por WhatsApp: https://wa.me/525511322231")
	case len(recs) == 1:
		b.WriteString(proem)
		r := recs[0]
		fmt.Fprintf(&b, "Te sugiero **%s**", r.Nombre)
		if r.Bodega != "" {
			fmt.Fprintf(&b, " de %s", r.Bodega)
		}
		fmt.Fprintf(&b, " — $%.2f MXN", r.Precio)
		if r.Existencia <= 0 {
			b.WriteString(" (lo tenemos encargado; confirma disponibilidad por WhatsApp)")
		} else {
			fmt.Fprintf(&b, " ✅ con existencia (%.0f)", r.Existencia)
		}
		if r.Maridaje != "" {
			fmt.Fprintf(&b, ". Marida con: %s", r.Maridaje)
		}
		b.WriteString(". ¿Lo agrego a tu pedido?")
	default:
		fmt.Fprintf(&b, "Esto es lo que más te conviene de nuestro inventario actual:\n")
		for i, r := range recs {
			if i >= 4 {
				break
			}
			fmt.Fprintf(&b, "• %s — $%.2f", r.Nombre, r.Precio)
			if len(r.Uvas) > 0 {
				fmt.Fprintf(&b, " (%s)", strings.Join(r.Uvas, ", "))
			}
			if r.Existencia <= 0 {
				b.WriteString(" — agotado por ahora")
			}
			b.WriteString("\n")
		}
		b.WriteString("¿Cuál te late? Puedo darte la ficha completa o agregarlo a tu pedido.")
	}
	out := b.String()
	if delta != nil {
		delta(out)
	}
	return out, recs, nil
}

var varietalList = []string{
	"malbec", "merlot", "cabernet sauvignon", "cabernet franc", "syrah", "shiraz",
	"pinot noir", "tempranillo", "garnacha", "nebbiolo", "sangiovese", "barbera",
	"zinfandel", "petit verdot", "carmenere", "bonarda", "tannat", "mencía", "mencia",
	"albariño", "albarino", "verdejo", "chardonnay", "sauvignon blanc", "riesling",
	"gewürztraminer", "gewurztraminer", "viognier", "pinot grigio", "pinot gris",
	"moscatel", "torrontés", "torrontes", "lambrusco", "glera", "airén", "airen",
	"palomino", "macabeo", "parellada", "xarel·lo", "xarello", "mourvèdre", "mourvedre",
	"graciano", "monastrell", "carignan", "primitivo", "aglianico", "negroamaro",
}

// tokens splits free text into search tokens (stopwords dropped).
func tokens(low string) []string {
	stop := map[string]bool{
		"para": true, "con": true, "una": true, "uno": true, "unos": true, "unas": true,
		"busco": true, "quiero": true, "necesito": true, "recomienda": true, "recomiendame": true,
		"tienen": true, "tenemos": true, "me": true, "mi": true, "de": true, "del": true, "la": true,
		"el": true, "los": true, "las": true, "en": true, "y": true, "o": true, "por": true,
		"favor": true, "gracias": true, "algo": true, "botella": true, "botellas": true,
		"vino": true, "vinos": true, "parece": true, "parecen": true,
	}
	var out []string
	for _, t := range strings.Fields(low) {
		t = strings.Trim(t, "¿?¡!.,;:()\"'$")
		if t != "" && !stop[t] && len(t) > 2 {
			out = append(out, t)
		}
	}
	return out
}

// parseIntent extracts a Filtro from free text (shared by the fallback and
// quick-prompt chips). Returns a friendly proem acknowledging the intent.
func parseIntent(msg string) (Filtro, string) {
	low := strings.ToLower(msg)
	f := Filtro{Disponibles: true, MaxResultados: 8}

	switch {
	case strings.Contains(low, "regalo"), strings.Contains(low, "regalar"), strings.Contains(low, "obsequio"):
		f.Ocasion = "regalo"
		return f, "Para regalo busco algo con buena presentación y carácter ✨. "
	case strings.Contains(low, "romántica"), strings.Contains(low, "romantica"), strings.Contains(low, "aniversario"), strings.Contains(low, "pareja"):
		f.Ocasion = "cena romántica"
		return f, "Para una cena romántica 🕯️ apuesto por tintos sedosos o un espumoso elegante. "
	case strings.Contains(low, "asado"), strings.Contains(low, "carne"), strings.Contains(low, "parrilla"), strings.Contains(low, "bbq"), strings.Contains(low, "barbacoa"):
		f.Tag = "tánico"
		f.Tipo = "Tinto"
		return f, "Con carne/asado manda un tinto con estructura 🥩. "
	case strings.Contains(low, "pescado"), strings.Contains(low, "marisco"), strings.Contains(low, "ceviche"), strings.Contains(low, "sushi"):
		f.Tipo = "Blanco"
		return f, "Para mar y sushi, blancos frescos o un rosado vivo 🐟. "
	case strings.Contains(low, "celebraci"), strings.Contains(low, "brindis"), strings.Contains(low, "fiesta"), strings.Contains(low, "boda"), strings.Contains(low, "cumpleaños"), strings.Contains(low, "cumpleanos"):
		f.Ocasion = "celebración"
		return f, "¡A brindar! 🥂 Busco espumosos y champagnes con chispa. "
	case strings.Contains(low, "verano"), strings.Contains(low, "terraza"), strings.Contains(low, "calor"), strings.Contains(low, "pool"):
		f.Ocasion = "verano"
		return f, "Para el calor ☀️ algo fresquito y fácil de tomar. "
	case strings.Contains(low, "dulce"), strings.Contains(low, "postre"), strings.Contains(low, "chocolate"):
		f.Tag = "dulce"
		return f, "Con postre, dulzor a la vista 🍫. "
	case strings.Contains(low, "digestivo"), strings.Contains(low, "café"), strings.Contains(low, "cafe"), strings.Contains(low, "after"):
		f.Ocasion = "digestivo"
		return f, "Para cerrar con broche de oro ☕. "
	case strings.Contains(low, "cristal"), strings.Contains(low, "reserva"):
		f.Tag = "crianza"
		return f, "Buscaste botellas con crianza — hay joyas. "
	}

	// type
	for _, t := range []struct{ k, v string }{
		{"tinto", "Tinto"}, {"blanco", "Blanco"}, {"rosado", "Rosado"}, {"rosé", "Rosado"}, {"rose", "Rosado"},
		{"espumoso", "Espumoso"}, {"champagne", "Champagne"}, {"champán", "Champagne"}, {"champan", "Champagne"},
		{"licor", "Licor"}, {"mezcal", "Licor"}, {"tequila", "Licor"}, {"whisky", "Licor"}, {"ron", "Licor"},
	} {
		if strings.Contains(low, t.k) {
			f.Tipo = t.v
			break
		}
	}
	// varietal
	for _, u := range varietalList {
		if strings.Contains(low, u) {
			f.Uva = u
			break
		}
	}
	// budget
	if n := extractMoney(low); n > 0 {
		f.PrecioMax = n
		return f, fmt.Sprintf("Con presupuesto hasta $%.0f, esto va muy bien 👌. ", n)
	}
	// free text minus stopwords becomes q
	f.Q = strings.TrimSpace(strings.Join(tokens(low), " "))
	return f, "Déjame revisar el inventario actual 🍷. "
}

func extractMoney(low string) float64 {
	fields := strings.FieldsFunc(low, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r == '$' || r == ',')
	})
	for _, w := range fields {
		w = strings.Trim(w, "$,")
		w = strings.ReplaceAll(w, ",", "")
		if n, err := strconv.ParseFloat(w, 64); err == nil && n >= 50 && n <= 100000 {
			return n
		}
	}
	return 0
}
