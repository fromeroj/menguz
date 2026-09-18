package sae

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"time"

	"menguz/internal/models"
)

// MockSource generates a realistic demo catalog (wines, tequilas, mezcals,
// whiskies, mixers...) plus clients and 12 months of sales history, so the
// whole platform (storefront, B2B, CRM, chatbot) can run before the real
// Firebird credentials are available. Enabled with -sae-mock=true (default)
// or when -sae-host is empty.
//
// If MOCK_CATALOG (or the default path) points to a products.json export in
// the storefront shape [{n,p,c,g,i}], the mock sync uses those real products
// (names, prices, categories and photos) instead of the generated catalog.
type MockSource struct {
	rnd *rand.Rand
}

func NewMockSource() *MockSource {
	return &MockSource{rnd: rand.New(rand.NewSource(42))}
}

func (m *MockSource) Close() error { return nil }

// fromCatalogFile loads a storefront products.json export (n,p,c,g,i) and
// converts it to product rows with stable claves and B2B price lists.
func (m *MockSource) fromCatalogFile() ([]ProductoRow, error) {
	path := os.Getenv("MOCK_CATALOG")
	if path == "" {
		path = "../design/app/public/data/products.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []struct {
		N string  `json:"n"`
		P float64 `json:"p"`
		C string  `json:"c"`
		G string  `json:"g"`
		I string  `json:"i"`
	}
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	rows := make([]ProductoRow, 0, len(items))
	for _, it := range items {
		if it.N == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(it.N))
		cve := fmt.Sprintf("MOCK-%08X", h.Sum32())
		p1 := it.P
		var p2, p3 float64
		if p1 > 1 { // p<=1 means "a cotizar" (quote only) in the storefront
			p2 = round2(p1 * 0.95)
			p3 = round2(p1 * 0.90)
		}
		linea := reverseLinea(it.C, it.G)
		rows = append(rows, ProductoRow{
			CveArt: cve, Nombre: it.N, Descr: it.C,
			Linea: linea, Unidad: "PZ", Exist: float64(3 + m.rnd.Intn(48)),
			Precio1: p1, Precio2: p2, Precio3: p3,
			Graduacion: "", Origen: "", NotasCata: "",
			Estatus: "A", Imagen: it.I,
		})
	}
	return rows, nil
}

// reverseLinea approximates a SAE line code from the storefront category.
func reverseLinea(cat, grupo string) string {
	for linea, m := range lineaMap {
		if m[0] == cat {
			return linea
		}
	}
	switch grupo {
	case "Vinos":
		return "VIN-TIN"
	case "Mezcladores":
		return "MEZ-REF"
	case "Abarrotes":
		return "ABA-GRAL"
	default:
		return "LIC-CRE"
	}
}

func (m *MockSource) Productos() ([]ProductoRow, error) {
	if rows, err := m.fromCatalogFile(); err == nil && len(rows) > 0 {
		return rows, nil
	}
	return m.generatedProductos()
}

