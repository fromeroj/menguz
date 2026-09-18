package sqlite

import (
	"menguz/internal/badger"
	"menguz/internal/models"
)

// Rebuild refreshes the analytical tables from Badger operational data.
// Called after every SAE sync and after order events. Full rebuild keeps
// the two stores consistent without tracking fine-grained deltas.
func (d *DB) Rebuild(st *badger.Store) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, q := range []string{
		"DELETE FROM products",
		"DELETE FROM customers",
		"DELETE FROM invoices",
		"DELETE FROM orders",
		"DELETE FROM order_items",
	} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}

	// products
	pstmt, err := tx.Prepare(`INSERT INTO products
		(cve_art, nombre, categoria, grupo, linea, precio_base, precio_l2, precio_l3, existencia, activo, ultima_sync)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer pstmt.Close()
	var p models.Producto
	if err := st.ScanPrefix(badger.PrefProducto, &p, func(key string) bool {
		_, _ = pstmt.Exec(p.CveArt, p.Nombre, p.Categoria, p.Grupo, p.Linea,
			p.PrecioBase, p.PrecioLista2, p.PrecioLista3, p.Existencia, b2i(p.Activo), p.UltimaSync.Format("2006-01-02T15:04:05Z07:00"))
		return true
	}); err != nil {
		return err
	}

	// customers
	cstmt, err := tx.Prepare(`INSERT INTO customers
		(id, nombre, rfc, email, telefono, ciudad, estado, lista_precio, saldo, ultima_sync)
		VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer cstmt.Close()
	var c models.Cliente
	if err := st.ScanPrefix(badger.PrefCliente, &c, func(key string) bool {
		_, _ = cstmt.Exec(c.ID, c.Nombre, c.RFC, c.Email, c.Telefono, c.Ciudad, c.Estado, c.ListaPrecio, c.Saldo, c.UltimaSync.Format("2006-01-02T15:04:05Z07:00"))
		return true
	}); err != nil {
		return err
	}

	// invoices (header only for analytics)
	istmt, err := tx.Prepare(`INSERT INTO invoices (id, cliente_id, fecha, total) VALUES (?,?,?,?)`)
	if err != nil {
		return err
	}
	defer istmt.Close()
	var f models.Factura
	if err := st.ScanPrefix(badger.PrefFactura, &f, func(key string) bool {
		_, _ = istmt.Exec(f.ID, f.ClienteID, f.Fecha.Format("2006-01-02"), f.Total)
		return true
	}); err != nil {
		return err
	}

	// orders + items
	ostmt, err := tx.Prepare(`INSERT INTO orders (id, folio, tipo, cliente_id, estado, total, created_at) VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer ostmt.Close()
	oiStmt, err := tx.Prepare(`INSERT INTO order_items (order_id, cve_art, nombre, cantidad, precio) VALUES (?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer oiStmt.Close()
	var o models.Orden
	if err := st.ScanPrefix(badger.PrefOrden, &o, func(key string) bool {
		_, _ = ostmt.Exec(o.ID, o.Folio, o.Tipo, o.ClienteID, o.Estado, o.Total, o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
		for _, it := range o.Items {
			_, _ = oiStmt.Exec(o.ID, it.CveArt, it.Nombre, it.Cantidad, it.Precio)
		}
		return true
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
