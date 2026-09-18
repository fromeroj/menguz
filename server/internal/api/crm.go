package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// ---------- CRM base (Month 3): pipeline, customer 360, segmentation, export ----------

var pipelineStages = []string{"nuevo", "contactado", "propuesta", "ganado", "perdido"}

func (s *Server) crmPipelineStages(c echo.Context) error {
	return c.JSON(http.StatusOK, pipelineStages)
}

// ---------- Deals ----------

func (s *Server) crmListDeals(c echo.Context) error {
	etapa := c.QueryParam("etapa")
	out := []models.Deal{}
	var d models.Deal
	err := s.Store.ScanPrefix(badger.PrefDeal, &d, func(key string) bool {
		if etapa != "" && d.Etapa != etapa {
			return true
		}
		out = append(out, d)
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

type dealReq struct {
	ClienteID string  `json:"cliente_id"`
	Titulo    string  `json:"titulo" validate:"required,min=3"`
	Monto     float64 `json:"monto"`
	Etapa     string  `json:"etapa" validate:"required,oneof=nuevo contactado propuesta ganado perdido"`
	Notas     string  `json:"notas"`
	Origen    string  `json:"origen"`
}

func (s *Server) crmCreateDeal(c echo.Context) error {
	var req dealReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	d := models.Deal{
		ID: newID("deal"), ClienteID: req.ClienteID, Titulo: req.Titulo, Monto: req.Monto,
		Etapa: req.Etapa, Notas: req.Notas, Origen: req.Origen,
		CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
	}
	if err := s.Store.PutJSON(badger.PrefDeal+d.ID, d); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, d)
}

func (s *Server) crmUpdateDeal(c echo.Context) error {
	id := c.Param("id")
	var d models.Deal
	if err := s.Store.GetJSON(badger.PrefDeal+id, &d); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "deal no encontrado")
	}
	var req dealReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	d.ClienteID = req.ClienteID
	d.Titulo = req.Titulo
	d.Monto = req.Monto
	d.Etapa = req.Etapa
	d.Notas = req.Notas
	d.Origen = req.Origen
	d.UpdatedAt = nowUTC()
	if err := s.Store.PutJSON(badger.PrefDeal+id, d); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, d)
}

func (s *Server) crmDeleteDeal(c echo.Context) error {
	if err := s.Store.Delete(badger.PrefDeal + c.Param("id")); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "deal no encontrado")
	}
	return c.NoContent(http.StatusNoContent)
}

// ---------- Customer 360 ----------

// customerStats computes aggregates from synced SAE invoices + web orders.
type customerStats struct {
	TotalFacturado float64
	UltimaCompra   *time.Time
	Frecuencia     int // invoices per month (approx)
	WebOrders      []*models.Orden
}

func (s *Server) statsFor(clienteID string) customerStats {
	st := customerStats{}
	var f models.Factura
	count := 0
	_ = s.Store.ScanPrefix(badger.PrefFactura, &f, func(key string) bool {
		if f.ClienteID != clienteID {
			return true
		}
		st.TotalFacturado += f.Total
		count++
		t := f.Fecha
		if st.UltimaCompra == nil || t.After(*st.UltimaCompra) {
			st.UltimaCompra = &t
		}
		return true
	})
	if count > 0 {
		st.Frecuencia = count / 12
		if st.Frecuencia < 1 {
			st.Frecuencia = 1
		}
	}
	var o models.Orden
	_ = s.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		if o.ClienteID == clienteID {
			cp := o
			st.WebOrders = append(st.WebOrders, &cp)
		}
		return true
	})
	sort.Slice(st.WebOrders, func(i, j int) bool { return st.WebOrders[i].CreatedAt.After(st.WebOrders[j].CreatedAt) })
	return st
}

func (s *Server) crmListCustomers(c echo.Context) error {
	q := lower(c.QueryParam("q"))
	out := []map[string]any{}
	var cli models.Cliente
	_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(key string) bool {
		if q != "" && !contains(lower(cli.Nombre), q) && !contains(lower(cli.Email), q) && !contains(lower(cli.ID), q) {
			return true
		}
		st := s.statsFor(cli.ID)
		out = append(out, map[string]any{
			"cliente":         withoutHash(cli),
			"total_facturado": round2(st.TotalFacturado),
			"ultima_compra":   st.UltimaCompra,
			"frecuencia_mes":  st.Frecuencia,
			"web_orders":      len(st.WebOrders),
		})
		return true
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i]["total_facturado"].(float64) > out[j]["total_facturado"].(float64)
	})
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

func (s *Server) crmCustomer360(c echo.Context) error {
	id := c.Param("id")
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+id, &cli); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "cliente no encontrado")
	}
	st := s.statsFor(id)
	// attach invoice headers
	facs := []models.Factura{}
	var f models.Factura
	_ = s.Store.ScanPrefix(badger.PrefFactura, &f, func(key string) bool {
		if f.ClienteID == id {
			f.Detalles = nil
			facs = append(facs, f)
		}
		return true
	})
	sort.Slice(facs, func(i, j int) bool { return facs[i].Fecha.After(facs[j].Fecha) })
	if len(facs) > 24 {
		facs = facs[:24]
	}
	c360 := models.Customer360{
		Cliente:        withoutHashPtr(&cli),
		TotalFacturado: round2(st.TotalFacturado),
		UltimaCompra:   st.UltimaCompra,
		WebOrders:      st.WebOrders,
		Facturas:       facs,
		Frecuencia:     st.Frecuencia,
	}
	c360.Segmentos = s.segmentsFor(&cli, &st)
	return c.JSON(http.StatusOK, c360)
}