var mockCatalog = []struct {
	cve, nombre, linea, um, grad, origen, notas string
	precio                                      float64
}{
	{"VIN-TIN-001", "Burgundy Pinot Noir 750ml", "VIN-TIN", "BOT-750ML", "13%", "Borgoña, Francia", "Cereza, vainilla, taninos suaves", 229.50},
	{"VIN-TIN-002", "Rioja Crianza Tempranillo 750ml", "VIN-TIN", "BOT-750ML", "13.5%", "La Rioja, España", "Frutos rojos, roble americano", 299.70},
	{"VIN-TIN-003", "La Katina Malbec 750ml", "VIN-TIN", "BOT-750ML", "14%", "Mendoza, Argentina", "Mora, pimienta negra, final largo", 259.70},
	{"VIN-TIN-004", "Chianti Classico DOCG 750ml", "VIN-TIN", "BOT-750ML", "12.5%", "Toscana, Italia", "Cereza ácida, hierbas secas", 415.00},
	{"VIN-TIN-005", "Cabernet Sauvignon Valle de Guadalupe 750ml", "VIN-TIN", "BOT-750ML", "13.8%", "Baja California, México", "Cassis, tabaco, chocolate amargo", 385.00},
	{"VIN-BLA-001", "Goose Berry Sauvignon Blanc 750ml", "VIN-BLA", "BOT-750ML", "12.5%", "Valle Central, Chile", "Maracuyá, hierba fresca, cítricos", 299.70},
	{"VIN-BLA-002", "Chardonnay Reserva 750ml", "VIN-BLA", "BOT-750ML", "13%", "Napa Valley, EE.UU.", "Mantequilla, manzana asada, roble", 520.00},
	{"VIN-BLA-003", "Albariño Rías Baixas 750ml", "VIN-BLA", "BOT-750ML", "12.5%", "Galicia, España", "Melón, salinidad, flor blanca", 445.00},
	{"VIN-ROS-001", "Provence Rosé 750ml", "VIN-ROS", "BOT-750ML", "12%", "Provenza, Francia", "Fresa, pétalos, final mineral", 390.00},
	{"VIN-ESP-001", "Cava Brut Nature 750ml", "VIN-ESP", "BOT-750ML", "11.5%", "Penedès, España", "Manzana verde, pan tostado", 465.00},
	{"ESP-CAVA-001", "Champagne Brut Premier 750ml", "ESP-CAVA", "BOT-750ML", "12%", "Champagne, Francia", "Brioche, cítricos, burbuja fina", 1450.00},
	{"TEQ-BLA-001", "Tequila Blanco Don Julio 750ml", "TEQ-BLA", "BOT-750ML", "38%", "Jalisco, México", "Agave cocido, cítricos, pimienta", 950.00},
	{"TEQ-REP-001", "Tequila Reposado Herradura 750ml", "TEQ-REP", "BOT-750ML", "40%", "Jalisco, México", "Agave, vainilla, roble suave", 780.00},
	{"TEQ-ANE-001", "Tequila Añejo Don Julio 750ml", "TEQ-ANE", "BOT-750ML", "38%", "Jalisco, México", "Caramelo, roble, agave asado", 1250.00},
	{"TEQ-BLA-002", "Tequila Blanco Olmeca 750ml", "TEQ-BLA", "BOT-750ML", "38%", "Jalisco, México", "Agave fresco, hierba limón", 295.00},
	{"MEZ-JOV-001", "Mezcal Joven Espadín 750ml", "MEZ-JOV", "BOT-750ML", "45%", "Oaxaca, México", "Humo, agave, cítricos quemados", 650.00},
	{"MEZ-JOV-002", "Mezcal Tobalá Joven 750ml", "MEZ-JOV", "BOT-750ML", "47%", "Oaxaca, México", "Mineral, tropical, ahumado elegante", 1150.00},
	{"MEZ-ANE-001", "Mezcal Añejo Reposado en Barrica 750ml", "MEZ-ANE", "BOT-750ML", "44%", "Guerrero, México", "Chocolate, cuero, humo dulce", 1390.00},
	{"WHI-SCOT-001", "Whisky Escocés 12 Años 750ml", "WHI-SCOT", "BOT-750ML", "40%", "Speyside, Escocia", "Miel, roble, frutos secos", 1150.00},
	{"WHI-SCOT-002", "Whisky Islay Single Malt 750ml", "WHI-SCOT", "BOT-750ML", "43%", "Islay, Escocia", "Turba, mar, humo medicinal", 1650.00},
	{"WHI-BOURB-001", "Bourbon Kentucky Straight 750ml", "WHI-BOURB", "BOT-750ML", "45%", "Kentucky, EE.UU.", "Vainilla, caramelo, maíz tostado", 890.00},
	{"RON-BLA-001", "Ron Blanco Carta Blanca 750ml", "RON-BLA", "BOT-750ML", "40%", "Puerto Rico", "Caña fresca, vainilla blanca", 320.00},
	{"RON-ANE-001", "Ron Añejo 7 Años 750ml", "RON-ANE", "BOT-750ML", "40%", "Guatemala", "Caramelo, higo, roble tostado", 620.00},
	{"VOD-PRE-001", "Vodka Premium 750ml", "VOD-PRE", "BOT-750ML", "40%", "Suecia", "Neutral, cremoso, final limpio", 480.00},
	{"GIN-LON-001", "Ginebra London Dry 750ml", "GIN-LON", "BOT-750ML", "41%", "Inglaterra", "Enebro, cáscara de limón, angélica", 560.00},
	{"BRA-JOV-001", "Brandy Torres 10 750ml", "BRA-JOV", "BOT-750ML", "38%", "Cataluña, España", "Uva, roble americano, avellana", 540.00},
	{"CON-VS-001", "Cognac VSOP 700ml", "CON-VS", "BOT-700ML", "40%", "Cognac, Francia", "Floral, durazno, roble fino", 1980.00},
	{"LIC-CRE-001", "Licor de Crema Irlandesa 750ml", "LIC-CRE", "BOT-750ML", "17%", "Irlanda", "Café, crema, cacao", 430.00},
	{"LIC-CRE-002", "Licor Naranja Triple Sec 750ml", "LIC-CRE", "BOT-750ML", "30%", "Francia", "Naranja amarga, azúcar, cáscara", 380.00},
	{"ROM-TRAD-001", "Rompope Artesanal 750ml", "ROM-TRAD", "BOT-750ML", "10%", "Puebla, México", "Vainilla, canela, yema", 265.00},
	{"APER-BIT-001", "Aperitivo Bitter 750ml", "APER-BIT", "BOT-750ML", "11%", "Italia", "Hierbas amargas, naranja", 410.00},
	{"CER-ART-001", "Cerveza Artesanal IPA 355ml", "CER-ART", "LAT-355ML", "6.5%", "CDMX, México", "Lúpulo tropical, amargor medio", 85.00},
	{"CER-ART-002", "Cerveza Artesanal Stout 355ml", "CER-ART", "LAT-355ML", "5.8%", "CDMX, México", "Café, cacao, cuerpo sedoso", 90.00},
	{"MEZ-REF-001", "Refresco Tónica 600ml", "MEZ-REF", "PET-600ML", "0%", "México", "Quinina cítrica, burbuja fina", 38.00},
	{"MEZ-JAR-001", "Jarabe de Agave 500ml", "MEZ-JAR", "PET-500ML", "0%", "Jalisco, México", "Dulzor de agave, neutral", 95.00},
	{"MEZ-ENE-001", "Bebida Energizante 473ml", "MEZ-ENE", "LAT-473ML", "0%", "México", "Cítricos, cafeína", 52.00},
	{"MEZ-AGU-001", "Agua Mineral 600ml", "MEZ-AGU", "PET-600ML", "0%", "México", "Mineral, sin gas", 25.00},
	{"MEZ-JUG-001", "Jugo de Arándano 1L", "MEZ-JUG", "TET-1L", "0%", "México", "Arándano, uva", 64.00},
	{"ABA-GRAL-001", "Aceitunas Rellenas 200g", "ABA-GRAL", "ENV-200G", "0%", "España", "Almendra, anchoa", 145.00},
	{"ABA-QUE-001", "Queso Manchego Curado 250g", "ABA-QUE", "PZ-250G", "0%", "La Mancha, España", "Nuez, salino, textura firme", 310.00},
}

