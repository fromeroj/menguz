package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"menguz/internal/badger"
	"menguz/internal/models"
	"menguz/internal/sae"
)

// ---------- orders ----------

func badgeEstado(estado string) string {
	colors := map[string]string{
		models.OrdenPendientePago: "bg-amber-100 text-amber-800",
		models.OrdenPagado:        "bg-blue-100 text-blue-800",
		models.OrdenPreparando:    "bg-indigo-100 text-indigo-800",
		models.OrdenEnviado:       "bg-purple-100 text-purple-800",
		models.OrdenEntregado:     "bg-green-100 text-green-800",
		models.OrdenCancelado:     "bg-red-100 text-red-800",
	}
	cls := colors[estado]
	if cls == "" {
		cls = "bg-neutral-100 text-neutral-700"
	}
	return fmt.Sprintf(`<span class="text-xs rounded-full px-2.5 py-1 %s">%s</span>`, cls, esc(estado))
}

func (s *Server) adminOrdersPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	estado := c.QueryParam("estado")
	ordenes := []models.Orden{}
	var o models.Orden
	_ = s.Store.ScanPrefix(badger.PrefOrden, &o, func(k string) bool {
		if estado != "" && o.Estado != estado {
			return true
		}
		ordenes = append(ordenes, o)
		return true
	})
	sortOrdersDesc(ordenes)
	rows := []string{}
	for _, o := range ordenes {
		rows = append(rows, fmt.Sprintf(`<tr class="border-t hover:bg-neutral-50">
			<td class="px-4 py-3"><a class="underline font-medium" href="/admin/pedidos/%s">%s</a></td>
			<td class="px-4 py-3 text-sm">%s</td>
			<td class="px-4 py-3 text-sm">%s</td>
			<td class="px-4 py-3 text-sm">%s</td>
			<td class="px-4 py-3 text-sm text-right">%s</td>
			<td class="px-4 py-3">%s</td></tr>`,
			o.ID, esc(o.Folio), dateFmt(o.CreatedAt), esc(o.ClienteNombre), esc(o.Tipo), money(o.Total), badgeEstado(o.Estado)))
	}
	if len(rows) == 0 {
		rows = append(rows, `<tr><td colspan="6" class="px-4 py-10 text-center text-neutral-400">Sin pedidos todavía.</td></tr>`)
	}
	filtros := []string{`<a class="text-sm underline" href="/admin/pedidos">todos</a>`}
	for _, e := range []string{models.OrdenPendientePago, models.OrdenPagado, models.OrdenPreparando, models.OrdenEnviado, models.OrdenEntregado, models.OrdenCancelado} {
		filtros = append(filtros, fmt.Sprintf(`<a class="text-sm underline" href="/admin/pedidos?estado=%s">%s</a>`, e, e))
	}
	body := fmt.Sprintf(`
<div class="flex items-center justify-between mb-4">
  <h1 class="text-2xl font-bold">Pedidos</h1>
  <div class="flex gap-3 flex-wrap">%s</div>
</div>
<div class="bg-white rounded-2xl shadow overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500">
<th class="px-4 py-3">Folio</th><th class="px-4 py-3">Fecha</th><th class="px-4 py-3">Cliente</th><th class="px-4 py-3">Tipo</th><th class="px-4 py-3 text-right">Total</th><th class="px-4 py-3">Estado</th></tr></thead>
<tbody>%s</tbody></table></div>`,
		strings.Join(filtros, " · "), strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Pedidos", "/admin/pedidos", body))
}