// ---------- Segmentation ----------

// segmentsFor computes which built-in segments a customer belongs to.
func (s *Server) segmentsFor(cli *models.Cliente, st *customerStats) []string {
	var segs []string
	now := nowUTC()
	if st.TotalFacturado >= 25000 {
		segs = append(segs, "alto_valor")
	}
	if st.UltimaCompra != nil && now.Sub(*st.UltimaCompra) > 90*24*time.Hour {
		segs = append(segs, "sin_compra_90d")
	}
	if st.Frecuencia >= 2 {
		segs = append(segs, "frecuente")
	}
	if cli.Saldo > 0 {
		segs = append(segs, "con_saldo")
	}
	if cli.B2BEnabled {
		segs = append(segs, "b2b_activo")
	}
	return segs
}

var segmentDefs = map[string]string{
	"alto_valor":     "Clientes con más de $25,000 facturados (histórico SAE)",
	"sin_compra_90d": "Clientes sin compra en los últimos 90 días",
	"frecuente":      "Clientes con 2+ compras al mes en promedio",
	"con_saldo":      "Clientes con saldo pendiente (cobranza)",
	"b2b_activo":     "Clientes con acceso al portal B2B",
}

func (s *Server) crmListSegments(c echo.Context) error {
	return c.JSON(http.StatusOK, segmentDefs)
}

// crmSegmentMembers lists the customers belonging to a segment.
func (s *Server) crmSegmentMembers(c echo.Context) error {
	seg := c.Param("segmento")
	if _, ok := segmentDefs[seg]; !ok {
		return badRequest("segmento inválido: " + seg)
	}
	out := []map[string]any{}
	var cli models.Cliente
	_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(key string) bool {
		st := s.statsFor(cli.ID)
		segs := s.segmentsFor(&cli, &st)
		for _, sg := range segs {
			if sg == seg {
				out = append(out, map[string]any{
					"id": cli.ID, "nombre": cli.Nombre, "email": cli.Email,
					"total_facturado": round2(st.TotalFacturado), "ultima_compra": st.UltimaCompra,
					"segmentos": segs,
				})
				break
			}
		}
		return true
	})
	return c.JSON(http.StatusOK, map[string]any{"segmento": seg, "items": out, "total": len(out)})
}

// ---------- Structured export (for CRM onboarding / external tools) ----------

func (s *Server) crmExport(c echo.Context) error {
	kind := c.Param("kind") // customers | orders | deals
	w := c.Response().Writer
	c.Response().Header().Set("Content-Type", "text/csv; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=menguz-%s-%s.csv", kind, nowUTC().Format("20060102")))
	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch kind {
	case "customers":
		_ = cw.Write([]string{"id", "nombre", "rfc", "email", "telefono", "ciudad", "lista_precio", "saldo", "total_facturado", "ultima_compra"})
		var cli models.Cliente
		_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(key string) bool {
			st := s.statsFor(cli.ID)
			ult := ""
			if st.UltimaCompra != nil {
				ult = st.UltimaCompra.Format("2006-01-02")
			}
			_ = cw.Write([]string{cli.ID, cli.Nombre, cli.RFC, cli.Email, cli.Telefono, cli.Ciudad,
				itoa(int64(cli.ListaPrecio)), fmt.Sprintf("%.2f", cli.Saldo),
				fmt.Sprintf("%.2f", st.TotalFacturado), ult})
			return true
		})
	case "orders":
		_ = cw.Write([]string{"folio", "tipo", "cliente", "estado", "total", "referencia_pago", "created_at"})
		var o models.Orden
		_ = s.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
			_ = cw.Write([]string{o.Folio, o.Tipo, o.ClienteNombre, o.Estado,
				fmt.Sprintf("%.2f", o.Total), o.ReferenciaPago, o.CreatedAt.Format(time.RFC3339)})
			return true
		})
	case "deals":
		_ = cw.Write([]string{"id", "titulo", "cliente_id", "monto", "etapa", "origen", "updated_at"})
		var d models.Deal
		_ = s.Store.ScanPrefix(badger.PrefDeal, &d, func(key string) bool {
			_ = cw.Write([]string{d.ID, d.Titulo, d.ClienteID, fmt.Sprintf("%.2f", d.Monto), d.Etapa, d.Origen, d.UpdatedAt.Format(time.RFC3339)})
			return true
		})
	default:
		return badRequest("tipo de exportación inválido (customers|orders|deals)")
	}
	return nil
}

// ---------- tiny string helpers (avoid extra deps) ----------

func lower(s string) string { return fmt.Sprintf("%s", toLower(s)) }

func toLower(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if ch >= 'A' && ch <= 'Z' {
			b[i] = ch + 32
		}
	}
	return string(b)
}

func contains(hay, needle string) bool {
	return len(needle) == 0 || (len(hay) >= len(needle) && indexOf(hay, needle) >= 0)
}

func indexOf(hay, needle string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func withoutHash(c models.Cliente) models.Cliente {
	c.B2BPassHash = ""
	return c
}

func withoutHashPtr(c *models.Cliente) *models.Cliente {
	cp := *c
	cp.B2BPassHash = ""
	return &cp
}