var mockClientes = []ClienteRow{
	{Clave: "C0001", Nombre: "Bar La Cava del Dragón", RFC: "BCD120305KX2", Calle: "Av. de los Insurgentes 940", Colonia: "Narvarte", Ciudad: "CDMX", Estado: "CDMX", CP: "03020", Telefono: "55 7098 3876", Email: "compras@lacava.mx", ListaPrec: 3, LimiteCred: 50000, Saldo: 12800},
	{Clave: "C0002", Nombre: "Vinatería El Corcho", RFC: "VEC150811QA4", Calle: "Calz. del Hueso 340", Colonia: "Coapa", Ciudad: "CDMX", Estado: "CDMX", CP: "04980", Telefono: "55 7098 3913", Email: "pedidos@elcorcho.mx", ListaPrec: 3, LimiteCred: 35000, Saldo: 5400},
	{Clave: "C0003", Nombre: "Restaurante Casa Dragones", RFC: "RCD170922MN7", Calle: "Miguel Ángel de Quevedo 128", Colonia: "Roma Norte", Ciudad: "CDMX", Estado: "CDMX", CP: "06700", Telefono: "55 3483 6109", Email: "compras@casadragones.mx", ListaPrec: 2, LimiteCred: 80000, Saldo: 22100},
	{Clave: "C0004", Nombre: "Hotel Boutique Reforma 27", RFC: "HBR190415PL9", Calle: "Reforma 27", Colonia: "Centro", Ciudad: "CDMX", Estado: "CDMX", CP: "06000", Telefono: "55 5530 1284", Email: "almacen@reforma27.mx", ListaPrec: 2, LimiteCred: 120000, Saldo: 41800},
	{Clave: "C0005", Nombre: "Cantina La Última y Nos Vamos", RFC: "CLU140728KR1", Calle: "Av. Cuauhtémoc 810", Colonia: "Doctores", Ciudad: "CDMX", Estado: "CDMX", CP: "06720", Telefono: "55 5762 8834", Email: "encargado@laultima.mx", ListaPrec: 3, LimiteCred: 25000, Saldo: 1900},
}

