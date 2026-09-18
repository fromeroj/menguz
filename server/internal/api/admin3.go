package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/models"
	"menguz/internal/sommelier"
)

// ---------- Admin API (JWT): wine profiles ----------

// adminGetPerfil returns the curated profile for one product.
// GET /api/admin/products/:cve/perfil
func (s *Server) adminGetPerfil(c echo.Context) error {
	eng := sommelier.NewEngine(s.Store, nil)
	pf := eng.GetPerfil(c.Param("cve"))
	var p models.Producto
	_ = s.Store.GetJSON(badger.PrefProducto+pf.CveArt, &p)
	return c.JSON(http.StatusOK, map[string]any{"perfil": pf, "producto": p})
}

// adminSavePerfil upserts the curated profile (JSON body = PerfilVino).
// PUT /api/admin/products/:cve/perfil
func (s *Server) adminSavePerfil(c echo.Context) error {
	var pf models.PerfilVino
	if err := c.Bind(&pf); err != nil {
		return badRequest("JSON inválido")
	}
	pf.CveArt = c.Param("cve")
	var p models.Producto
	if err := s.Store.GetJSON(badger.PrefProducto+pf.CveArt, &p); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "producto no encontrado")
	}
	if err := sommelier.NewEngine(s.Store, nil).SavePerfil(pf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, pf)
}

// adminBootstrapPerfiles generates draft profiles from catalog data.
// POST /api/admin/perfiles/bootstrap
func (s *Server) adminBootstrapPerfiles(c echo.Context) error {
	created, updated, err := sommelier.BootstrapDrafts(s.Store)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"creados": created, "actualizados": updated})
}

// ---------- Admin panel (HTML): wine profile form ----------

