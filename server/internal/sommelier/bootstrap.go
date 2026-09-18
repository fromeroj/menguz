package sommelier

import (
	"strings"
	"time"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// knownVarietals maps lowercase substrings (as they appear in article names)
// to canonical varietal names. Used by BootstrapDrafts to pre-fill uvas.
var knownVarietals = map[string]string{
	"malbec": "Malbec", "merlot": "Merlot", "cabernet sauvignon": "Cabernet Sauvignon",
	"cabernet franc": "Cabernet Franc", "syrah": "Syrah", "shiraz": "Shiraz",
	"pinot noir": "Pinot Noir", "tempranillo": "Tempranillo", "garnacha": "Garnacha",
	"nebbiolo": "Nebbiolo", "sangiovese": "Sangiovese", "barbera": "Barbera",
	"zinfandel": "Zinfandel", "petit verdot": "Petit Verdot", "carmenere": "Carmenère",
	"bonarda": "Bonarda", "tannat": "Tannat", "mencia": "Mencía",
	"albariño": "Albariño", "albarino": "Albariño", "verdejo": "Verdejo",
	"chardonnay": "Chardonnay", "sauvignon blanc": "Sauvignon Blanc", "riesling": "Riesling",
	"gewürztraminer": "Gewürztraminer", "gewurztraminer": "Gewürztraminer",
	"viognier": "Viognier", "pinot grigio": "Pinot Grigio", "pinot gris": "Pinot Gris",
	"moscatel": "Moscatel", "torrontés": "Torrontés", "torrontes": "Torrontés",
	"lambrusco": "Lambrusco", "glera": "Glera", "airén": "Airén", "airen": "Airén",
	"palomino": "Palomino", "macabeo": "Macabeo", "parellada": "Parellada",
	"xarel": "Xarel·lo", "mourvedre": "Mourvèdre", "mourvèdre": "Mourvèdre",
	"graciano": "Graciano", "monastrell": "Monastrell", "carignan": "Carignan",
	"primitivo": "Primitivo", "aglianico": "Aglianico", "negroamaro": "Negroamaro",
	"crianza": "", "reserva": "", "gran reserva": "",
}

// tipoFromCategoria maps SAE/frontend categories to sommelier types.
func tipoFromCategoria(cat string) string {
	c := strings.ToLower(strings.TrimSpace(cat))
	switch {
	case strings.Contains(c, "champagne"):
		return "Champagne"
	case strings.Contains(c, "espumos"):
		return "Espumoso"
	case strings.Contains(c, "rosad"), strings.Contains(c, "rosé"), strings.Contains(c, "rose"):
		return "Rosado"
	case strings.Contains(c, "tinto"):
		return "Tinto"
	case strings.Contains(c, "blanco"):
		return "Blanco"
	case strings.Contains(c, "licor"), strings.Contains(c, "mezcal"), strings.Contains(c, "tequila"),
		strings.Contains(c, "whisky"), strings.Contains(c, "ron"), strings.Contains(c, "vodka"),
		strings.Contains(c, "brandy"), strings.Contains(c, "coñac"):
		return "Licor"
	}
	return ""
}

// BootstrapDrafts creates draft profiles for products that lack one. It only
// fills fields derivable from catalog data (tipo, uvas parsed from the name)
// and NEVER overwrites fields already curated by the admin. Returns counts.
func BootstrapDrafts(store *badger.Store) (created, updated int, err error) {
	e := &Engine{Store: store}
	var p models.Producto
	err = store.ScanPrefix(badger.PrefProducto, &p, func(k string) bool {
		if !p.Activo {
			return true
		}
		pf := e.GetPerfil(p.CveArt)
		changed := false

		tipo := pf.Tipo
		if tipo == "" {
			tipo = tipoFromCategoria(p.Categoria)
		}
		if tipo != "" && pf.Tipo == "" {
			pf.Tipo = tipo
			changed = true
		}
		if len(pf.Uvas) == 0 {
			low := strings.ToLower(p.Nombre)
			uvas := []string{}
			for probe, canon := range knownVarietals {
				if canon == "" {
					continue
				}
				if strings.Contains(low, probe) {
					uvas = append(uvas, canon)
				}
			}
			// keep a stable order
			for i := 1; i < len(uvas); i++ {
				for j := i; j > 0 && uvas[j] < uvas[j-1]; j-- {
					uvas[j], uvas[j-1] = uvas[j-1], uvas[j]
				}
			}
			if len(uvas) > 0 {
				pf.Uvas = uvas
				changed = true
			}
		}
		if pf.Descripcion == "" && strings.TrimSpace(p.Descripcion) != "" {
			pf.Descripcion = strings.TrimSpace(p.Descripcion)
			changed = true
		}
		if pf.Anada == "" {
			// vintage appears in some names, e.g. "Protos 2019"
			for _, w := range strings.Fields(p.Nombre) {
				w = strings.Trim(w, "()")
				if len(w) == 4 && w[0] == '1' || len(w) == 4 && w[0] == '2' {
					if n, ok := parseYear(w); ok && n >= 1980 && n <= time.Now().Year() {
						pf.Anada = w
						changed = true
						break
					}
				}
			}
		}

		if pf.CveArt == "" {
			pf.CveArt = p.CveArt
		}
		if pf.UpdatedAt.IsZero() {
			created++
		} else if changed {
			updated++
		}
		if changed || pf.UpdatedAt.IsZero() {
			if pf.UpdatedAt.IsZero() {
				pf.UpdatedAt = time.Now().UTC()
			}
			_ = store.PutJSON(badger.PrefPerfilVino+pf.CveArt, pf)
		}
		return true
	})
	return created, updated, err
}

func parseYear(w string) (int, bool) {
	if len(w) != 4 {
		return 0, false
	}
	n := 0
	for _, ch := range w {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		n = n*10 + int(ch-'0')
	}
	return n, true
}
