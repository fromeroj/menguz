// Package sae implements the Aspel SAE 8.0 connector: nightly extraction
// of products, prices, stock, clients and sales history from the Firebird
// database into the platform's operational store (Badger).
//
// Source tables (per technical annex):
//
//	INVE01       catálogo de productos, existencias, precios base
//	INVE_CLIB01  campos libres de producto (graduación, notas de cata...)
//	CLIE01       clientes, RFC, dirección, lista de precios asignada
//	FACT01 / FACTD01  facturas de venta (encabezado y detalle)
//	CUEN_M01     cuentas por cobrar, saldos, vencimientos
//	PRECIO_X01   listas de precios por cliente o volumen
//
// NOTE: column names vary slightly across SAE 8.0 revisions and customized
// installations. Adjust queries.go to the customer's actual dictionary
// (SELECT RDB$FIELD_NAME FROM RDB$RELATION_FIELDS WHERE RDB$RELATION_NAME='INVE01').
package sae

import (
	"menguz/internal/models"
)

// ProductoRow mirrors the INVE01 (+ INVE_CLIB01) extraction.
type ProductoRow struct {
	CveArt     string
	Nombre     string
	Descr      string
	Linea      string
	Unidad     string
	Exist      float64
	Precio1    float64
	Precio2    float64
	Precio3    float64
	Graduacion string
	Origen     string
	NotasCata  string
	Estatus    string // "A" activo
}

// ClienteRow mirrors the CLIE01 extraction.
type ClienteRow struct {
	Clave      string
	Nombre     string
	RFC        string
	Calle      string
	Colonia    string
	Ciudad     string
	Estado     string
	CP         string
	Telefono   string
	Email      string
	ListaPrec  int
	LimiteCred float64
	Saldo      float64
}

// FacturaRow mirrors FACT01 header + FACTD01 detail.
type FacturaRow struct {
	Folio    string
	ClaveCli string
	Fecha    string // YYYY-MM-DD
	Total    float64
	Detalles []models.FacturaDetalle
}

// lineaMap translates SAE product lines to storefront categories/groups.
// Extend with the customer's real line catalog during onboarding.
var lineaMap = map[string][2]string{
	"VIN-TIN":   {"Vino Tinto", "Vinos"},
	"VIN-BLA":   {"Vino Blanco", "Vinos"},
	"VIN-ROS":   {"Vino Rosado", "Vinos"},
	"VIN-ESP":   {"Espumoso", "Vinos"},
	"TEQ-BLA":   {"Tequila", "Licores"},
	"TEQ-REP":   {"Tequila", "Licores"},
	"TEQ-ANE":   {"Tequila", "Licores"},
	"MEZ-JOV":   {"Mezcal", "Licores"},
	"MEZ-ANE":   {"Mezcal", "Licores"},
	"WHI-SCOT":  {"Whisky", "Licores"},
	"WHI-BOURB": {"Whisky", "Licores"},
	"RON-BLA":   {"Ron", "Licores"},
	"RON-ANE":   {"Ron", "Licores"},
	"VOD-PRE":   {"Vodka", "Licores"},
	"GIN-LON":   {"Ginebra", "Licores"},
	"BRA-JOV":   {"Brandy", "Licores"},
	"CON-VS":    {"Cognac", "Licores"},
	"LIC-CRE":   {"Licor", "Licores"},
	"ROM-TRAD":  {"Rompope", "Licores"},
	"MEZ-CL":    {"Aguardiente", "Licores"},
	"APER-BIT":  {"Aperitivo", "Licores"},
	"ESP-CAVA":  {"Champagne", "Vinos"},
	"CER-ART":   {"Cerveza", "Licores"},
	"MEZ-REF":   {"Refresco", "Mezcladores"},
	"MEZ-JAR":   {"Jarabe", "Mezcladores"},
	"MEZ-ENE":   {"Energizante", "Mezcladores"},
	"MEZ-AGU":   {"Agua", "Mezcladores"},
	"MEZ-JUG":   {"Jugo", "Mezcladores"},
	"ABA-GRAL":  {"Abarrotes", "Abarrotes"},
	"ABA-QUE":   {"Quesos", "Abarrotes"},
}

// mapLinea returns (categoria, grupo) for a SAE line code.
func mapLinea(linea string) (string, string) {
	if m, ok := lineaMap[linea]; ok {
		return m[0], m[1]
	}
	return "Otros", "Otros"
}