func (s *Server) adminOrderDetail(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	var o models.Orden
	if err := s.Store.GetJSON(badger.PrefOrden+c.Param("id"), &o); err != nil {
		return c.Redirect(http.StatusFound, "/admin/pedidos")
	}
	items := []string{}
	for _, it := range o.Items {
		items = append(items, fmt.Sprintf(`<tr class="border-t">
			<td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td>
			<td class="px-4 py-2 text-sm text-right">%.0f</td><td class="px-4 py-2 text-sm text-right">%s</td>
			<td class="px-4 py-2 text-sm text-right font-medium">%s</td></tr>`,
			esc(it.CveArt), esc(it.Nombre), it.Cantidad, money(it.Precio), money(it.Precio*it.Cantidad)))
	}
	opts := []string{}
	for _, e := range []string{models.OrdenPendientePago, models.OrdenPagado, models.OrdenPreparando, models.OrdenEnviado, models.OrdenEntregado, models.OrdenCancelado} {
		sel := ""
		if o.Estado == e {
			sel = "selected"
		}
		opts = append(opts, fmt.Sprintf(`<option value="%s" %s>%s</option>`, e, sel, e))
	}
	pago := ""
	if o.ReferenciaPago != "" {
		pago = fmt.Sprintf(`<div class="bg-amber-50 border border-amber-200 rounded-xl p-4 text-sm">
			<p class="font-semibold">Referencia de pago: %s</p>
			<p>Transferencia a %s · CLABE %s · a nombre de %s</p>
			<p class="text-neutral-500 mt-1">Confirma la recepción del pago y cambia el estado a «pagado».</p></div>`,
			esc(o.ReferenciaPago), esc(s.Cfg.BankName), esc(s.Cfg.BankCLABE), esc(s.Cfg.BankHolder))
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-1">Pedido %s %s</h1>
<p class="text-sm text-neutral-500 mb-4">%s · %s · %s · tel %s</p>
%s
<div class="bg-white rounded-2xl shadow mt-4 overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500">
<th class="px-4 py-3">Clave</th><th class="px-4 py-3">Producto</th><th class="px-4 py-3 text-right">Cant</th><th class="px-4 py-3 text-right">Precio</th><th class="px-4 py-3 text-right">Importe</th></tr></thead>
<tbody>%s
<tr class="border-t font-semibold"><td colspan="4" class="px-4 py-3 text-right">Total</td><td class="px-4 py-3 text-right">%s</td></tr>
</tbody></table></div>
<form method="post" action="/admin/pedidos/%s/estado" class="mt-4 flex items-center gap-3">
%s
<select name="estado" class="border rounded-lg px-3 py-2">%s</select>
<button class="bg-[#7B1F3A] text-white px-4 py-2 rounded-lg text-sm">Cambiar estado</button>
<a href="/admin/pedidos" class="text-sm underline ml-2">← Volver</a></form>`,
		esc(o.Folio), badgeEstado(o.Estado), dateFmt(o.CreatedAt), esc(o.ClienteNombre), esc(o.ClienteEmail), esc(o.ClienteTelefono),
		pago, strings.Join(items, "\n"), money(o.Total), o.ID, csrfInput(c), strings.Join(opts, ""))
	return c.HTML(http.StatusOK, layout("Pedido "+o.Folio, "/admin/pedidos", body))
}

func (s *Server) adminOrderStatus(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	id := c.Param("id")
	var o models.Orden
	if err := s.Store.GetJSON(badger.PrefOrden+id, &o); err == nil {
		nuevo := c.FormValue("estado")
		valid := map[string]bool{models.OrdenPendientePago: true, models.OrdenPagado: true, models.OrdenPreparando: true, models.OrdenEnviado: true, models.OrdenEntregado: true, models.OrdenCancelado: true}
		if valid[nuevo] {
			if nuevo == models.OrdenCancelado && o.Estado != models.OrdenCancelado {
				_ = s.releaseStock(o.Items)
			}
			o.Estado = nuevo
			o.UpdatedAt = nowUTC()
			_ = s.Store.PutJSON(badger.PrefOrden+id, o)
			if s.Ana != nil {
				_ = s.Ana.Rebuild(s.Store)
			}
		}
	}
	return c.Redirect(http.StatusFound, "/admin/pedidos/"+id)
}

// ---------- products ----------

func (s *Server) adminProductsPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	q := strings.ToLower(c.QueryParam("q"))
	prods := []models.Producto{}
	var p models.Producto
	_ = s.Store.ScanPrefix(badger.PrefProducto, &p, func(k string) bool {
		if q != "" && !strings.Contains(strings.ToLower(p.Nombre), q) && !strings.Contains(strings.ToLower(p.CveArt), q) {
			return true
		}
		prods = append(prods, p)
		return true
	})
	sort.Slice(prods, func(i, j int) bool { return prods[i].CveArt < prods[j].CveArt })
	rows := []string{}
	for _, p := range prods {
		stock := `<span class="text-green-700">%.0f</span>`
		if p.Existencia <= 0 {
			stock = `<span class="text-red-700 font-semibold">agotado</span>`
		} else if p.Existencia < 10 {
			stock = `<span class="text-amber-600">%.0f ⚠</span>`
		}
		rows = append(rows, fmt.Sprintf(`<tr class="border-t">
			<td class="px-4 py-2 text-sm font-mono">%s</td><td class="px-4 py-2 text-sm">%s</td>
			<td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td>
			<td class="px-4 py-2 text-sm text-right">%s</td><td class="px-4 py-2 text-sm text-right">%s</td>
			<td class="px-4 py-2 text-sm text-right">`+stock+`</td>
			<td class="px-4 py-2 text-sm"><a class="underline text-[#7B1F3A]" href="/admin/productos/%s/perfil">ficha</a></td></tr>`,
			esc(p.CveArt), esc(p.Nombre), esc(p.Categoria), esc(p.Linea),
			money(p.PrecioBase), money(p.PrecioLista3), p.Existencia, esc(p.CveArt)))
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-4">Productos <span class="text-sm font-normal text-neutral-400">(%d, sincronizados de SAE)</span></h1>
<form class="mb-4 flex gap-3 items-center" method="get"><input name="q" value="%s" placeholder="buscar por nombre o clave…" class="border rounded-lg px-3 py-2 w-72">
<button class="border rounded-lg px-3 py-2 text-sm" formmethod="post" formaction="/admin/perfiles/bootstrap" title="Genera fichas de sommelier (tipo, uvas detectadas) para productos sin ficha — no pisa ediciones manuales">⚡ generar fichas borrador</button></form>
<div class="bg-white rounded-2xl shadow overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500">
<th class="px-4 py-3">Clave</th><th class="px-4 py-3">Producto</th><th class="px-4 py-3">Categoría</th><th class="px-4 py-3">Línea SAE</th><th class="px-4 py-3 text-right">P. lista 1</th><th class="px-4 py-3 text-right">P. lista 3 (B2B)</th><th class="px-4 py-3 text-right">Existencia</th><th class="px-4 py-3">Sommelier</th></tr></thead>
<tbody>%s</tbody></table></div>
<p class="text-xs text-neutral-500 mt-3">Los precios y existencias se actualizan con la sync nocturna de SAE (03:00). Para modificarlos, edita en Aspel SAE. La <b>ficha</b> alimenta al Sommelier IA (uvas, notas, maridaje) y se conserva entre syncs.</p>`,
		len(prods), esc(c.QueryParam("q")), strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Productos", "/admin/productos", body))
}

// ---------- clients + B2B ----------

func (s *Server) adminClientsPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	q := strings.ToLower(c.QueryParam("q"))
	rows := []string{}
	var cli models.Cliente
	_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(k string) bool {
		if q != "" && !strings.Contains(strings.ToLower(cli.Nombre), q) && !strings.Contains(strings.ToLower(cli.Email), q) {
			return true
		}
		st := s.statsFor(cli.ID)
		b2b := `<span class="text-xs bg-neutral-100 rounded-full px-2 py-1">sin acceso</span>`
		if cli.B2BEnabled {
			b2b = `<span class="text-xs bg-green-100 text-green-800 rounded-full px-2 py-1">portal activo</span>`
		}
		rows = append(rows, fmt.Sprintf(`<tr class="border-t align-top">
			<td class="px-4 py-3"><p class="font-medium">%s</p><p class="text-xs text-neutral-400">%s · %s</p></td>
			<td class="px-4 py-3 text-sm">lista %d<br/>saldo %s</td>
			<td class="px-4 py-3 text-sm text-right">%s<br/><span class="text-xs text-neutral-400">facturado histórico</span></td>
			<td class="px-4 py-3">%s<br/><span class="text-xs text-neutral-400">%d pedidos web</span></td>
			<td class="px-4 py-3 text-sm">
				<form method="post" action="/admin/clientes/%s/b2b" class="flex gap-2 items-center">
				%s
				<input type="hidden" name="enabled" value="%s">
				<input name="password" type="text" placeholder="nueva contraseña (opcional)" class="border rounded px-2 py-1 text-xs w-44">
				<button class="underline text-xs">%s</button></form>
			</td></tr>`,
			esc(cli.Nombre), esc(cli.ID), esc(cli.Email),
			cli.ListaPrecio, money(cli.Saldo), money(st.TotalFacturado),
			b2b, len(st.WebOrders),
			cli.ID, csrfInput(c),
			map[bool]string{true: "0", false: "1"}[cli.B2BEnabled],
			map[bool]string{true: "Deshabilitar B2B", false: "Habilitar B2B"}[cli.B2BEnabled]))
		return true
	})
	if len(rows) == 0 {
		rows = append(rows, `<tr><td colspan="5" class="px-4 py-10 text-center text-neutral-400">Sin clientes. Ejecuta la sync SAE.</td></tr>`)
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-4">Clientes <span class="text-sm font-normal text-neutral-400">(desde CLIE01 de SAE)</span></h1>
<form class="mb-4" method="get"><input name="q" value="%s" placeholder="buscar…" class="border rounded-lg px-3 py-2 w-72"></form>
<div class="bg-white rounded-2xl shadow overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500">
<th class="px-4 py-3">Cliente</th><th class="px-4 py-3">Lista precios / Saldo</th><th class="px-4 py-3 text-right">Historial</th><th class="px-4 py-3">Portal B2B</th><th class="px-4 py-3">Acceso</th></tr></thead>
<tbody>%s</tbody></table></div>
<p class="text-xs text-neutral-500 mt-3">Al habilitar el portal, el cliente entra en <code>/api/b2b/login</code> con su correo y ve sus precios de lista SAE (1=menudeo, 2=medio mayorista, 3=vinatería).</p>`,
		esc(c.QueryParam("q")), strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Clientes", "/admin/clientes", body))
}

func (s *Server) adminClientB2B(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	id := c.Param("id")
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+id, &cli); err == nil {
		cli.B2BEnabled = c.FormValue("enabled") == "1"
		if pw := c.FormValue("password"); pw != "" {
			if len(pw) >= 8 {
				if hash, err := bcrypt.GenerateFromPassword([]byte(pw), s.Cfg.BcryptCost); err == nil {
					cli.B2BPassHash = string(hash)
				}
			}
		}
		_ = s.Store.PutJSON(badger.PrefCliente+id, cli)
	}
	return c.Redirect(http.StatusFound, "/admin/clientes")
}

// ---------- CRM page ----------

func (s *Server) adminCRMPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	// pipeline board
	byStage := map[string][]models.Deal{}
	var d models.Deal
	_ = s.Store.ScanPrefix(badger.PrefDeal, &d, func(k string) bool {
		byStage[d.Etapa] = append(byStage[d.Etapa], d)
		return true
	})
	stages := []string{"nuevo", "contactado", "propuesta", "ganado", "perdido"}
	cols := []string{}
	for _, st := range stages {
		cards := []string{}
		total := 0.0
		for _, deal := range byStage[st] {
			total += deal.Monto
			cards = append(cards, fmt.Sprintf(`<div class="bg-white rounded-xl shadow p-3 mb-2 text-sm">
				<p class="font-medium">%s</p><p class="text-xs text-neutral-500">%s · %s</p>
				<div class="flex justify-between items-center mt-2">
					<span class="font-semibold text-[#7B1F3A]">%s</span>
					<form method="post" action="/admin/crm/deals/%s/eliminar" onsubmit="return confirm('¿Eliminar?')">%s<button class="text-xs text-red-700 underline">eliminar</button></form>
				</div></div>`,
				esc(deal.Titulo), esc(deal.Origen), esc(deal.ClienteID), money(deal.Monto), deal.ID, csrfInput(c)))
		}
		cols = append(cols, fmt.Sprintf(`<div class="min-w-52">
			<p class="text-sm font-semibold mb-2 capitalize">%s <span class="text-neutral-400 font-normal">· %d · %s</span></p>%s</div>`,
			st, len(byStage[st]), money(total), strings.Join(cards, "")))
	}

	// top customers
	top := []string{}
	var cli models.Cliente
	type ranked struct {
		cli models.Cliente
		tot float64
		ult *time.Time
	}
	var all []ranked
	_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(k string) bool {
		st := s.statsFor(cli.ID)
		all = append(all, ranked{cli: cli, tot: st.TotalFacturado, ult: st.UltimaCompra})
		return true
	})
	sort.Slice(all, func(i, j int) bool { return all[i].tot > all[j].tot })
	for i, r := range all {
		if i >= 10 {
			break
		}
		ult := "—"
		if r.ult != nil {
			ult = r.ult.Format("02/01/2006")
		}
		top = append(top, fmt.Sprintf(`<tr class="border-t">
			<td class="px-4 py-2 text-sm"><a class="underline" href="/admin/crm/clientes/%s">%s</a></td>
			<td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm text-right">%s</td>
			<td class="px-4 py-2 text-sm text-right">%s</td></tr>`,
			r.cli.ID, esc(r.cli.Nombre), esc(r.cli.Email), money(r.tot), ult))
	}

	// segments
	segs := []string{}
	for id, def := range segmentDefsOrdered() {
		n := 0
		var c2 models.Cliente
		_ = s.Store.ScanPrefix(badger.PrefCliente, &c2, func(k string) bool {
			st := s.statsFor(c2.ID)
			for _, sg := range s.segmentsFor(&c2, &st) {
				if sg == id {
					n++
				}
			}
			return true
		})
		segs = append(segs, fmt.Sprintf(`<li class="flex justify-between text-sm py-1.5 border-b last:border-0">
			<span><b>%s</b> — <span class="text-neutral-500">%s</span></span>
			<a class="underline" href="/api/admin/crm/segments/%s/members" target="_blank">%d →</a></li>`, id, def, id, n))
	}

	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-4">CRM</h1>
<div class="grid md:grid-cols-3 gap-6">
  <div class="md:col-span-2">
    <div class="flex justify-between items-center mb-2">
      <h2 class="font-semibold">Pipeline de ventas</h2>
      <form method="post" action="/admin/crm/deals/guardar" class="flex gap-2 items-center flex-wrap">
        %s
        <input name="titulo" required placeholder="Título de la oportunidad" class="border rounded-lg px-2 py-1.5 text-sm w-52">
        <input name="monto" type="number" step="0.01" placeholder="Monto" class="border rounded-lg px-2 py-1.5 text-sm w-28">
        <input name="cliente_id" placeholder="Cliente ID (opc.)" class="border rounded-lg px-2 py-1.5 text-sm w-36">
        <select name="etapa" class="border rounded-lg px-2 py-1.5 text-sm">
          <option>nuevo</option><option>contactado</option><option>propuesta</option><option>ganado</option><option>perdido</option>
        </select>
        <button class="bg-[#7B1F3A] text-white px-3 py-1.5 rounded-lg text-sm">＋ Agregar</button>
      </form>
    </div>
    <div class="flex gap-3 overflow-x-auto bg-neutral-200/60 rounded-2xl p-3">%s</div>
    <h2 class="font-semibold mt-6 mb-2">Top clientes por facturación</h2>
    <div class="bg-white rounded-2xl shadow overflow-x-auto">
      <table class="w-full text-left"><thead><tr class="text-xs uppercase text-neutral-500">
      <th class="px-4 py-2">Cliente</th><th class="px-4 py-2">Email</th><th class="px-4 py-2 text-right">Facturado</th><th class="px-4 py-2 text-right">Última compra</th>
      </tr></thead><tbody>%s</tbody></table>
    </div>
  </div>
  <div>
    <h2 class="font-semibold mb-2">Segmentación</h2>
    <div class="bg-white rounded-2xl shadow p-4"><ul>%s</ul></div>
    <div class="bg-white rounded-2xl shadow p-4 mt-4 text-sm space-y-1">
      <p class="font-semibold mb-1">Exportar</p>
      <a class="underline block" href="/api/admin/crm/export/customers">clientes.csv</a>
      <a class="underline block" href="/api/admin/crm/export/orders">pedidos.csv</a>
      <a class="underline block" href="/api/admin/crm/export/deals">pipeline.csv</a>
    </div>
  </div>
</div>`,
		csrfInput(c), strings.Join(cols, ""), strings.Join(top, "\n"), strings.Join(segs, "\n"))
	return c.HTML(http.StatusOK, layout("CRM", "/admin/crm", body))
}

