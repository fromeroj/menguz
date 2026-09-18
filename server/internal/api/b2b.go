package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"menguz/internal/badger"
	"menguz/internal/middleware"
	"menguz/internal/models"
)

// ---------- B2B portal (Month 3): SAE price lists, bulk orders, history ----------

// priceForList returns the unit price for a client according to its SAE price list.
func priceForList(p models.Producto, lista int) float64 {
	switch lista {
	case 2:
		if p.PrecioLista2 > 0 {
			return p.PrecioLista2
		}
	case 3:
		if p.PrecioLista3 > 0 {
			return p.PrecioLista3
		}
	}
	if p.PrecioOferta > 0 && p.PrecioOferta < p.PrecioBase {
		return p.PrecioOferta
	}
	return p.PrecioBase
}

func (s *Server) b2bLogin(c echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var found *models.Cliente
	var cli models.Cliente
	_ = s.Store.ScanPrefix(badger.PrefCliente, &cli, func(key string) bool {
		if strings.EqualFold(cli.Email, email) {
			cp := cli
			found = &cp
			return false
		}
		return true
	})
	if found == nil || !found.B2BEnabled || found.B2BPassHash == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "acceso B2B no habilitado para este correo")
	}
	if bcrypt.CompareHashAndPassword([]byte(found.B2BPassHash), []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "credenciales inválidas")
	}
	return s.respondWithTokens(c, found.ID, found.Email, "b2b", found.ListaPrecio)
}

func (s *Server) b2bMe(c echo.Context) error {
	cl := middleware.ClaimsOf(c)
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+cl.Sub, &cli); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "cliente no encontrado")
	}
	cli.B2BPassHash = ""
	return c.JSON(http.StatusOK, map[string]any{
		"cliente":      cli,
		"lista_precio": cl.Lista,
	})
}

// b2bCatalog returns the full catalog priced with the client's SAE list.
func (s *Server) b2bCatalog(c echo.Context) error {
	cl := middleware.ClaimsOf(c)
	q := strings.ToLower(strings.TrimSpace(c.QueryParam("q")))
	cat := c.QueryParam("categoria")
	type item struct {
		models.Producto
		PrecioCliente float64 `json:"precio_cliente"`
	}
	out := []item{}
	var p models.Producto
	err := s.Store.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		if !p.Activo {
			return true
		}
		if cat != "" && p.Categoria != cat {
			return true
		}
		if q != "" && !strings.Contains(strings.ToLower(p.Nombre), q) {
			return true
		}
		out = append(out, item{Producto: p, PrecioCliente: priceForList(p, cl.Lista)})
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out), "lista_precio": cl.Lista})
}

// b2bOrders lists the client's own web orders.
func (s *Server) b2bOrders(c echo.Context) error {
	cl := middleware.ClaimsOf(c)
	out := []models.Orden{}
	var o models.Orden
	err := s.Store.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		if o.ClienteID == cl.Sub {
			out = append(out, o)
		}
		return true
	})
	if err != nil {
		return err
	}
	sortOrdersDesc(out)
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

type b2bOrderReq struct {
	Items     []models.CartItem `json:"items" validate:"required,min=1"`
	Direccion string            `json:"direccion"`
	Notas     string            `json:"notas"`
}

// b2bCreateOrder places a bulk order priced with the client's SAE list.
// B2B clients pay on credit terms (per SAE) — order starts as "pagado" pending confirmation.
func (s *Server) b2bCreateOrder(c echo.Context) error {
	cl := middleware.ClaimsOf(c)
	var req b2bOrderReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	if len(req.Items) == 0 || len(req.Items) > 500 {
		return badRequest("entre 1 y 500 partidas por pedido")
	}
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+cl.Sub, &cli); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "cliente no encontrado")
	}
	items := []models.OrderItem{}
	subtotal := 0.0
	for _, it := range req.Items {
		var p models.Producto
		if err := s.Store.GetJSON(badger.PrefProducto+it.CveArt, &p); err != nil || !p.Activo {
			return badRequest("producto no disponible: " + it.CveArt)
		}
		if p.Existencia < it.Cantidad {
			return badRequest("existencia insuficiente: " + p.Nombre)
		}
		price := priceForList(p, cli.ListaPrecio)
		items = append(items, models.OrderItem{CveArt: it.CveArt, Nombre: p.Nombre, Cantidad: it.Cantidad, Precio: price})
		subtotal += price * it.Cantidad
	}
	if err := s.reserveStock(items); err != nil {
		return err
	}
	seq, err := s.Store.NextFolio()
	if err != nil {
		return err
	}
	orden := models.Orden{
		ID: newID("ord"), Folio: "MGZ-" + zeroPad(seq, 6), Tipo: "b2b",
		ClienteID: cli.ID, ClienteNombre: cli.Nombre, ClienteEmail: cli.Email, ClienteTelefono: cli.Telefono,
		Direccion: req.Direccion, Items: items,
		Subtotal: round2(subtotal), Total: round2(subtotal),
		Estado: models.OrdenPreparando, Notas: req.Notas,
		CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
	}
	if err := s.Store.PutJSON(badger.PrefOrden+orden.ID, orden); err != nil {
		return err
	}
	_ = s.Store.PutString(badger.PrefOrdenCliente+cli.ID+":"+orden.ID, "1")
	if s.Ana != nil {
		_ = s.Ana.Rebuild(s.Store)
	}
	return c.JSON(http.StatusCreated, orden)
}

// ---------- Admin: client management ----------

func (s *Server) adminListClients(c echo.Context) error {
	q := strings.ToLower(strings.TrimSpace(c.QueryParam("q")))
	out := []models.Cliente{}
	var cli models.Cliente
	err := s.Store.ScanPrefix(badger.PrefCliente, &cli, func(key string) bool {
		if q != "" && !strings.Contains(strings.ToLower(cli.Nombre), q) &&
			!strings.Contains(strings.ToLower(cli.Email), q) {
			return true
		}
		cp := cli
		cp.B2BPassHash = ""
		out = append(out, cp)
		return true
	})
	if err != nil {
		return err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nombre < out[j].Nombre })
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}

// adminSetB2B enables/disables portal access and sets the initial password.
func (s *Server) adminSetB2B(c echo.Context) error {
	id := c.Param("id")
	var body struct {
		Enabled  bool   `json:"enabled"`
		Password string `json:"password"`
	}
	if err := c.Bind(&body); err != nil {
		return badRequest("JSON inválido")
	}
	var cli models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+id, &cli); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "cliente no encontrado")
	}
	cli.B2BEnabled = body.Enabled
	if body.Password != "" {
		if len(body.Password) < 8 {
			return badRequest("password mínimo 8 caracteres")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), s.Cfg.BcryptCost)
		if err != nil {
			return err
		}
		cli.B2BPassHash = string(hash)
	}
	if err := s.Store.PutJSON(badger.PrefCliente+id, cli); err != nil {
		return err
	}
	cli.B2BPassHash = ""
	return c.JSON(http.StatusOK, cli)
}
