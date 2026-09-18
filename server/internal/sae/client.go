package sae

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/nakagami/firebirdsql"
	"menguz/internal/config"
	"menguz/internal/models"
)

// FirebirdSource reads directly from the SAE 8.0 database using a
// read-only user. Connection runs over VPN / IP whitelist per security annex.
type FirebirdSource struct {
	db *sql.DB
}

func NewFirebirdSource(cfg *config.Config) (*FirebirdSource, error) {
	dsn := fmt.Sprintf("%s:%s@%s:%d/%s",
		cfg.SAEUser, cfg.SAEPass, cfg.SAEHost, cfg.SAEPort, cfg.SAEDB)
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging firebird at %s:%d: %w", cfg.SAEHost, cfg.SAEPort, err)
	}
	return &FirebirdSource{db: db}, nil
}

func (f *FirebirdSource) Close() error { return f.db.Close() }

func (f *FirebirdSource) Productos() ([]ProductoRow, error) {
	rows, err := f.db.Query(qProductos)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductoRow
	for rows.Next() {
		var r ProductoRow
		if err := rows.Scan(&r.CveArt, &r.Nombre, &r.Descr, &r.Linea, &r.Unidad,
			&r.Exist, &r.Precio1, &r.Precio2, &r.Precio3,
			&r.Graduacion, &r.Origen, &r.NotasCata, &r.Estatus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (f *FirebirdSource) Clientes() ([]ClienteRow, error) {
	rows, err := f.db.Query(qClientes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClienteRow
	for rows.Next() {
		var r ClienteRow
		if err := rows.Scan(&r.Clave, &r.Nombre, &r.RFC, &r.Calle, &r.Colonia,
			&r.Ciudad, &r.Estado, &r.CP, &r.Telefono, &r.Email,
			&r.ListaPrec, &r.LimiteCred, &r.Saldo); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (f *FirebirdSource) Facturas() ([]FacturaRow, error) {
	rows, err := f.db.Query(qFacturas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type hdr struct {
		folio, clave, fecha string
		total               float64
	}
	var hdrs []hdr
	for rows.Next() {
		var h hdr
		if err := rows.Scan(&h.folio, &h.clave, &h.fecha, &h.total); err != nil {
			return nil, err
		}
		hdrs = append(hdrs, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]FacturaRow, 0, len(hdrs))
	for _, h := range hdrs {
		fr := FacturaRow{Folio: h.folio, ClaveCli: h.clave, Fecha: h.fecha, Total: h.total}
		drows, err := f.db.Query(qFacturaDetalle, h.folio)
		if err != nil {
			return out, err
		}
		for drows.Next() {
			var art string
			var cant, prec float64
			if err := drows.Scan(&art, &cant, &prec); err != nil {
				drows.Close()
				return out, err
			}
			fr.Detalles = append(fr.Detalles, models.FacturaDetalle{CveArt: art, Cant: cant, Precio: prec})
		}
		drows.Close()
		out = append(out, fr)
	}
	return out, nil
}
