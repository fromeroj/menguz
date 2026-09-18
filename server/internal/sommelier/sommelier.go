// Package sommelier implements the Menguz "Sommelier" — a Hedonism-style
// conversational wine advisor. It is grounded in the live SAE inventory
// (prices + stock) and the curated wine profiles (PerfilVino: grape,
// producer, region, tasting notes, pairing, occasions) via two tools the
// model may call: buscar_vinos and ficha_vino.
//
// Engine selection mirrors the chatbot: DeepSeek (OpenAI-compatible API,
// tool calling, streaming) when DEEPSEEK_API_KEY is set; a deterministic
// rule-based sommelier over the same inventory otherwise. Guardrails keep
// every price/stock claim tied to the synced catalog.
package sommelier

import (
	"fmt"
	"strings"
	"time"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// Engine drives one sommelier conversation turn.
type Engine struct {
	Store         *badger.Store
	LLM           *Client // DeepSeek; nil → deterministic fallback
	MaxToolRounds int
	// OnRecommend, if set, receives every product surfaced by a tool call
	// (deduplicated by the engine) so the caller can render rich cards.
	OnRecommend func([]Recomendacion)
}

// NewEngine builds the engine from config-ish inputs.
func NewEngine(store *badger.Store, llm *Client) *Engine {
	return &Engine{Store: store, LLM: llm, MaxToolRounds: 4}
}

// Recomendacion is the public product card streamed to the storefront.
type Recomendacion struct {
	CveArt     string   `json:"cve_art"`
	Nombre     string   `json:"nombre"`
	Precio     float64  `json:"precio"` // effective price (oferta applied)
	Existencia float64  `json:"existencia"`
	Imagen     string   `json:"imagen"`
	Categoria  string   `json:"categoria"`
	Bodega     string   `json:"bodega,omitempty"`
	Uvas       []string `json:"uvas,omitempty"`
	Anada      string   `json:"anada,omitempty"`
	Maridaje   string   `json:"maridaje,omitempty"`
}

// Filtro is the structured search the model (or fallback) can express.
type Filtro struct {
	Q             string // free text over name/description/profile
	Tipo          string // Tinto | Blanco | Rosado | Espumoso | Champagne | Licor
	Uva           string // single varietal to match in uvas
	Bodega        string
	Region        string
	Pais          string
	Ocasion       string
	Tag           string
	PrecioMin     float64
	PrecioMax     float64
	Disponibles   bool // only in-stock items
	MaxResultados int
}

// Search runs Filtro over the live catalog joined with wine profiles.
func (e *Engine) Search(f Filtro) []Recomendacion {
	if f.MaxResultados <= 0 || f.MaxResultados > 12 {
		f.MaxResultados = 8
	}
	// load profiles into a lookup (small: one entry per wine)
	perfiles := map[string]models.PerfilVino{}
	var pf models.PerfilVino
	_ = e.Store.ScanPrefix(badger.PrefPerfilVino, &pf, func(k string) bool {
		perfiles[pf.CveArt] = pf
		return true
	})
	low := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	tipo, uva, bodega, region, pais, ocasion, tag, q := low(f.Tipo), low(f.Uva), low(f.Bodega),
		low(f.Region), low(f.Pais), low(f.Ocasion), low(f.Tag), low(f.Q)

	type scored struct {
		r Recomendacion
		n int
	}
	var hits []scored
	var p models.Producto
	_ = e.Store.ScanPrefix(badger.PrefProducto, &p, func(k string) bool {
		if !p.Activo {
			return true
		}
		price := p.PrecioBase
		if p.PrecioOferta > 0 && p.PrecioOferta < price {
			price = p.PrecioOferta
		}
		if f.Disponibles && p.Existencia <= 0 {
			return true
		}
		if f.PrecioMax > 0 && price > f.PrecioMax {
			return true
		}
		if f.PrecioMin > 0 && price < f.PrecioMin {
			return true
		}
		prof := perfiles[p.CveArt]
		tipoP := low(prof.Tipo)
		if tipoP == "" {
			tipoP = low(strings.TrimPrefix(p.Categoria, "Vino "))
		}
		if tipo != "" && tipoP != tipo && !strings.Contains(tipoP, tipo) {
			return true
		}
		// uva: exclude only on explicit conflict (profile names other grapes
		// and the name doesn't mention it); empty profile = unknown, pass.
		if uva != "" && len(prof.Uvas) > 0 && !containsFold(prof.Uvas, uva) && !strings.Contains(low(p.Nombre), uva) {
			return true
		}
		// soft filters: exclude only when the profile explicitly conflicts;
		// unknown (empty) profile fields are neutral, matches score higher.
		n := 0
		if bodega != "" {
			if conflictExclude(low(prof.Bodega), bodega) {
				return true
			}
			if strings.Contains(low(prof.Bodega), bodega) {
				n += 2
			}
		}
		if region != "" {
			if conflictExclude(low(prof.Region+" "+p.Origen), region) {
				return true
			}
			if strings.Contains(low(prof.Region), region) {
				n += 2
			}
		}
		if pais != "" {
			if conflictExclude(low(prof.Pais), pais) {
				return true
			}
			if strings.Contains(low(prof.Pais), pais) {
				n += 2
			}
		}
		if ocasion != "" {
			occJoin := low(strings.Join(prof.Ocasiones, " "))
			if conflictExclude(occJoin, ocasion) {
				return true
			}
			if strings.Contains(occJoin, ocasion) || containsFold(prof.Ocasiones, ocasion) {
				n += 2
			}
		}
		if tag != "" {
			tagJoin := low(strings.Join(prof.Tags, " "))
			if conflictExclude(tagJoin, tag) {
				return true
			}
			if strings.Contains(tagJoin, tag) || containsFold(prof.Tags, tag) {
				n += 2
			}
		}
		// free-text: token overlap over the enriched haystack
		if q != "" {
			hay := low(p.Nombre + " " + p.Descripcion + " " + p.Categoria + " " + p.Origen + " " +
				prof.Bodega + " " + prof.Region + " " + prof.Pais + " " + prof.Maridaje + " " +
				prof.NotasCata + " " + strings.Join(prof.Uvas, " ") + " " + strings.Join(prof.Tags, " "))
			for _, t := range strings.Fields(q) {
				if len(t) > 2 && strings.Contains(hay, t) {
					n++
				}
			}
			if n == 0 {
				return true
			}
		}
		rec := Recomendacion{
			CveArt: p.CveArt, Nombre: p.Nombre, Precio: price, Existencia: p.Existencia,
			Imagen: strings.TrimPrefix(p.ImagenURL, "/"), Categoria: p.Categoria,
			Bodega: prof.Bodega, Uvas: prof.Uvas, Anada: prof.Anada, Maridaje: prof.Maridaje,
		}
		// prefer in-stock in ranking tie-break
		if p.Existencia > 0 {
			n++
		}
		hits = append(hits, scored{r: rec, n: n})
		return true
	})
	// sort: score desc, then in-stock first, then price asc
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && less(hits[j], hits[j-1]); j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := []Recomendacion{}
	for i, h := range hits {
		if i >= f.MaxResultados {
			break
		}
		out = append(out, h.r)
	}
	return out
}

func less(a, b struct {
	r Recomendacion
	n int
}) bool {
	if a.n != b.n {
		return a.n > b.n
	}
	if (a.r.Existencia > 0) != (b.r.Existencia > 0) {
		return a.r.Existencia > 0
	}
	return a.r.Precio < b.r.Precio
}

func containsFold(list []string, s string) bool {
	for _, x := range list {
		if strings.Contains(strings.ToLower(x), s) {
			return true
		}
	}
	return false
}

// conflictExclude reports whether an explicit profile value contradicts a
// soft filter. Empty profile = unknown = no conflict.
func conflictExclude(profileVal, filt string) bool {
	return profileVal != "" && !strings.Contains(profileVal, filt)
}

// GetPerfil loads the curated profile for one product (or a zero one).
func (e *Engine) GetPerfil(cve string) models.PerfilVino {
	var pf models.PerfilVino
	if err := e.Store.GetJSON(badger.PrefPerfilVino+cve, &pf); err != nil {
		pf.CveArt = cve
	}
	return pf
}

// SavePerfil upserts a curated profile (admin panel / API).
func (e *Engine) SavePerfil(pf models.PerfilVino) error {
	pf.CveArt = strings.TrimSpace(pf.CveArt)
	if pf.CveArt == "" {
		return fmt.Errorf("cve_art requerido")
	}
	pf.UpdatedAt = time.Now().UTC()
	return e.Store.PutJSON(badger.PrefPerfilVino+pf.CveArt, pf)
}

// FichaVino returns the full sommelier card for one product: identity,
// effective price, live stock, image, profile — everything the model or
// the storefront needs for a product detail answer.
func (e *Engine) FichaVino(cve string) (map[string]any, bool) {
	var p models.Producto
	if err := e.Store.GetJSON(badger.PrefProducto+cve, &p); err != nil || !p.Activo {
		return nil, false
	}
	price := p.PrecioBase
	if p.PrecioOferta > 0 && p.PrecioOferta < price {
		price = p.PrecioOferta
	}
	pf := e.GetPerfil(cve)
	tipo := pf.Tipo
	if tipo == "" {
		tipo = strings.TrimPrefix(p.Categoria, "Vino ")
	}
	return map[string]any{
		"cve_art": p.CveArt, "nombre": p.Nombre, "categoria": p.Categoria,
		"tipo": tipo, "precio": price, "existencia": p.Existencia,
		"imagen":      strings.TrimPrefix(p.ImagenURL, "/"),
		"descripcion": firstNonEmpty(pf.Descripcion, p.Descripcion),
		"bodega":      pf.Bodega, "uvas": pf.Uvas, "region": pf.Region, "pais": pf.Pais,
		"anada": firstNonEmpty(pf.Anada, "sin añada (NV)"), "graduacion": firstNonEmpty(pf.Graduacion, p.Graduacion),
		"crianza": pf.Crianza, "notas_cata": firstNonEmpty(pf.NotasCata, p.NotasCata),
		"maridaje": pf.Maridaje, "ocasiones": pf.Ocasiones, "tags": pf.Tags,
	}, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
