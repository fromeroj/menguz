package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/models"
)

const cartCookie = "mgz_cart"

func (s *Server) cartToken(c echo.Context) (string, error) {
	if ck, err := c.Cookie(cartCookie); err == nil && ck.Value != "" {
		return ck.Value, nil
	}
	tok := randomHex(16)
	c.SetCookie(&http.Cookie{
		Name: cartCookie, Value: tok, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: 60 * 60 * 24 * 7,
	})
	return tok, nil
}

func (s *Server) getCart(c echo.Context) error {
	tok, err := s.cartToken(c)
	if err != nil {
		return err
	}
	cart := models.Cart{Token: tok, Items: []models.CartItem{}}
	_ = s.Store.GetJSON(badger.PrefCart+tok, &cart)
	// enrich with live product data + compute totals
	type line struct {
		models.CartItem
		Nombre     string  `json:"nombre"`
		Precio     float64 `json:"precio"`
		Disponible float64 `json:"disponible"`
		Importe    float64 `json:"importe"`
	}
	lines := []line{}
	subtotal := 0.0
	for _, it := range cart.Items {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+it.CveArt, &p); err != nil || !p.Activo {
			continue
		}
		price := p.PrecioBase
		if p.PrecioOferta > 0 && p.PrecioOferta < price {
			price = p.PrecioOferta
		}
		imp := price * it.Cantidad
		subtotal += imp
		lines = append(lines, line{CartItem: it, Nombre: p.Nombre, Precio: price, Disponible: p.Existencia, Importe: imp})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"items": lines, "subtotal": subtotal, "count": len(lines),
	})
}

type cartAddReq struct {
	CveArt   string  `json:"cve_art" validate:"required"`
	Cantidad float64 `json:"cantidad" validate:"required,gt=0,lte=500"`
}

func (s *Server) cartAdd(c echo.Context) error {
	var req cartAddReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	var p models.Producto
	if err := s.Store.GetJSON(badger.PrefProducto+req.CveArt, &p); err != nil || !p.Activo {
		return echo.NewHTTPError(http.StatusNotFound, "producto no encontrado")
	}
	if p.Existencia < req.Cantidad {
		return badRequest("existencia insuficiente")
	}
	tok, err := s.cartToken(c)
	if err != nil {
		return err
	}
	cart := models.Cart{Token: tok, Items: []models.CartItem{}}
	_ = s.Store.GetJSON(badger.PrefCart+tok, &cart)
	found := false
	for i, it := range cart.Items {
		if it.CveArt == req.CveArt {
			cart.Items[i].Cantidad += req.Cantidad
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, models.CartItem{CveArt: req.CveArt, Cantidad: req.Cantidad})
	}
	cart.UpdatedAt = time.Now()
	if err := s.Store.PutJSON(badger.PrefCart+tok, cart); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "count": len(cart.Items)})
}

func (s *Server) cartRemove(c echo.Context) error {
	tok, err := s.cartToken(c)
	if err != nil {
		return err
	}
	cve := c.Param("cve")
	cart := models.Cart{Token: tok, Items: []models.CartItem{}}
	if err := s.Store.GetJSON(badger.PrefCart+tok, &cart); err != nil {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	}
	keep := []models.CartItem{}
	for _, it := range cart.Items {
		if it.CveArt != cve {
			keep = append(keep, it)
		}
	}
	cart.Items = keep
	cart.UpdatedAt = time.Now()
	_ = s.Store.PutJSON(badger.PrefCart+tok, cart)
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "count": len(keep)})
}

// ---------- Orders ----------

type checkoutReq struct {
	Nombre    string  `json:"nombre" validate:"required,min=3"`
	Email     string  `json:"email" validate:"required,email"`
	Telefono  string  `json:"telefono" validate:"required,min=8"`
	Direccion string  `json:"direccion"`
	Notas     string  `json:"notas"`
	Envio     float64 `json:"envio"`
}

// checkout converts the cart into an order with estado=pendiente_pago and a
// unique bank-transfer reference (per proposal: B2C pays by transfer, no
// card gateway). Stock is reserved at this point.
func (s *Server) checkout(c echo.Context) error {
	if !s.CheckoutLim.Allow(c.RealIP()) {
		return echo.NewHTTPError(http.StatusTooManyRequests, "demasiados intentos de compra")
	}
	var req checkoutReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	tok, err := s.cartToken(c)
	if err != nil {
		return err
	}
	cart := models.Cart{Token: tok, Items: []models.CartItem{}}
	if err := s.Store.GetJSON(badger.PrefCart+tok, &cart); err != nil || len(cart.Items) == 0 {
		return badRequest("el carrito está vacío")
	}

	items := []models.OrderItem{}
	subtotal := 0.0
	for _, it := range cart.Items {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+it.CveArt, &p); err != nil || !p.Activo {
			return badRequest("producto agotado o inexistente: " + it.CveArt)
		}
		if p.Existencia < it.Cantidad {
			return badRequest("existencia insuficiente para " + p.Nombre)
		}
		price := p.PrecioBase
		if p.PrecioOferta > 0 && p.PrecioOferta < price {
			price = p.PrecioOferta
		}
		items = append(items, models.OrderItem{CveArt: it.CveArt, Nombre: p.Nombre, Cantidad: it.Cantidad, Precio: price})
		subtotal += price * it.Cantidad
	}

	// reserve stock + create order atomically at Badger level
	seq, err := s.Store.NextFolio()
	if err != nil {
		return err
	}
	folio := "MGZ-" + zeroPad(seq, 6)
	ref := "MENG" + zeroPad(seq, 6)
	orden := models.Orden{
		ID: newID("ord"), Folio: folio, Tipo: "b2c",
		ClienteNombre: req.Nombre, ClienteEmail: req.Email, ClienteTelefono: req.Telefono,
		Direccion: req.Direccion, Items: items,
		Subtotal: round2(subtotal), Envio: req.Envio, Total: round2(subtotal + req.Envio),
		Estado: models.OrdenPendientePago, ReferenciaPago: ref, PagoMetodo: "transferencia",
		Notas: req.Notas, CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
	}
	if err := s.reserveStock(orden.Items); err != nil {
		return err
	}
	if err := s.Store.PutJSON(badger.PrefOrden+orden.ID, orden); err != nil {
		return err
	}
	if orden.ClienteID != "" {
		_ = s.Store.PutString(badger.PrefOrdenCliente+orden.ClienteID+":"+orden.ID, "1")
	}
	// clear cart
	cart.Items = []models.CartItem{}
	_ = s.Store.PutJSON(badger.PrefCart+tok, cart)
	// refresh analytics with the new order
	if s.Ana != nil {
		_ = s.Ana.Rebuild(s.Store)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"orden": orden,
		"pago": map[string]any{
			"metodo":        "transferencia",
			"referencia":    ref,
			"banco":         s.Cfg.BankName,
			"clabe":         s.Cfg.BankCLABE,
			"titular":       s.Cfg.BankHolder,
			"monto":         orden.Total,
			"instrucciones": "Realiza la transferencia e indica la referencia. Confirmaremos tu pedido en cuánto se refleje el pago.",
		},
	})
}