func segmentDefsOrdered() map[string]string {
	// deterministic small map iteration for stable display
	return map[string]string{
		"alto_valor":     segmentDefText("alto_valor"),
		"sin_compra_90d": segmentDefText("sin_compra_90d"),
		"frecuente":      segmentDefText("frecuente"),
		"con_saldo":      segmentDefText("con_saldo"),
		"b2b_activo":     segmentDefText("b2b_activo"),
	}
}

func segmentDefText(id string) string {
	defs := map[string]string{
		"alto_valor":     "más de $25k facturados",
		"sin_compra_90d": "sin compra en 90 días",
		"frecuente":      "2+ compras/mes",
		"con_saldo":      "saldo pendiente",
		"b2b_activo":     "portal B2B activo",
	}
	return defs[id]
}

func (s *Server) adminCustomer360(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	id := c.Param("id")
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+id, &cli); err != nil {
		return c.Redirect(http.StatusFound, "/admin/crm")
	}
	st := s.statsFor(id)
	segs := s.segmentsFor(&cli, &st)
	segHTML := ""
	for _, sg := range segs {
		segHTML += fmt.Sprintf(`<span class="text-xs bg-[#7B1F3A]/10 text-[#7B1F3A] rounded-full px-2.5 py-1 mr-1">%s</span>`, sg)
	}
	facs := []string{}
	var f models.Factura
	_ = s.Store.ScanPrefix(badger.PrefFactura, &f, func(k string) bool {
		if f.ClienteID == id {
			facs = append(facs, fmt.Sprintf(`<tr class="border-t"><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm text-right">%s</td></tr>`,
				esc(f.ID), f.Fecha.Format("02/01/2006"), money(f.Total)))
		}
		return true
	})
	sort.Slice(facs, func(i, j int) bool { return facs[i] > facs[j] })
	if len(facs) > 15 {
		facs = facs[:15]
	}
	web := []string{}
	for _, o := range st.WebOrders {
		web = append(web, fmt.Sprintf(`<tr class="border-t"><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm text-right">%s</td></tr>`,
			esc(o.Folio), dateFmt(o.CreatedAt), esc(o.Estado), money(o.Total)))
	}
	ult := "—"
	if st.UltimaCompra != nil {
		ult = st.UltimaCompra.Format("02/01/2006")
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold">%s</h1>
<p class="text-sm text-neutral-500 mb-1">%s · RFC %s · lista de precios %d · tel %s</p>
<p class="mb-4">%s</p>
<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
  <div class="bg-white rounded-2xl shadow p-4"><p class="text-2xl font-bold text-[#7B1F3A]">%s</p><p class="text-xs text-neutral-500">facturado histórico</p></div>
  <div class="bg-white rounded-2xl shadow p-4"><p class="text-2xl font-bold text-[#7B1F3A]">%s</p><p class="text-xs text-neutral-500">última compra</p></div>
  <div class="bg-white rounded-2xl shadow p-4"><p class="text-2xl font-bold text-[#7B1F3A]">%d/mes</p><p class="text-xs text-neutral-500">frecuencia</p></div>
  <div class="bg-white rounded-2xl shadow p-4"><p class="text-2xl font-bold text-[#7B1F3A]">%s</p><p class="text-xs text-neutral-500">saldo</p></div>
</div>
<div class="grid md:grid-cols-2 gap-6">
  <div class="bg-white rounded-2xl shadow overflow-x-auto">
    <p class="font-semibold px-4 pt-3">Facturas recientes (SAE)</p>
    <table class="w-full text-left mt-2"><thead><tr class="text-xs uppercase text-neutral-500">
    <th class="px-4 py-2">Folio</th><th class="px-4 py-2">Fecha</th><th class="px-4 py-2 text-right">Total</th></tr></thead>
    <tbody>%s</tbody></table>
  </div>
  <div class="bg-white rounded-2xl shadow overflow-x-auto">
    <p class="font-semibold px-4 pt-3">Pedidos web</p>
    <table class="w-full text-left mt-2"><thead><tr class="text-xs uppercase text-neutral-500">
    <th class="px-4 py-2">Folio</th><th class="px-4 py-2">Fecha</th><th class="px-4 py-2">Estado</th><th class="px-4 py-2 text-right">Total</th></tr></thead>
    <tbody>%s</tbody></table>
  </div>
</div>
<p class="mt-4"><a class="underline text-sm" href="/admin/crm">← Volver al CRM</a></p>`,
		esc(cli.Nombre), esc(cli.Email), esc(cli.RFC), cli.ListaPrecio, esc(cli.Telefono),
		segHTML, money(st.TotalFacturado), ult, st.Frecuencia, money(cli.Saldo),
		strings.Join(facs, "\n"), strings.Join(web, "\n"))
	return c.HTML(http.StatusOK, layout("Cliente "+cli.Nombre, "/admin/crm", body))
}

func (s *Server) adminDealSave(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	monto := 0.0
	fmt.Sscanf(c.FormValue("monto"), "%f", &monto)
	d := models.Deal{
		ID: newID("deal"), ClienteID: c.FormValue("cliente_id"),
		Titulo: c.FormValue("titulo"), Monto: monto, Etapa: c.FormValue("etapa"),
		CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
	}
	if d.Etapa == "" {
		d.Etapa = "nuevo"
	}
	_ = s.Store.PutJSON(badger.PrefDeal+d.ID, d)
	return c.Redirect(http.StatusFound, "/admin/crm")
}

func (s *Server) adminDealDelete(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	_ = s.Store.Delete(badger.PrefDeal + c.Param("id"))
	return c.Redirect(http.StatusFound, "/admin/crm")
}

// ---------- chat review ----------

func (s *Server) adminChatPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	sess := []models.ChatSession{}
	var cs models.ChatSession
	_ = s.Store.ScanPrefix(badger.PrefChatSess, &cs, func(k string) bool {
		sess = append(sess, cs)
		return true
	})
	for i := 1; i < len(sess); i++ {
		for j := i; j > 0 && sess[j].UpdatedAt.After(sess[j-1].UpdatedAt); j-- {
			sess[j], sess[j-1] = sess[j-1], sess[j]
		}
	}
	rows := []string{}
	for _, cs := range sess {
		if len(cs.Messages) == 0 {
			continue
		}
		last := cs.Messages[len(cs.Messages)-1]
		preview := last.Content
		if len(preview) > 90 {
			preview = preview[:90] + "…"
		}
		escal := ""
		if cs.Escalado {
			escal = `<span class="text-xs bg-red-100 text-red-800 rounded-full px-2 py-0.5 ml-1">escalado</span>`
		}
		rows = append(rows, fmt.Sprintf(`<details class="border-t"><summary class="px-4 py-3 cursor-pointer text-sm">
			<b>%s</b> %s <span class="text-neutral-400">· %s · %d mensajes</span><span class="block text-neutral-500 text-xs mt-0.5">%s</span></summary>
			<div class="px-6 pb-4 space-y-2">%s</div></details>`,
			esc(cs.Canal), escal, dateFmt(cs.UpdatedAt), len(cs.Messages), esc(preview), chatTranscript(cs.Messages)))
	}
	if len(rows) == 0 {
		rows = append(rows, `<p class="px-4 py-10 text-center text-neutral-400 text-sm">Aún no hay conversaciones.</p>`)
	}
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-4">Conversaciones del chatbot</h1>
<div class="bg-white rounded-2xl shadow">%s</div>
<p class="text-xs text-neutral-500 mt-3">El chatbot responde con GPT-4o + catálogo (RAG) cuando OPENAI_API_KEY está configurado; si no, usa respuestas deterministas. Las escalaciones llegan a WhatsApp.</p>`,
		strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Chat", "/admin/chat", body))
}

