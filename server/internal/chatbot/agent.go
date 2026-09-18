// Package chatbot implements the Menguz conversational assistant: OpenAI
// GPT-4o streaming with catalog RAG, plus a deterministic rule-based
// fallback used when no API key is configured. Guardrails keep prices and
// stock claims strictly tied to the synced catalog.
package chatbot

import (
	"fmt"
	"strings"
	"time"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// FAQ answers are static, brand-safe content.
var faq = []struct {
	keys []string
	a    string
}{
	{[]string{"horario", "hora", "abierto", "cierran"}, "Nuestro horario es lunes a viernes 9:00–18:30 y sábados 9:00–13:30. Te esperamos en Menguz Vinos y Licores."},
	{[]string{"ubicacion", "direccion", "donde", "sucursal"}, "Estamos en la Ciudad de México. Escríbenos por WhatsApp al 55 1132 2231 y te compartimos la ubicación exacta y disponibilidad."},
	{[]string{"envio", "entrega", "domicilio", "reparto"}, "Hacemos entregas en CDMX en 24 horas sobre pedidos confirmados. El costo de envío se confirma al momento de la compra."},
	{[]string{"pago", "pagar", "transferencia", "clabe", "efectivo", "tarjeta"}, "Aceptamos transferencia bancaria (te damos una referencia única al confirmar tu pedido) y efectivo contra entrega en zonas de reparto."},
	{[]string{"factura", "cfdi", "rfc"}, "Claro, emitimos factura CFDI. Compártenos tus datos fiscales (RFC, razón social, régimen, CP) por WhatsApp y la recibirás en tu correo."},
	{[]string{"mayoreo", "mayorista", "b2b", "vinateria"}, "Manejamos precios especiales para vinaterías, bares y restaurantes. En la sección Mayorista del sitio encuentras el acceso al portal B2B con tu lista de precios."},
	{[]string{"evento", "cata", "bodas", "fiesta"}, "Organizamos catas y cotizamos para eventos. Escríbenos por WhatsApp con la fecha, número de invitados y presupuesto."},
	{[]string{"gracias"}, "Con gusto. ¿Hay algo más en lo que te pueda ayudar?"},
	{[]string{"hola", "buenas", "buenos dias", "buenas tardes"}, "¡Hola! Bienvenido a Menguz Vinos y Licores. Puedo ayudarte a encontrar productos, cotizar, revisar tu pedido o resolver dudas."},
}

const escalationHint = "No encontré lo que buscas con certeza. Si prefieres, te comunico con un asesor por WhatsApp: https://wa.me/525511322231"

// Agent answers one user message (fallback engine).
type Agent struct {
	Store *badger.Store
}

// tokens extracts lowercase search tokens (drops stopwords).
func tokens(msg string) []string {
	stop := map[string]bool{"de": true, "el": true, "la": true, "los": true, "las": true, "un": true, "una": true, "con": true, "para": true, "que": true, "me": true, "quiero": true, "busco": true, "necesito": true, "tienen": true, "hay": true, "en": true, "y": true, "o": true, "por": true, "favor": true, "porfavor": true, "holaa": true}
	var out []string
	for _, t := range strings.Fields(strings.ToLower(msg)) {
		t = strings.Trim(t, "¿?¡!.,;:()\"")
		if t != "" && !stop[t] && len(t) > 2 {
			out = append(out, t)
		}
	}
	return out
}

// SearchProducts finds catalog matches by token overlap (the RAG retrieval
// used by both the fallback and the OpenAI prompt builder).
func (a *Agent) SearchProducts(msg string, limit int) []models.Producto {
	toks := tokens(msg)
	if len(toks) == 0 {
		return nil
	}
	type scored struct {
		p models.Producto
		n int
	}
	var hits []scored
	var p models.Producto
	_ = a.Store.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		if !p.Activo {
			return true
		}
		hay := strings.ToLower(p.Nombre + " " + p.Descripcion + " " + p.Categoria + " " + p.Origen + " " + p.NotasCata)
		n := 0
		for _, t := range toks {
			if strings.Contains(hay, t) {
				n++
			}
		}
		if n > 0 {
			hits = append(hits, scored{p: p, n: n})
		}
		return true
	})
	// simple insertion sort by score then price availability
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && (hits[j].n > hits[j-1].n || (hits[j].n == hits[j-1].n && hits[j].p.Existencia > hits[j-1].p.Existencia)); j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := []models.Producto{}
	for i, h := range hits {
		if i >= limit {
			break
		}
		out = append(out, h.p)
	}
	return out
}

// Answer produces the fallback reply for a user message.
func (a *Agent) Answer(session *models.ChatSession, msg string) string {
	low := strings.ToLower(msg)

	// 1. explicit escalation
	if strings.Contains(low, "agente") || strings.Contains(low, "humano") || strings.Contains(low, "asesor") {
		session.Escalado = true
		return "Por supuesto. Un asesor te atenderá por WhatsApp enseguida: https://wa.me/525511322231"
	}

	// 2. order tracking: look for folio like MGZ-000123 or reference MENG000123
	if folio := findFolio(msg); folio != "" {
		if ans := a.trackOrder(folio); ans != "" {
			return ans
		}
	}

	// 3. FAQ
	for _, f := range faq {
		for _, k := range f.keys {
			if strings.Contains(low, k) {
				return f.a
			}
		}
	}

	// 4. catalog search
	prods := a.SearchProducts(msg, 4)
	if len(prods) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "Esto es lo que encontré en nuestro catálogo:\n")
		for _, p := range prods {
			price := p.PrecioBase
			if p.PrecioOferta > 0 && p.PrecioOferta < price {
				price = p.PrecioOferta
			}
			stock := "disponible"
			if p.Existencia <= 0 {
				stock = "agotado"
			}
			fmt.Fprintf(&b, "• %s — $%.2f (%s)%s\n", p.Nombre, price, p.Categoria, stockNote(stock))
		}
		b.WriteString("\n¿Quieres cotizar alguno? Te llevo a WhatsApp: https://wa.me/525511322231")
		return b.String()
	}
	return escalationHint
}

func stockNote(stock string) string {
	if stock == "agotado" {
		return " — momentáneamente agotado"
	}
	return ""
}

func findFolio(msg string) string {
	for _, w := range strings.Fields(msg) {
		w = strings.ToUpper(strings.Trim(w, ".,;:"))
		if strings.HasPrefix(w, "MGZ-") && len(w) >= 8 {
			return w
		}
		if strings.HasPrefix(w, "MENG") && len(w) >= 8 {
			return w
		}
	}
	return ""
}

func (a *Agent) trackOrder(folio string) string {
	var o models.Orden
	found := false
	_ = a.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		if o.Folio == folio || o.ReferenciaPago == folio {
			found = true
			return false
		}
		return true
	})
	if !found {
		return "No localicé ese folio. Verifica el número (ej. MGZ-000123) o escríbenos por WhatsApp."
	}
	return fmt.Sprintf("Tu pedido %s está: %s. Total $%.2f. ¿Algo más en lo que ayudarte?", o.Folio, o.Estado, o.Total)
}

// AppendMessage records a message in the session (both engines use this).
func AppendMessage(s *models.ChatSession, role, content string) {
	s.Messages = append(s.Messages, models.ChatMessage{Role: role, Content: content, CreatedAt: time.Now()})
	if len(s.Messages) > 60 { // keep sessions bounded
		s.Messages = s.Messages[len(s.Messages)-60:]
	}
	s.UpdatedAt = time.Now()
}
