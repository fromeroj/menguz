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

// ---------- Campaign CRUD (Editor de Campañas / promos admin) ----------

func (s *Server) adminListCampaigns(c echo.Context) error {
	out := []models.Campana{}
	var camp models.Campana
	err := s.Store.ScanPrefix(badger.PrefCampana, &camp, func(key string) bool {
		out = append(out, camp)
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

func (s *Server) adminGetCampaign(c echo.Context) error {
	id := c.Param("id")
	var camp models.Campana
	if err := s.Store.GetJSON(badger.PrefCampana+id, &camp); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "campaña no encontrada")
	}
	return c.JSON(http.StatusOK, camp)
}

type campaignReq struct {
	Titulo       string  `json:"titulo" validate:"required,min=3"`
	Tipo         string  `json:"tipo" validate:"required,oneof=carousel producto_mes especial"`
	ImagenURL    string  `json:"imagen_url"`
	Descripcion  string  `json:"descripcion"`
	CveArt       string  `json:"cve_art"`
	PrecioOferta float64 `json:"precio_oferta"`
	CtaTexto     string  `json:"cta_texto"`
	CtaURL       string  `json:"cta_url"`
	WAMensaje    string  `json:"wa_mensaje"`
	Inicio       string  `json:"inicio" validate:"required"` // RFC3339 or YYYY-MM-DD
	Fin          string  `json:"fin" validate:"required"`
	Activa       bool    `json:"activa"`
	Orden        int     `json:"orden"`
}

func parseCampaignTime(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t, nil
	}
	return time.Time{}, badRequest("fecha inválida, usa RFC3339 o YYYY-MM-DD: " + v)
}

func (s *Server) adminCreateCampaign(c echo.Context) error {
	var req campaignReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	inicio, err := parseCampaignTime(req.Inicio)
	if err != nil {
		return err
	}
	fin, err := parseCampaignTime(req.Fin)
	if err != nil {
		return err
	}
	if !fin.After(inicio) {
		return badRequest("fin debe ser posterior a inicio")
	}
	if req.CveArt != "" {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+req.CveArt, &p); err != nil {
			return badRequest("cve_art no existe en el catálogo: " + req.CveArt)
		}
	}
	camp := models.Campana{
		ID: newID("camp"), Titulo: req.Titulo, Tipo: req.Tipo,
		ImagenURL: req.ImagenURL, Descripcion: req.Descripcion, CveArt: strings.TrimSpace(req.CveArt),
		PrecioOferta: req.PrecioOferta, CtaTexto: req.CtaTexto, CtaURL: req.CtaURL, WAMensaje: req.WAMensaje,
		Inicio: inicio, Fin: fin, Activa: req.Activa, Orden: req.Orden,
		CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
	}
	if err := s.Store.PutJSON(badger.PrefCampana+camp.ID, camp); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, camp)
}

func (s *Server) adminUpdateCampaign(c echo.Context) error {
	id := c.Param("id")
	var camp models.Campana
	if err := s.Store.GetJSON(badger.PrefCampana+id, &camp); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "campaña no encontrada")
	}
	var req campaignReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	inicio, err := parseCampaignTime(req.Inicio)
	if err != nil {
		return err
	}
	fin, err := parseCampaignTime(req.Fin)
	if err != nil {
		return err
	}
	if !fin.After(inicio) {
		return badRequest("fin debe ser posterior a inicio")
	}
	camp.Titulo = req.Titulo
	camp.Tipo = req.Tipo
	camp.ImagenURL = req.ImagenURL
	camp.Descripcion = req.Descripcion
	camp.CveArt = strings.TrimSpace(req.CveArt)
	camp.PrecioOferta = req.PrecioOferta
	camp.CtaTexto = req.CtaTexto
	camp.CtaURL = req.CtaURL
	camp.WAMensaje = req.WAMensaje
	camp.Inicio = inicio
	camp.Fin = fin
	camp.Activa = req.Activa
	camp.Orden = req.Orden
	camp.UpdatedAt = nowUTC()
	if err := s.Store.PutJSON(badger.PrefCampana+id, camp); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, camp)
}

func (s *Server) adminToggleCampaign(c echo.Context) error {
	id := c.Param("id")
	var camp models.Campana
	if err := s.Store.GetJSON(badger.PrefCampana+id, &camp); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "campaña no encontrada")
	}
	camp.Activa = !camp.Activa
	camp.UpdatedAt = nowUTC()
	if err := s.Store.PutJSON(badger.PrefCampana+id, camp); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, camp)
}

func (s *Server) adminDeleteCampaign(c echo.Context) error {
	id := c.Param("id")
	if err := s.Store.Delete(badger.PrefCampana + id); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "campaña no encontrada")
	}
	return c.NoContent(http.StatusNoContent)
}