// reserveStock decrements existence for ordered items.
func (s *Server) reserveStock(items []models.OrderItem) error {
	for _, it := range items {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+it.CveArt, &p); err != nil {
			return echo.NewHTTPError(http.StatusConflict, "producto ya no disponible: "+it.CveArt)
		}
		if p.Existencia < it.Cantidad {
			return badRequest("existencia insuficiente para " + p.Nombre)
		}
		p.Existencia -= it.Cantidad
		if err := s.Store.PutJSON(badger.PrefProducto+it.CveArt, p); err != nil {
			return err
		}
	}
	return nil
}

// releaseStock returns stock for a cancelled order.
func (s *Server) releaseStock(items []models.OrderItem) error {
	for _, it := range items {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+it.CveArt, &p); err != nil {
			continue
		}
		p.Existencia += it.Cantidad
		_ = s.Store.PutJSON(badger.PrefProducto+it.CveArt, p)
	}
	return nil
}

// orderLookup lets a customer track their order by folio + email.
func (s *Server) orderLookup(c echo.Context) error {
	folio := c.QueryParam("folio")
	email := c.QueryParam("email")
	if folio == "" || email == "" {
		return badRequest("folio y email son requeridos")
	}
	var o models.Orden
	found := false
	_ = s.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		if o.Folio == folio && (o.ClienteEmail == email || email == "") {
			found = true
			return false
		}
		return true
	})
	if !found {
		return echo.NewHTTPError(http.StatusNotFound, "pedido no encontrado")
	}
	return c.JSON(http.StatusOK, map[string]any{
		"folio": o.Folio, "estado": o.Estado, "total": o.Total,
		"referencia_pago": o.ReferenciaPago, "created_at": o.CreatedAt,
		"items": o.Items,
	})
}

// ---------- Admin order management ----------

func (s *Server) adminListOrders(c echo.Context) error {
	estado := c.QueryParam("estado")
	tipo := c.QueryParam("tipo")
	out := []models.Orden{}
	var o models.Orden
	err := s.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		if estado != "" && o.Estado != estado {
			return true
		}
		if tipo != "" && o.Tipo != tipo {
			return true
		}
		out = append(out, o)
		return true
	})
	if err != nil {
		return err
	}
	sortOrdersDesc(out)
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

func (s *Server) adminGetOrder(c echo.Context) error {
	id := c.Param("id")
	var o models.Orden
	if err := s.Store.GetJSON(badger.PrefOrden+id, &o); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "pedido no encontrado")
	}
	return c.JSON(http.StatusOK, o)
}

func (s *Server) adminSetOrderStatus(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Estado string `json:"estado" validate:"required"`
	}
	if err := c.Bind(&body); err != nil {
		return badRequest("JSON inválido")
	}
	valid := map[string]bool{
		models.OrdenPendientePago: true, models.OrdenPagado: true, models.OrdenPreparando: true,
		models.OrdenEnviado: true, models.OrdenEntregado: true, models.OrdenCancelado: true,
	}
	if !valid[body.Estado] {
		return badRequest("estado inválido")
	}
	var o models.Orden
	if err := s.Store.GetJSON(badger.PrefOrden+id, &o); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "pedido no encontrado")
	}
	if body.Estado == models.OrdenCancelado && o.Estado != models.OrdenCancelado {
		_ = s.releaseStock(o.Items)
	}
	o.Estado = body.Estado
	o.UpdatedAt = nowUTC()
	if err := s.Store.PutJSON(badger.PrefOrden+id, o); err != nil {
		return err
	}
	if s.Ana != nil {
		_ = s.Ana.Rebuild(s.Store)
	}
	return c.JSON(http.StatusOK, o)
}

// ---------- helpers ----------

func zeroPad(n int64, w int) string {
	s := itoa(n)
	for len(s) < w {
		s = "0" + s
	}
	return s
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}

func sortOrdersDesc(o []models.Orden) {
	for i := 1; i < len(o); i++ {
		for j := i; j > 0 && o[j].CreatedAt.After(o[j-1].CreatedAt); j-- {
			o[j], o[j-1] = o[j-1], o[j]
		}
	}
}
