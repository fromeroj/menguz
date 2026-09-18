package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"menguz/internal/badger"
	"menguz/internal/models"
)

// ---------- cookie session for the admin panel ----------

func (s *Server) webClaims(c echo.Context) *claimsLite {
	ck, err := c.Cookie("mgz_token")
	if err != nil || ck.Value == "" {
		return nil
	}
	cl := &claimsLite{}
	_, err = jwt.ParseWithClaims(ck.Value, cl, func(t *jwt.Token) (any, error) {
		return []byte(s.Cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || cl.Sub == "" || (cl.Rol != "admin" && cl.Rol != "ventas") {
		return nil
	}
	return cl
}

type claimsLite struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Rol   string `json:"rol"`
	jwt.RegisteredClaims
}

// requireWeb redirects to the login page when the cookie session is missing.
func (s *Server) requireWeb(c echo.Context) *claimsLite {
	cl := s.webClaims(c)
	if cl == nil {
		_ = c.Redirect(http.StatusFound, "/admin/login")
		return nil
	}
	return cl
}

func csrfInput(c echo.Context) string {
	ck, err := c.Cookie("csrf")
	if err != nil {
		return ""
	}
	return fmt.Sprintf(`<input type="hidden" name="csrf" value="%s">`, esc(ck.Value))
}

// ---------- layout ----------

func layout(title, active, body string) string {
	nav := func(href, label string) string {
		cls := "px-3 py-2 rounded-lg text-sm font-medium "
		if active == href {
			cls += "bg-[#7B1F3A] text-white"
		} else {
			cls += "text-neutral-700 hover:bg-neutral-200"
		}
		return fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, href, cls, label)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s · Menguz Admin</title>
<script src="https://cdn.tailwindcss.com"></script>
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<style>body{font-family:ui-sans-serif,system-ui,sans-serif}</style>
</head>
<body class="bg-neutral-100 text-neutral-900">
<header class="bg-white border-b sticky top-0 z-40">
  <div class="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-4">
    <a href="/admin" class="text-lg font-bold text-[#7B1F3A]">Menguz<span class="text-neutral-400 font-normal"> admin</span></a>
    <nav class="flex flex-wrap gap-1">
      %s%s%s%s%s%s%s%s
    </nav>
    <form method="post" action="/admin/logout">%s<button class="text-sm text-neutral-500 hover:text-red-700">Salir</button></form>
  </div>
</header>
<main class="max-w-7xl mx-auto px-4 py-6">%s</main>
</body></html>`,
		esc(title),
		nav("/admin", "Panel"), nav("/admin/campanas", "Campañas"), nav("/admin/pedidos", "Pedidos"),
		nav("/admin/productos", "Productos"), nav("/admin/clientes", "Clientes"), nav("/admin/crm", "CRM"),
		nav("/admin/chat", "Chat"), nav("/admin/sync-logs", "Sync"),
		"", body)
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func money(v float64) string { return fmt.Sprintf("$%.2f", v) }

func dateFmt(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("02/01/2006 15:04")
}

// ---------- auth pages ----------

func (s *Server) adminLoginPage(c echo.Context) error {
	html := fmt.Sprintf(`<!DOCTYPE html><html lang="es"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Acceso · Menguz Admin</title>
<script src="https://cdn.tailwindcss.com"></script></head>
<body class="min-h-screen bg-[#2B1810] flex items-center justify-center p-4">
<form method="post" action="/admin/login" class="bg-white rounded-2xl shadow-xl p-8 w-full max-w-sm space-y-4">
  <h1 class="text-2xl font-bold text-[#7B1F3A]">Menguz Admin</h1>
  <p class="text-sm text-neutral-500">Panel de administración de campañas, pedidos y CRM.</p>
  <input name="email" type="email" required placeholder="correo" class="w-full border rounded-lg px-3 py-2">
  <input name="password" type="password" required placeholder="contraseña" class="w-full border rounded-lg px-3 py-2">
  <button class="w-full bg-[#7B1F3A] text-white rounded-lg py-2.5 font-semibold">Entrar</button>
</form></body></html>`)
	return c.HTML(http.StatusOK, html)
}

func (s *Server) adminLoginSubmit(c echo.Context) error {
	email := c.FormValue("email")
	pass := c.FormValue("password")
	id, err := s.Store.GetString(badger.PrefUserEmail + email)
	if err != nil {
		return c.HTML(http.StatusUnauthorized, loginError("credenciales inválidas"))
	}
	var u models.User
	if err := s.Store.GetJSON(badger.PrefUser+id, &u); err != nil ||
		bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pass)) != nil {
		return c.HTML(http.StatusUnauthorized, loginError("credenciales inválidas"))
	}
	access, _ := s.issueAccess(u.ID, u.Email, string(u.Rol), 0)
	refresh, _ := s.issueRefresh(u.ID)
	setAuthCookies(c, access, refresh, s.Cfg)
	return c.Redirect(http.StatusFound, "/admin")
}

func loginError(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html lang="es"><head><meta charset="utf-8"><title>Acceso</title>
<script src="https://cdn.tailwindcss.com"></script></head><body class="min-h-screen bg-[#2B1810] flex items-center justify-center">
<div class="bg-white rounded-2xl p-8 max-w-sm text-center space-y-4">
<p class="text-red-700">%s</p><a href="/admin/login" class="text-[#7B1F3A] underline">Reintentar</a>
</div></body></html>`, esc(msg))
}

func (s *Server) adminLogoutSubmit(c echo.Context) error {
	if ck, err := c.Cookie("mgz_refresh"); err == nil {
		_ = s.Store.Delete(badger.PrefRefresh + ck.Value)
	}
	c.SetCookie(&http.Cookie{Name: "mgz_token", Value: "", Path: "/", MaxAge: -1})
	c.SetCookie(&http.Cookie{Name: "mgz_refresh", Value: "", Path: "/", MaxAge: -1})
	return c.Redirect(http.StatusFound, "/admin/login")
}

// ---------- dashboard ----------

func (s *Server) adminHome(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	count := func(prefix string) int {
		n := 0
		_ = s.Store.ScanPrefix(prefix, &struct{}{}, func(k string) bool { n++; return true })
		return n
	}
	var lastSync string
	lastSync, _ = s.Store.GetString(badger.PrefMeta + badger.MetaLastSync)

	var pendientes, hoy int
	today := time.Now().Format("2006-01-02")
	var o models.Orden
	_ = s.Store.ScanPrefix(badger.PrefOrden, &o, func(k string) bool {
		if o.Estado == models.OrdenPendientePago {
			pendientes++
		}
		if o.CreatedAt.Format("2006-01-02") == today {
			hoy++
		}
		return true
	})

	stat := func(label string, v int, href string) string {
		return fmt.Sprintf(`<a href="%s" class="bg-white rounded-2xl shadow p-5 block hover:shadow-md">
			<p class="text-3xl font-bold text-[#7B1F3A]">%d</p><p class="text-sm text-neutral-500 mt-1">%s</p></a>`, href, v, label)
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-1">Panel</h1>
<p class="text-sm text-neutral-500 mb-6">Última sync SAE: %s · <a class="underline" href="/admin/sync-logs">ver bitácora</a></p>
<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
%s%s%s%s
</div>
<div class="mt-8 bg-white rounded-2xl shadow p-5">
  <h2 class="font-semibold mb-3">Acciones rápidas</h2>
  <div class="flex flex-wrap gap-3">
    <a href="/admin/campanas/nueva" class="bg-[#7B1F3A] text-white px-4 py-2 rounded-lg text-sm">＋ Nueva campaña</a>
    <a href="/admin/pedidos" class="border px-4 py-2 rounded-lg text-sm">Ver pedidos pendientes</a>
    <form method="post" action="/admin/sync">%s<button class="border px-4 py-2 rounded-lg text-sm">↻ Sync SAE ahora</button></form>
  </div>
</div>`,
		esc(lastSync),
		stat("Productos en catálogo", count(badger.PrefProducto), "/admin/productos"),
		stat("Clientes SAE", count(badger.PrefCliente), "/admin/clientes"),
		stat("Pedidos pendientes de pago", pendientes, "/admin/pedidos?estado=pendiente_pago"),
		stat("Pedidos de hoy", hoy, "/admin/pedidos"),
		csrfInput(c))
	return c.HTML(http.StatusOK, layout("Panel", "/admin", body))
}

// ---------- campaigns pages ----------

func (s *Server) adminCampaignsPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	rows := []string{}
	var camp models.Campana
	_ = s.Store.ScanPrefix(badger.PrefCampana, &camp, func(k string) bool {
		estado := `<span class="text-neutral-400">inactiva</span>`
		if camp.Activa {
			if nowUTC().Before(camp.Inicio) {
				estado = `<span class="text-amber-600">programada</span>`
			} else if nowUTC().After(camp.Fin) {
				estado = `<span class="text-neutral-400">vencida</span>`
			} else {
				estado = `<span class="text-green-700 font-semibold">vigente</span>`
			}
		}
		rows = append(rows, fmt.Sprintf(`<tr class="border-t">
			<td class="px-4 py-3 font-medium">%s</td>
			<td class="px-4 py-3"><span class="text-xs bg-neutral-200 rounded-full px-2 py-0.5">%s</span></td>
			<td class="px-4 py-3 text-sm">%s → %s</td>
			<td class="px-4 py-3 text-sm">%s</td>
			<td class="px-4 py-3 text-sm whitespace-nowrap">
				<a class="underline mr-2" href="/admin/campanas/%s">Editar</a>
				<form method="post" action="/admin/campanas/%s/toggle" class="inline">%s<button class="underline mr-2">%s</button></form>
				<form method="post" action="/admin/campanas/%s/eliminar" class="inline" onsubmit="return confirm('¿Eliminar campaña?')">%s<button class="underline text-red-700">Eliminar</button></form>
			</td></tr>`,
			esc(camp.Titulo), esc(camp.Tipo), dateFmt(camp.Inicio), dateFmt(camp.Fin), estado,
			camp.ID, camp.ID, csrfInput(c), map[bool]string{true: "Desactivar", false: "Activar"}[camp.Activa],
			camp.ID, csrfInput(c)))
		return true
	})
	body := fmt.Sprintf(`
<div class="flex items-center justify-between mb-4">
  <h1 class="text-2xl font-bold">Campañas y promociones</h1>
  <a href="/admin/campanas/nueva" class="bg-[#7B1F3A] text-white px-4 py-2 rounded-lg text-sm">＋ Nueva campaña</a>
</div>
<div class="bg-white rounded-2xl shadow overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500"><th class="px-4 py-3">Título</th><th class="px-4 py-3">Tipo</th><th class="px-4 py-3">Vigencia</th><th class="px-4 py-3">Estado</th><th class="px-4 py-3">Acciones</th></tr></thead>
<tbody>%s</tbody></table>
</div>
<p class="text-xs text-neutral-500 mt-3">Tipos: <b>carousel</b> (banners de promociones), <b>producto_mes</b> (destacado de portada), <b>especial</b>. Las campañas vigentes se publican automáticamente en el sitio sin tocar código.</p>`,
		strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Campañas", "/admin/campanas", body))
}

func (s *Server) adminCampaignForm(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	id := c.Param("id")
	camp := models.Campana{
		Tipo: models.CampanaCarousel, Activa: true,
		Inicio: time.Now(), Fin: time.Now().AddDate(0, 1, 0), Orden: 10,
	}
	titulo := "Nueva campaña"
	if id != "" {
		if err := s.Store.GetJSON(badger.PrefCampana+id, &camp); err != nil {
			return c.Redirect(http.StatusFound, "/admin/campanas")
		}
		titulo = "Editar campaña"
	}
	sel := func(t string) string {
		if camp.Tipo == t {
			return "selected"
		}
		return ""
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-4">%s</h1>
<form method="post" action="/admin/campanas/guardar" class="bg-white rounded-2xl shadow p-6 max-w-2xl space-y-4">
%s
<input type="hidden" name="id" value="%s">
<div><label class="text-sm font-medium">Título *</label>
  <input name="titulo" required value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
<div><label class="text-sm font-medium">Tipo</label>
  <select name="tipo" class="mt-1 w-full border rounded-lg px-3 py-2">
    <option value="carousel" %s>Carrusel de promociones</option>
    <option value="producto_mes" %s>Producto del mes</option>
    <option value="especial" %s>Especial</option>
  </select></div>
<div><label class="text-sm font-medium">URL de imagen (banner)</label>
  <input name="imagen_url" value="%s" placeholder="/img/promos/mi-banner.jpg" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
<div><label class="text-sm font-medium">Descripción</label>
  <textarea name="descripcion" rows="3" class="mt-1 w-full border rounded-lg px-3 py-2">%s</textarea></div>
<div class="grid grid-cols-2 gap-4">
  <div><label class="text-sm font-medium">Clave de producto ligado (opcional)</label>
    <input name="cve_art" value="%s" placeholder="TEQ-ANE-001" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Precio oferta</label>
    <input name="precio_oferta" type="number" step="0.01" value="%v" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div class="grid grid-cols-2 gap-4">
  <div><label class="text-sm font-medium">Inicio *</label>
    <input name="inicio" type="date" required value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Fin *</label>
    <input name="fin" type="date" required value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
</div>
<div class="grid grid-cols-3 gap-4 items-end">
  <div><label class="text-sm font-medium">Texto del botón</label>
    <input name="cta_texto" value="%s" placeholder="Cotiza ahora" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div><label class="text-sm font-medium">Orden</label>
    <input name="orden" type="number" value="%d" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
  <div class="flex items-center gap-2 pb-2">
    <input type="checkbox" name="activa" value="1" %s id="activa"><label for="activa" class="text-sm">Activa</label></div>
</div>
<div><label class="text-sm font-medium">Mensaje de WhatsApp (cotización)</label>
  <input name="wa_mensaje" value="%s" class="mt-1 w-full border rounded-lg px-3 py-2"></div>
<div class="flex gap-3 pt-2">
  <button class="bg-[#7B1F3A] text-white px-6 py-2.5 rounded-lg font-semibold">Guardar</button>
  <a href="/admin/campanas" class="border px-6 py-2.5 rounded-lg">Cancelar</a>
</div>
</form>`,
		titulo, csrfInput(c), id,
		esc(camp.Titulo), sel("carousel"), sel("producto_mes"), sel("especial"),
		esc(camp.ImagenURL), esc(camp.Descripcion), esc(camp.CveArt), camp.PrecioOferta,
		camp.Inicio.Format("2006-01-02"), camp.Fin.Format("2006-01-02"),
		esc(camp.CtaTexto), camp.Orden, checked(camp.Activa), esc(camp.WAMensaje))
	return c.HTML(http.StatusOK, layout(titulo, "/admin/campanas", body))
}

func checked(b bool) string {
	if b {
		return "checked"
	}
	return ""
}

func (s *Server) adminCampaignSave(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	inicio, err1 := time.Parse("2006-01-02", c.FormValue("inicio"))
	fin, err2 := time.Parse("2006-01-02", c.FormValue("fin"))
	if err1 != nil || err2 != nil || !fin.After(inicio) {
		return c.HTML(http.StatusBadRequest, "Fechas inválidas: fin debe ser posterior a inicio.")
	}
	precio := 0.0
	fmt.Sscanf(c.FormValue("precio_oferta"), "%f", &precio)
	orden := 0
	fmt.Sscanf(c.FormValue("orden"), "%d", &orden)
	id := c.FormValue("id")
	camp := models.Campana{Activa: c.FormValue("activa") == "1", CreatedAt: nowUTC()}
	if id != "" {
		if err := s.Store.GetJSON(badger.PrefCampana+id, &camp); err != nil {
			return c.Redirect(http.StatusFound, "/admin/campanas")
		}
	} else {
		camp.ID = newID("camp")
	}
	camp.Titulo = c.FormValue("titulo")
	camp.Tipo = c.FormValue("tipo")
	camp.ImagenURL = c.FormValue("imagen_url")
	camp.Descripcion = c.FormValue("descripcion")
	camp.CveArt = strings.TrimSpace(c.FormValue("cve_art"))
	camp.PrecioOferta = precio
	camp.CtaTexto = c.FormValue("cta_texto")
	camp.WAMensaje = c.FormValue("wa_mensaje")
	camp.Inicio, camp.Fin = inicio, fin
	camp.Orden = orden
	camp.UpdatedAt = nowUTC()
	if err := s.Store.PutJSON(badger.PrefCampana+camp.ID, camp); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/admin/campanas")
}

func (s *Server) adminCampaignToggle(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	var camp models.Campana
	if err := s.Store.GetJSON(badger.PrefCampana+c.Param("id"), &camp); err == nil {
		camp.Activa = !camp.Activa
		camp.UpdatedAt = nowUTC()
		_ = s.Store.PutJSON(badger.PrefCampana+camp.ID, camp)
	}
	return c.Redirect(http.StatusFound, "/admin/campanas")
}

func (s *Server) adminCampaignDelete(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	_ = s.Store.Delete(badger.PrefCampana + c.Param("id"))
	return c.Redirect(http.StatusFound, "/admin/campanas")
}
