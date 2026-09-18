# menguz-server

Backend completo de **Menguz Vinos & Licores** — un solo binario en Go que incluye
servidor HTTP, base de datos embebida, panel administrativo, storefront API y la
sincronización nocturna con **Aspel SAE 8.0** (Firebird).

Implementa los **meses 1–3** de la propuesta de Transformación Digital
(Konen.guru, agosto 2026):

| Mes | Entregable | Estado |
|-----|-----------|--------|
| M1 | Sitio B2C (catálogo, carrito, checkout con **referencia de pago bancaria** — sin pasarela de tarjeta, según propuesta PDF), sync diaria SAE 8.0, panel admin base | ✅ |
| M1/M2 | **Editor de Campañas**: CRUD de promociones (carrusel, Producto del Mes, especiales) publicadas en el sitio sin tocar código | ✅ |
| M2 | **Chatbot IA** (OpenAI GPT-4o con streaming SSE + RAG sobre el catálogo, guardrails de precios; motor determinista de respaldo) | ✅ |
| M2 | **WhatsApp Business API** (webhook de verificación + recepción, respuesta automática) | ✅ |
| M2/M3 | **Portal B2B** con precios por lista de SAE (listas 1/2/3), pedidos masivos, historial | ✅ |
| M3 | **CRM base**: pipeline de ventas, vista 360° del cliente, segmentación, exportación CSV | ✅ |

## Arquitectura

```
React storefront (design/app/dist, servido por Go)
        │
  Echo v4  ──►  Badger (operación: productos, pedidos, campañas, chat)
        │            │
  SQLite ◄── Rebuild (analítica: CRM/BI)
        │
  sync nocturna 03:00 ──► Firebird SAE 8.0 (INVE01, CLIE01, FACT01…)
                          o MockSource en desarrollo
```

- **Go 1.23+**, un solo binario, sin Docker ni dependencias de sistema.
- **Badger** (key-value LSM) como base operativa; **SQLite** (modernc, sin Cgo) como capa analítica.
- **JWT** (acceso 15 min) + **refresh rotativo** (7 días); bcrypt costo 12.
- Middleware: CORS, rate limit por IP, security headers, CSRF (double-submit) en el panel.

## Arranque rápido (desarrollo)

```bash
cd server
cp .env.example .env          # ajusta si quieres
make frontend                  # construye el storefront React (design/app)
make run                       # sirve todo en :8080 — catálogo demo incluido
```

Sin datos de SAE a la mano, el servidor corre en **modo mock**: genera un catálogo
realista (120 productos, 5 clientes, ~12 meses de facturas) para desarrollar y
probar todo el ecosistema. Cuando Menguz entregue las credenciales Firebird
(ver *Requerimientos M1* de la propuesta), basta con llenar `SAE_*` en `.env`.

## Panel administrativo

`http://localhost:8080/admin` — el correo/contraseña de admin se generan en el
primer arranque (aparecen en el log como `admin_bootstrapped`).

- **Campañas**: alta/edición/programación de promos; se publican solas en el sitio.
- **Pedidos**: seguimiento y cambio de estado (cancelación libera existencia).
- **Productos**: vista de catálogo sincronizado (lectura; se edita en SAE).
- **Clientes**: habilitación del portal B2B por cliente (correo + contraseña).
- **CRM**: pipeline, top clientes, segmentos, exportación CSV, vista 360°.
- **Chat**: revisión de conversaciones y escalaciones.
- **Sync**: bitácora de sincronizaciones y ejecución manual.

## API principal

| Ruta | Descripción |
|------|-------------|
| `GET /data/products.json` | Catálogo en el formato del storefront (`{n,p,c,g,i}`) |
| `GET /api/products?q=&categoria=` | Búsqueda/filtros con paginación |
| `GET /api/promos?tipo=carousel\|producto_mes\|especial` | Campañas vigentes |
| `POST /api/cart/items` · `GET /api/cart` | Carrito (cookie) |
| `POST /api/checkout` | Crea pedido con **referencia bancaria única** |
| `GET /api/orders/lookup?folio=&email=` | Rastreo de pedido |
| `POST /api/chat` | Chatbot (SSE) |
| `POST /api/b2b/login` · `GET /api/b2b/catalog` · `POST /api/b2b/orders` | Portal mayorista |
| `POST /api/auth/login` · `POST /api/auth/refresh` | Autenticación admin |
| `GET/POST /api/admin/*` | CRUD admin (JWT) |
| `GET /webhooks/whatsapp` · `POST` | Meta WhatsApp |
| `GET /healthz` | Salud + última sync |

## Conexión a Aspel SAE 8.0

1. Crear usuario Firebird de **solo lectura** en el servidor SAE.
2. Abrir VPN / whitelist de IP entre el VPS y el servidor Menguz.
3. Configurar en `.env`: `SAE_HOST`, `SAE_PORT` (3050), `SAE_USER`, `SAE_PASS`,
   `SAE_DB` (ruta del `.fdb`, p. ej. `C:\Archivos Comunes\Aspel\SAE8.00\Empresa01\Datos\Empresa01.fdb`).
4. La sync corre diariamente a las `SYNC_AT` (03:00) y deja bitácora en **Sync**.

> Los nombres de columnas pueden variar entre revisiones de SAE 8.0. Ajusta
> `internal/sae/queries.go` con el diccionario real:
> `SELECT RDB$FIELD_NAME FROM RDB$RELATION_FIELDS WHERE RDB$RELATION_NAME='INVE01'`.

## Deploy a producción

```bash
make build-linux               # binario estático linux/amd64
# subir + systemd:
cp deploy/menguz.service /etc/systemd/system/
systemctl enable --now menguz
# o con un comando:  make deploy VPS_USER=root VPS_HOST=tu-vps
```

Requisitos del VPS (según propuesta): 1 vCPU / 2 GB RAM / 50 GB SSD, Ubuntu 22.04+.
TLS 1.3 con Caddy o lego como reverse proxy (puertos 80/443).

## Backup

- Cada sync escribe un snapshot consistente de Badger en `data/backups/*.bak`.
- `scripts/backup.sh` (cron nocturno) replica `data/` al storage secundario,
  conservando los últimos 14 días. Restauración: `scripts/restore.sh`.

## Estructura

```
server/
├── main.go                     # entry point: init DBs, servidor, scheduler
├── Makefile
├── internal/
│   ├── api/                    # handlers HTTP (público, admin, B2B, CRM, chat)
│   ├── badger/                 # wrapper KV + esquema de keys + backup
│   ├── chatbot/                # agente, RAG, cliente OpenAI SSE
│   ├── config/                 # flags + variables de entorno
│   ├── middleware/             # JWT, CORS, rate limit, headers, CSRF
│   ├── models/                 # structs compartidos
│   ├── sae/                    # conector Firebird + mock + pipeline de sync
│   └── sqlite/                 # analítica + rebuild desde Badger
├── migrations/001_init.sql     # schema analítico (copia en internal/sqlite)
├── scripts/                    # deploy.sh, backup.sh, restore.sh
└── deploy/menguz.service       # unidad systemd
```
