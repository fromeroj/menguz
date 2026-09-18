package sae

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"menguz/internal/badger"
	"menguz/internal/config"
	"menguz/internal/models"
	"menguz/internal/sqlite"
)

// Source abstracts the data origin: the real Firebird SAE database or the
// mock generator used in development.
type Source interface {
	Productos() ([]ProductoRow, error)
	Clientes() ([]ClienteRow, error)
	Facturas() ([]FacturaRow, error)
	Close() error
}

// NewSource picks Firebird when a host is configured, otherwise the mock.
func NewSource(cfg *config.Config) (Source, string, error) {
	if !cfg.SAEMock && cfg.SAEHost != "" {
		src, err := NewFirebirdSource(cfg)
		if err != nil {
			return nil, "", err
		}
		return src, "firebird", nil
	}
	return NewMockSource(), "mock", nil
}

// imageURL normalizes a storefront image path to an absolute URL path.
func imageURL(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "http") {
		return p
	}
	return "/" + p
}

// Sync runs the full nightly pipeline: backup → extract → load Badger →
// rebuild SQLite analytics → record log. Safe to call concurrently with
// serving traffic (Badger supports concurrent readers/writers).
func Sync(cfg *config.Config, st *badger.Store, ana *sqlite.DB) (*models.SyncLog, error) {
	start := time.Now()
	slog := &models.SyncLog{
		ID:     start.Format("20060102150405"),
		Inicio: start,
	}
	defer func() {
		slog.Fin = time.Now()
		st.PutJSON(badger.PrefSyncLog+slog.ID, slog)
		st.PutString(badger.PrefMeta+badger.MetaLastSync, slog.Fin.Format(time.RFC3339))
	}()

	// backup before overwriting operational data
	if path, err := st.Backup(cfg.DataDir); err != nil {
		log.Printf("sync: backup failed (continuing): %v", err)
	} else {
		log.Printf("sync: backup written to %s", path)
	}

	src, origen, err := NewSource(cfg)
	if err != nil {
		slog.Error = err.Error()
		return slog, err
	}
	defer src.Close()
	slog.Origen = origen

	prods, err := src.Productos()
	if err != nil {
		slog.Error = err.Error()
		return slog, fmt.Errorf("reading products: %w", err)
	}
	clis, err := src.Clientes()
	if err != nil {
		slog.Error = err.Error()
		return slog, fmt.Errorf("reading clients: %w", err)
	}
	facs, err := src.Facturas()
	if err != nil {
		slog.Error = err.Error()
		return slog, fmt.Errorf("reading invoices: %w", err)
	}

	// wipe + reload using a single WriteBatch (atomic)
	now := time.Now()
	wb := st.DB().NewWriteBatch()
	defer wb.Cancel()

	// delete previous snapshot keys
	for _, pref := range []string{badger.PrefProducto, badger.PrefCliente, badger.PrefFactura} {
		if err := st.ScanPrefix(pref, &struct{}{}, func(key string) bool {
			_ = wb.Delete([]byte(key))
			return true
		}); err != nil {
			slog.Error = err.Error()
			return slog, err
		}
	}

	for _, r := range prods {
		cat, grupo := mapLinea(r.Linea)
		p := models.Producto{
			CveArt: r.CveArt, Nombre: r.Nombre, Descripcion: r.Descr,
			PrecioBase: r.Precio1, PrecioLista2: r.Precio2, PrecioLista3: r.Precio3,
			Existencia: r.Exist, UnidadMedida: r.Unidad,
			Linea: r.Linea, Categoria: cat, Grupo: grupo,
			Graduacion: r.Graduacion, Origen: r.Origen, NotasCata: r.NotasCata,
			ImagenURL: imageURL(r.Imagen),
			Activo:    r.Estatus != "B", UltimaSync: now,
		}
		if b, err := json.Marshal(p); err == nil {
			_ = wb.Set([]byte(badger.PrefProducto+p.CveArt), b)
		}
		slog.Productos++
	}
	for _, r := range clis {
		c := models.Cliente{
			ID: r.Clave, Nombre: r.Nombre, RFC: r.RFC, Email: r.Email, Telefono: r.Telefono,
			Calle: r.Calle, Colonia: r.Colonia, Ciudad: r.Ciudad, Estado: r.Estado, CP: r.CP,
			ListaPrecio: r.ListaPrec, LimiteCred: r.LimiteCred, Saldo: r.Saldo,
			UltimaSync: now,
		}
		if b, err := json.Marshal(c); err == nil {
			_ = wb.Set([]byte(badger.PrefCliente+c.ID), b)
		}
		slog.Clientes++
	}
	for _, r := range facs {
		f := models.Factura{
			ID: r.Folio, ClienteID: r.ClaveCli, Total: r.Total,
			Detalles: r.Detalles,
		}
		if t, err := time.Parse("2006-01-02", r.Fecha); err == nil {
			f.Fecha = t
		}
		if b, err := json.Marshal(f); err == nil {
			_ = wb.Set([]byte(badger.PrefFactura+f.ID), b)
		}
		slog.Facturas++
	}
	if err := wb.Flush(); err != nil {
		slog.Error = err.Error()
		return slog, err
	}

	// refresh analytical layer
	if ana != nil {
		if err := ana.Rebuild(st); err != nil {
			slog.Error = err.Error()
			return slog, err
		}
	}
	return slog, nil
}
