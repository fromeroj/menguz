// Package models defines shared domain types synced from Aspel SAE 8.0
// and created by the platform (campaigns, orders, deals...).
package models

import "time"

// ---------- Producto (from SAE INVE01 + INVE_CLIB01) ----------

type Producto struct {
	CveArt       string    `json:"cve_art"`
	Nombre       string    `json:"nombre"`
	Descripcion  string    `json:"descripcion"`
	PrecioBase   float64   `json:"precio_base"`             // PRECIO1 (lista general)
	PrecioLista2 float64   `json:"precio_lista_2"`          // mayorista medio
	PrecioLista3 float64   `json:"precio_lista_3"`          // vinatería / B2B
	PrecioOferta float64   `json:"precio_oferta,omitempty"` // override from campaign
	Existencia   float64   `json:"existencia"`
	UnidadMedida string    `json:"unidad_medida"`
	Linea        string    `json:"linea"`     // SAE LIN_ART
	Categoria    string    `json:"categoria"` // frontend category (Vino Tinto...)
	Grupo        string    `json:"grupo"`     // frontend group (Vinos, Licores...)
	Graduacion   string    `json:"graduacion,omitempty"`
	Origen       string    `json:"origen,omitempty"`
	NotasCata    string    `json:"notas_cata,omitempty"`
	ImagenURL    string    `json:"imagen_url,omitempty"`
	Activo       bool      `json:"activo"`
	UltimaSync   time.Time `json:"ultima_sync"`
}

// PublicProductsJSON is the shape consumed by the storefront /data/products.json:
// n=name, p=price, c=category, g=group, i=image path.
type PublicProductJSON struct {
	N string  `json:"n"`
	P float64 `json:"p"`
	C string  `json:"c"`
	G string  `json:"g"`
	I string  `json:"i"`
}

// ---------- PerfilVino (sommelier enrichment, admin-curated) ----------
// SAE sync provides price/stock; the wine profile adds the sensorial and
// editorial layer the sommelier IA reasons over: grape, producer, region,
// tasting notes, pairing and occasion tags.

type PerfilVino struct {
	CveArt      string    `json:"cve_art"`     // SAE article key
	Tipo        string    `json:"tipo"`        // Tinto | Blanco | Rosado | Espumoso | Champagne | Licor | Otro
	Uvas        []string  `json:"uvas"`        // varietals, e.g. ["Malbec", "Cabernet Sauvignon"]
	Bodega      string    `json:"bodega"`      // producer / winery
	Region      string    `json:"region"`      // e.g. "Valle de Guadalupe", "Rioja"
	Pais        string    `json:"pais"`        // e.g. "México", "España", "Francia"
	Anada       string    `json:"anada"`       // vintage, e.g. "2021"; "" = sin añada (NV)
	Graduacion  string    `json:"graduacion"`  // e.g. "13.5%"
	Crianza     string    `json:"crianza"`     // e.g. "Roble 6 meses", "Reserva 24 meses"
	NotasCata   string    `json:"notas_cata"`  // tasting notes
	Maridaje    string    `json:"maridaje"`    // food pairing
	Descripcion string    `json:"descripcion"` // long editorial description
	Ocasiones   []string  `json:"ocasiones"`   // regalo, cena romántica, celebración, asado, verano, digestivo...
	Tags        []string  `json:"tags"`        // free-form: frutal, afrutado, seco, dulce, tánico, añejo, artesanal
	UpdatedAt   time.Time `json:"updated_at"`
}

// ---------- Cliente (from SAE CLIE01) ----------

type Cliente struct {
	ID          string    `json:"id"` // SAE CLAVE (trimmed)
	Nombre      string    `json:"nombre"`
	RFC         string    `json:"rfc"`
	Email       string    `json:"email"`
	Telefono    string    `json:"telefono"`
	Calle       string    `json:"calle"`
	Colonia     string    `json:"colonia"`
	Ciudad      string    `json:"ciudad"`
	Estado      string    `json:"estado"`
	CP          string    `json:"cp"`
	ListaPrecio int       `json:"lista_precio"` // 1..10, price list assigned in SAE
	LimiteCred  float64   `json:"limite_credito"`
	Saldo       float64   `json:"saldo"`
	B2BEnabled  bool      `json:"b2b_enabled"` // portal access enabled by admin
	B2BPassHash string    `json:"b2b_pass_hash,omitempty"`
	UltimaSync  time.Time `json:"ultima_sync"`
}

// ---------- Factura (from SAE FACT01/FACTD01, for CRM/BI) ----------

type Factura struct {
	ID        string           `json:"id"`
	ClienteID string           `json:"cliente_id"`
	Fecha     time.Time        `json:"fecha"`
	Total     float64          `json:"total"`
	Detalles  []FacturaDetalle `json:"detalles,omitempty"`
}

type FacturaDetalle struct {
	CveArt string  `json:"cve_art"`
	Cant   float64 `json:"cant"`
	Precio float64 `json:"precio"`
}

// ---------- Campaigns / promos (managed by admin, no dev needed) ----------