func (m *MockSource) generatedProductos() ([]ProductoRow, error) {
	out := make([]ProductoRow, 0, len(mockCatalog)*3)
	for rep := 0; rep < 3; rep++ {
		for _, p := range mockCatalog {
			cve := p.cve
			if rep > 0 {
				cve = fmt.Sprintf("%s-%d", p.cve, rep+1)
			}
			precio := p.precio * (1 + 0.03*float64(rep))
			out = append(out, ProductoRow{
				CveArt: cve, Nombre: p.nombre, Descr: p.notas, Linea: p.linea,
				Unidad: p.um, Exist: float64(5 + m.rnd.Intn(60)),
				Precio1:    round2(precio),
				Precio2:    round2(precio * 0.95),
				Precio3:    round2(precio * 0.90),
				Graduacion: p.grad, Origen: p.origen, NotasCata: p.notas,
				Estatus: "A",
			})
		}
	}
	return out, nil
}

func (m *MockSource) Clientes() ([]ClienteRow, error) {
	out := make([]ClienteRow, len(mockClientes))
	copy(out, mockClientes)
	return out, nil
}

// Facturas generates ~12 months of sales history across the mock clients,
// concentrated on the top sellers so CRM segmentation has signal.
func (m *MockSource) Facturas() ([]FacturaRow, error) {
	now := time.Now()
	var out []FacturaRow
	folio := 1
	for i := 0; i < 365; i += 3 + m.rnd.Intn(4) {
		fecha := now.AddDate(0, 0, -i)
		cli := mockClientes[m.rnd.Intn(len(mockClientes))]
		items := 1 + m.rnd.Intn(4)
		fr := FacturaRow{
			Folio:    fmt.Sprintf("F%06d", folio),
			ClaveCli: cli.Clave,
			Fecha:    fecha.Format("2006-01-02"),
		}
		folio++
		for j := 0; j < items; j++ {
			p := mockCatalog[m.rnd.Intn(len(mockCatalog))]
			cant := float64(1 + m.rnd.Intn(6))
			fr.Detalles = append(fr.Detalles, models.FacturaDetalle{CveArt: p.cve, Cant: cant, Precio: p.precio})
			fr.Total += round2(cant * p.precio)
		}
		out = append(out, fr)
	}
	return out, nil
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
