-- Analytics schema (SQLite). Populated by the Badger → SQLite pipeline.
-- Read-only analytics layer for CRM / BI queries.

CREATE TABLE IF NOT EXISTS products (
  cve_art      TEXT PRIMARY KEY,
  nombre       TEXT NOT NULL,
  categoria    TEXT,
  grupo        TEXT,
  linea        TEXT,
  precio_base  REAL,
  precio_l2    REAL,
  precio_l3    REAL,
  existencia   REAL,
  activo       INTEGER,
  ultima_sync  TEXT
);

CREATE TABLE IF NOT EXISTS customers (
  id           TEXT PRIMARY KEY,
  nombre       TEXT NOT NULL,
  rfc          TEXT,
  email        TEXT,
  telefono     TEXT,
  ciudad       TEXT,
  estado       TEXT,
  lista_precio INTEGER DEFAULT 1,
  saldo        REAL DEFAULT 0,
  ultima_sync  TEXT
);

CREATE TABLE IF NOT EXISTS invoices (
  id         TEXT PRIMARY KEY,
  cliente_id TEXT,
  fecha      TEXT,
  total      REAL
);
CREATE INDEX IF NOT EXISTS idx_invoices_cliente ON invoices(cliente_id);
CREATE INDEX IF NOT EXISTS idx_invoices_fecha ON invoices(fecha);

CREATE TABLE IF NOT EXISTS orders (
  id         TEXT PRIMARY KEY,
  folio      TEXT,
  tipo       TEXT,
  cliente_id TEXT,
  estado     TEXT,
  total      REAL,
  created_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_orders_cliente ON orders(cliente_id);
CREATE INDEX IF NOT EXISTS idx_orders_estado ON orders(estado);

CREATE TABLE IF NOT EXISTS order_items (
  order_id TEXT,
  cve_art  TEXT,
  nombre   TEXT,
  cantidad REAL,
  precio   REAL
);
CREATE INDEX IF NOT EXISTS idx_oi_order ON order_items(order_id);

CREATE TABLE IF NOT EXISTS sync_log (
  id         TEXT PRIMARY KEY,
  inicio     TEXT,
  fin        TEXT,
  productos  INTEGER,
  clientes   INTEGER,
  facturas   INTEGER,
  origen     TEXT,
  error      TEXT
);