func splitList(s string) []string {
	out := []string{}
	for _, x := range strings.Split(s, ",") {
		if t := strings.TrimSpace(x); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func joinList(l []string) string { return strings.Join(l, ", ") }

// adminPerfilForm edits the sommelier profile for one product.
// GET /admin/productos/:cve/perfil
func (s *Server) adminPerfilForm(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	cve := c.Param("cve")
	var p models.Producto
	if err := s.Store.GetJSON(badger.PrefProducto+cve, &p); err != nil {
		return c.Redirect(http.StatusFound, "/admin/productos")
	}
	pf := sommelier.NewEngine(s.Store, nil).GetPerfil(cve)
	tipo := pf.Tipo
	if tipo == "" {
		tipo = strings.TrimPrefix(p.Categoria, "Vino ")
	}
	sel := func(t string) string {
		if tipo == t {
			return "selected"
		}
		return ""
	}
	img := ""
	if p.ImagenURL != "" {
		img = fmt.Sprintf(`<img src="%s" class="w-24 h-36 object-contain rounded-lg border bg-white p-1">`, esc(p.ImagenURL))
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-1">Ficha de sommelier</h1>
<p class="text-sm text-neutral-500 mb-4"><span class="font-mono">%s</span> · %s · %s · existencia %.0f</p>
<div class="flex gap-6 items-start">
<div class="bg-white rounded-2xl shadow p-6 max-w-2xl flex-1">
<form method="post" action="/admin/productos/%s/perfil/guardar" class="space-y-4">
%s
<div class="grid grid-cols-2 gap-4">
  <div><label class="text-sm font-medium">Tipo</label>
    <select name="tipo" class="mt-1 w-full border rounded-lg px-3 py-2">
      <option value="Tinto" %s>Tinto</option><option value="Blanco" %s>Blanco</option>
      <option value="Rosado" %s>Rosado</option><option value="Espumoso" %s>Espumoso</option>
      <option value="Champagne" %s>Champagne</option><option value="Licor" %s>Licor</option>
      <option value="Otro" %s>Otro</option>
    </select></div>
  <div><label class="text-sm font-medium">Añada</label>
    <input name="anada" value="%s" placeholder="2021 · vacío = sin añada" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div class="grid grid-cols-2 gap-4">
  <div><label class="text-sm font-medium">Uvas (separadas por coma)</label>
    <input name="uvas" value="%s" placeholder="Malbec, Cabernet Sauvignon" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Bodega / productor</label>
    <input name="bodega" value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div class="grid grid-cols-3 gap-4">
  <div><label class="text-sm font-medium">Región</label>
    <input name="region" value="%s" placeholder="Valle de Guadalupe, Rioja…" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">País</label>
    <input name="pais" value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Graduación</label>
    <input name="graduacion" value="%s" placeholder="13.5%%" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div><label class="text-sm font-medium">Crianza</label>
  <input name="crianza" value="%s" placeholder="Roble 6 meses, Reserva 24 meses…" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
<div><label class="text-sm font-medium">Notas de cata</label>
  <textarea name="notas_cata" rows="3" placeholder="Color, nariz, boca, final…" class="mt-1 w-full border rounded-lg px-3 py-2">%s</textarea></div>
<div><label class="text-sm font-medium">Maridaje</label>
  <textarea name="maridaje" rows="2" placeholder="Carnes rojas a la parrilla, quesos añejos…" class="mt-1 w-full border rounded-lg px-3 py-2">%s</textarea></div>
<div><label class="text-sm font-medium">Descripción editorial (larga)</label>
  <textarea name="descripcion" rows="4" placeholder="Historia, estilo, por qué comprarla…" class="mt-1 w-full border rounded-lg px-3 py-2">%s</textarea></div>
<div class="grid grid-cols-2 gap-4">
  <div><label class="text-sm font-medium">Ocasiones (coma)</label>
    <input name="ocasiones" value="%s" placeholder="regalo, cena romántica, celebración" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Tags (coma)</label>
    <input name="tags" value="%s" placeholder="frutal, tánico, añejo, artesanal" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div class="flex gap-3 pt-2">
  <button class="bg-[#7B1F3A] text-white px-6 py-2.5 rounded-lg font-semibold">Guardar ficha</button>
  <a href="/admin/productos" class="border px-6 py-2.5 rounded-lg">Volver</a>
</div>
</form></div>
<div class="w-40 text-center text-xs text-neutral-400">%s<p class="mt-2">Imagen del catálogo (se sirve desde el sync)</p></div>
</div>
<p class="text-xs text-neutral-500 mt-4 max-w-2xl">Esta ficha alimenta al Sommelier IA: uvas, notas y maridaje se usan para recomendar. Los borradores se generan desde el catálogo (tipo y uvas detectadas en el nombre); edita a mano lo importante. Los cambios NO se pisan al re-correr el generador de borradores.</p>`,
		esc(p.CveArt), esc(p.Nombre), esc(p.Categoria), p.Existencia,
		esc(cve), csrfInput(c),
		sel("Tinto"), sel("Blanco"), sel("Rosado"), sel("Espumoso"), sel("Champagne"), sel("Licor"), sel("Otro"),
		esc(pf.Anada), esc(joinList(pf.Uvas)), esc(pf.Bodega), esc(pf.Region), esc(pf.Pais), esc(pf.Graduacion),
		esc(pf.Crianza), esc(pf.NotasCata), esc(pf.Maridaje), esc(pf.Descripcion),
		esc(joinList(pf.Ocasiones)), esc(joinList(pf.Tags)), img)
	return c.HTML(http.StatusOK, layout("Ficha de sommelier", "/admin/productos", body))
}

// adminPerfilSave handles the profile form.
// POST /admin/productos/:cve/perfil/guardar
func (s *Server) adminPerfilSave(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	cve := c.Param("cve")
	var p models.Producto
	if err := s.Store.GetJSON(badger.PrefProducto+cve, &p); err != nil {
		return c.Redirect(http.StatusFound, "/admin/productos")
	}
	eng := sommelier.NewEngine(s.Store, nil)
	pf := eng.GetPerfil(cve)
	pf.CveArt = cve
	pf.Tipo = c.FormValue("tipo")
	pf.Anada = strings.TrimSpace(c.FormValue("anada"))
	pf.Uvas = splitList(c.FormValue("uvas"))
	pf.Bodega = strings.TrimSpace(c.FormValue("bodega"))
	pf.Region = strings.TrimSpace(c.FormValue("region"))
	pf.Pais = strings.TrimSpace(c.FormValue("pais"))
	pf.Graduacion = strings.TrimSpace(c.FormValue("graduacion"))
	pf.Crianza = strings.TrimSpace(c.FormValue("crianza"))
	pf.NotasCata = strings.TrimSpace(c.FormValue("notas_cata"))
	pf.Maridaje = strings.TrimSpace(c.FormValue("maridaje"))
	pf.Descripcion = strings.TrimSpace(c.FormValue("descripcion"))
	pf.Ocasiones = splitList(c.FormValue("ocasiones"))
	pf.Tags = splitList(c.FormValue("tags"))
	pf.UpdatedAt = time.Now().UTC()
	if err := eng.SavePerfil(pf); err != nil {
		return c.HTML(http.StatusBadRequest, err.Error())
	}
	return c.Redirect(http.StatusFound, "/admin/productos?q="+cve)
}

// adminPerfilBootstrap runs the draft generator from the panel.
// POST /admin/perfiles/bootstrap
func (s *Server) adminPerfilBootstrap(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	created, updated, err := sommelier.BootstrapDrafts(s.Store)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, err.Error())
	}
	return c.HTML(http.StatusOK, fmt.Sprintf(`<p class="mb-4">Borradores generados: <b>%d</b> nuevos, <b>%d</b> actualizados.</p>
<p><a class="underline" href="/admin/productos">← Volver a productos</a></p>`, created, updated))
}