const (
	CampanaCarousel    = "carousel"     // banner in Promociones section
	CampanaProductoMes = "producto_mes" // Producto del Mes feature
	CampanaEspecial    = "especial"     // highlighted specials
)

type Campana struct {
	ID           string    `json:"id"`
	Titulo       string    `json:"titulo"`
	Tipo         string    `json:"tipo"` // carousel | producto_mes | especial
	ImagenURL    string    `json:"imagen_url"`
	Descripcion  string    `json:"descripcion"`
	CveArt       string    `json:"cve_art,omitempty"` // linked product
	PrecioOferta float64   `json:"precio_oferta,omitempty"`
	CtaTexto     string    `json:"cta_texto,omitempty"`
	CtaURL       string    `json:"cta_url,omitempty"`
	WAMensaje    string    `json:"wa_mensaje,omitempty"`
	Inicio       time.Time `json:"inicio"`
	Fin          time.Time `json:"fin"`
	Activa       bool      `json:"activa"`
	Orden        int       `json:"orden"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ---------- Cart & Orders ----------

type CartItem struct {
	CveArt   string  `json:"cve_art"`
	Cantidad float64 `json:"cantidad"`
}

type Cart struct {
	Token     string     `json:"token"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type OrderItem struct {
	CveArt   string  `json:"cve_art"`
	Nombre   string  `json:"nombre"`
	Cantidad float64 `json:"cantidad"`
	Precio   float64 `json:"precio"` // unit price at purchase
}

const (
	OrdenPendientePago = "pendiente_pago"
	OrdenPagado        = "pagado"
	OrdenPreparando    = "preparando"
	OrdenEnviado       = "enviado"
	OrdenEntregado     = "entregado"
	OrdenCancelado     = "cancelado"
)

type Orden struct {
	ID              string      `json:"id"`
	Folio           string      `json:"folio"`                // human friendly MGZ-000123
	Tipo            string      `json:"tipo"`                 // b2c | b2b
	ClienteID       string      `json:"cliente_id,omitempty"` // B2B or registered
	ClienteNombre   string      `json:"cliente_nombre"`
	ClienteEmail    string      `json:"cliente_email"`
	ClienteTelefono string      `json:"cliente_telefono"`
	Direccion       string      `json:"direccion,omitempty"`
	Items           []OrderItem `json:"items"`
	Subtotal        float64     `json:"subtotal"`
	Envio           float64     `json:"envio"`
	Total           float64     `json:"total"`
	Estado          string      `json:"estado"`
	ReferenciaPago  string      `json:"referencia_pago,omitempty"` // bank transfer reference
	PagoMetodo      string      `json:"pago_metodo,omitempty"`     // transferencia
	Notas           string      `json:"notas,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// ---------- Users (admin) ----------

type Rol string

const (
	RolAdmin  Rol = "admin"
	RolVentas Rol = "ventas"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Nombre       string    `json:"nombre"`
	Rol          Rol       `json:"rol"`
	PasswordHash string    `json:"password_hash,omitempty"` // stored server-side only; never exposed by API responses
	CreatedAt    time.Time `json:"created_at"`
}

// ---------- CRM ----------

type Deal struct {
	ID        string    `json:"id"`
	ClienteID string    `json:"cliente_id,omitempty"`
	Titulo    string    `json:"titulo"`
	Monto     float64   `json:"monto"`
	Etapa     string    `json:"etapa"` // nuevo, contactado, propuesta, ganado, perdido
	Notas     string    `json:"notas,omitempty"`
	Origen    string    `json:"origen,omitempty"` // web, whatsapp, telefono, b2b
	AsignadoA string    `json:"asignado_a,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Customer360 aggregates SAE data + web orders for the CRM view.
type Customer360 struct {
	Cliente        *Cliente   `json:"cliente"`
	TotalFacturado float64    `json:"total_facturado"`
	UltimaCompra   *time.Time `json:"ultima_compra,omitempty"`
	WebOrders      []*Orden   `json:"web_orders,omitempty"`
	Facturas       []Factura  `json:"facturas,omitempty"`
	Frecuencia     int        `json:"frecuencia_mes"` // avg purchases per month
	Segmentos      []string   `json:"segmentos,omitempty"`
}

// ---------- Chat ----------

type ChatMessage struct {
	Role      string    `json:"role"` // user | assistant | system
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type ChatSession struct {
	ID         string        `json:"id"`
	Canal      string        `json:"canal"` // web | whatsapp
	ClienteTel string        `json:"cliente_tel,omitempty"`
	Escalado   bool          `json:"escalado"`
	Messages   []ChatMessage `json:"messages"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// ---------- Sync log ----------

type SyncLog struct {
	ID        string    `json:"id"`
	Inicio    time.Time `json:"inicio"`
	Fin       time.Time `json:"fin"`
	Productos int       `json:"productos"`
	Clientes  int       `json:"clientes"`
	Facturas  int       `json:"facturas"`
	Origen    string    `json:"origen"` // firebird | mock
	Error     string    `json:"error,omitempty"`
}
