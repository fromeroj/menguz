package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// ---------- Public catalog ----------

// dataProductsJSON serves the exact shape the storefront Especiales section
// consumes: [{n, p, c, g, i}] at GET /data/products.json.
func (s *Server) dataProductsJSON(c echo.Context) error {
	out := []models.PublicProductJSON{}
	var p models.Producto
	err := s.Store.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		if p.Activo {
			price := p.PrecioBase
			if p.PrecioOferta > 0 && p.PrecioOferta < price {
				price = p.PrecioOferta
			}
			out = append(out, models.PublicProductJSON{
				N: p.Nombre, P: price, C: p.Categoria, G: p.Grupo, I: p.ImagenURL,
			})
		}
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N < out[j].N })
	return c.JSON(http.StatusOK, out)
}

type productFilter struct {
	Q         string `query:"q"`
	Categoria string `query:"categoria"`
	Grupo     string `query:"grupo"`
	Page      int    `query:"page"`
	PerPage   int    `query:"per_page"`
}

// listProducts is the richer public API (search, filters, pagination).
func (s *Server) listProducts(c echo.Context) error {
	var f productFilter
	if err := c.Bind(&f); err != nil {
		return badRequest("filtros inválidos")
	}
	if f.PerPage <= 0 || f.PerPage > 100 {
		f.PerPage = 24
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	q := strings.ToLower(strings.TrimSpace(f.Q))
	type hit struct {
		p   models.Producto
		key string
	}
	var all []hit
	var p models.Producto
	err := s.Store.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		if !p.Activo {
			return true
		}
		if f.Categoria != "" && p.Categoria != f.Categoria {
			return true
		}
		if f.Grupo != "" && p.Grupo != f.Grupo {
			return true
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Nombre), q) &&
			!strings.Contains(strings.ToLower(p.Descripcion), q) &&
			!strings.Contains(strings.ToLower(p.Origen), q) {
			return true
		}
		all = append(all, hit{p: p, key: key})
		return true
	})
	if err != nil {
		return err
	}
	total := len(all)
	start := (f.Page - 1) * f.PerPage
	if start > total {
		start = total
	}
	end := start + f.PerPage
	if end > total {
		end = total
	}
	items := make([]models.Producto, 0, end-start)
	for _, h := range all[start:end] {
		items = append(items, h.p)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"items": items, "total": total, "page": f.Page, "per_page": f.PerPage,
	})
}

func (s *Server) getProduct(c echo.Context) error {
	cve := c.Param("cve")
	var p models.Producto
	if err := s.Store.GetJSON(badger.PrefProducto+cve, &p); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "producto no encontrado")
	}
	// attach active campaigns referencing this product
	var camp models.Campana
	promos := []models.Campana{}
	_ = s.Store.ScanPrefix(badger.PrefCampana, &camp, func(key string) bool {
		if camp.Activa && camp.CveArt == cve && nowUTC().Before(camp.Fin) && nowUTC().After(camp.Inicio) {
			promos = append(promos, camp)
		}
		return true
	})
	return c.JSON(http.StatusOK, map[string]any{"producto": p, "promociones": promos})
}

func (s *Server) listCategories(c echo.Context) error {
	set := map[string]int{}
	var p models.Producto
	_ = s.Store.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		if p.Activo {
			set[p.Categoria]++
		}
		return true
	})
	type cat struct {
		Nombre string `json:"nombre"`
		Grupo  string `json:"grupo"`
		Total  int    `json:"total"`
	}
	out := []cat{}
	var pp models.Producto
	seen := map[string]string{}
	_ = s.Store.ScanPrefix(badger.PrefProducto, &pp, func(key string) bool {
		if pp.Activo {
			seen[pp.Categoria] = pp.Grupo
		}
		return true
	})
	for nombre, grupo := range seen {
		out = append(out, cat{Nombre: nombre, Grupo: grupo, Total: set[nombre]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return c.JSON(http.StatusOK, out)
}

// ---------- Active promos (public) ----------

func (s *Server) activePromos(c echo.Context) error {
	tipo := c.QueryParam("tipo") // carousel | producto_mes | especial | ""
	now := nowUTC()
	out := []models.Campana{}
	var camp models.Campana
	err := s.Store.ScanPrefix(badger.PrefCampana, &camp, func(key string) bool {
		if !camp.Activa {
			return true
		}
		if tipo != "" && camp.Tipo != tipo {
			return true
		}
		if now.Before(camp.Inicio) || now.After(camp.Fin) {
			return true
		}
		// decorate linked product price
		if camp.CveArt != "" {
			var p models.Producto
			if s.Store.GetJSON(badger.PrefProducto+camp.CveArt, &p) == nil && camp.PrecioOferta <= 0 {
				if p.PrecioOferta > 0 {
					camp.PrecioOferta = p.PrecioOferta
				} else {
					camp.PrecioOferta = p.PrecioBase
				}
			}
		}
		out = append(out, camp)
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Orden != out[j].Orden {
			return out[i].Orden < out[j].Orden
		}
		return out[i].Inicio.Before(out[j].Inicio)
	})
	return c.JSON(http.StatusOK, out)
}

// Health & meta

func (s *Server) health(c echo.Context) error {
	lastSync, _ := s.Store.GetString(badger.PrefMeta + badger.MetaLastSync)
	return c.JSON(http.StatusOK, map[string]any{
		"status":    "ok",
		"version":   "0.3.0-m1m2m3",
		"last_sync": lastSync,
		"time":      time.Now().Format(time.RFC3339),
	})
}