func chatTranscript(msgs []models.ChatMessage) string {
	out := []string{}
	for _, m := range msgs {
		cls := "bg-neutral-100"
		who := "Cliente"
		if m.Role == "assistant" {
			cls = "bg-[#7B1F3A]/10"
			who = "Bot"
		}
		out = append(out, fmt.Sprintf(`<div class="%s rounded-xl px-3 py-2 text-sm"><b class="text-xs uppercase text-neutral-500">%s</b><br>%s</div>`,
			cls, who, esc(m.Content)))
	}
	return strings.Join(out, "")
}

// ---------- sync page ----------

func (s *Server) adminSyncNow(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	go func() {
		_, _ = sae.Sync(s.Cfg, s.Store, s.Ana)
	}()
	return c.Redirect(http.StatusFound, "/admin/sync-logs")
}

func (s *Server) adminSyncLogsPage(c echo.Context) error {
	if s.requireWeb(c) == nil {
		return nil
	}
	logs := []models.SyncLog{}
	var lg models.SyncLog
	_ = s.Store.ScanPrefix(badger.PrefSyncLog, &lg, func(k string) bool {
		logs = append(logs, lg)
		return true
	})
	for i := 1; i < len(logs); i++ {
		for j := i; j > 0 && logs[j].Inicio.After(logs[j-1].Inicio); j-- {
			logs[j], logs[j-1] = logs[j-1], logs[j]
		}
	}
	rows := []string{}
	for _, lg := range logs {
		status := `<span class="text-green-700">ok</span>`
		if lg.Error != "" {
			status = fmt.Sprintf(`<span class="text-red-700" title="%s">error</span>`, esc(lg.Error))
		}
		rows = append(rows, fmt.Sprintf(`<tr class="border-t">
			<td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td>
			<td class="px-4 py-2 text-sm text-right">%d</td><td class="px-4 py-2 text-sm text-right">%d</td>
			<td class="px-4 py-2 text-sm text-right">%d</td><td class="px-4 py-2 text-sm">%s</td><td class="px-4 py-2 text-sm">%s</td></tr>`,
			dateFmt(lg.Inicio), esc(lg.Origen), lg.Productos, lg.Clientes, lg.Facturas, status,
			lg.Fin.Sub(lg.Inicio).Round(time.Second).String()))
	}
	if len(rows) == 0 {
		rows = append(rows, `<tr><td colspan="7" class="px-4 py-10 text-center text-neutral-400">Sin sincronizaciones aún.</td></tr>`)
	}
	cfg := fmt.Sprintf("Modo: <b>%s</b> · programada diariamente a las <b>%s</b>", map[bool]string{true: "MOCK (datos demo)", false: "Firebird SAE"}[s.Cfg.SAEMock], esc(s.Cfg.SyncAt))
	body := fmt.Sprintf(`
<h1 class="text-2xl font-bold mb-1">Sincronización SAE</h1>
<p class="text-sm text-neutral-500 mb-4">%s</p>
<form method="post" action="/admin/sync">%s<button class="bg-[#7B1F3A] text-white px-4 py-2 rounded-lg text-sm">↻ Ejecutar ahora</button></form>
<div class="bg-white rounded-2xl shadow mt-4 overflow-x-auto">
<table class="w-full text-left">
<thead><tr class="text-xs uppercase text-neutral-500">
<th class="px-4 py-3">Inicio</th><th class="px-4 py-3">Origen</th><th class="px-4 py-3 text-right">Productos</th><th class="px-4 py-3 text-right">Clientes</th><th class="px-4 py-3 text-right">Facturas</th><th class="px-4 py-3">Estado</th><th class="px-4 py-3">Duración</th></tr></thead>
<tbody>%s</tbody></table></div>`,
		cfg, csrfInput(c), strings.Join(rows, "\n"))
	return c.HTML(http.StatusOK, layout("Sync", "/admin/sync-logs", body))
}
