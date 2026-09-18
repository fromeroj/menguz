# Estado del Proyecto — Plan vs. Realidad

**Menguz Vinos & Licores · Transformación Digital**
Fuente del plan: *Propuesta Transformación Digital — Menguz 2026* (PDF comercial) + *Propuesta Técnica-Comercial* (anexo HTML), ambas de agosto 2026, roadmap a 12 meses.
Estado al: **18 de septiembre de 2026** · Release: `v0.4.0` (tag en GitHub).

**Resumen en una línea:** los **3 primeros meses del plan están construidos, probados en navegador y desplegables como binario único**; además se entregaron funcionalidades que el plan no contemplaba (Sommelier IA con DeepSeek, Swagger UI, paginación). Lo que falta son los meses 4–12, más los accesos del lado de Menguz (Firebird real, WhatsApp Business verificado).

---

## 1. ✅ Puntos del plan CUMPLIDOS (Meses 1–3)

| # | Entregable del plan | Mes | Estado | Qué se entregó |
|---|---|---|---|---|
| 1 | **Sitio Web B2C** mobile-first | M1 | ✅ Cumplido | Catálogo con 962 productos reales y 907 fotos, búsqueda, categorías, paginación (10/página), carrito con cookie, checkout con **referencia de pago bancaria** (folio MGZ-XXXXXX + referencia MENGXXXXXX), rastreo de pedido, 100% responsive (QA en viewport móvil 390px y desktop) |
| 2 | **Sync diaria SAE 8.0 → servidor propio** | M1 | ✅ Cumplido (con matiz) | Pipeline nocturno completo: backup → extracción Firebird → Badger (operación) → SQLite (analítica). Programada 03:00 + sync al arranque si hay >24 h. **Matiz:** probada en modo mock contra el catálogo real exportado; la conexión Firebird directa queda listo para activarse con las credenciales de Menguz (ver §4) |
| 3 | **Editor de Campañas** sin desarrollador | M1 | ✅ Cumplido | CRUD completo en panel admin: banners de carrusel, Producto del Mes y especiales, con fechas, orden, imagen, precio de oferta y botón CTA; se publican solas en el sitio |
| 4 | **Panel admin base** | M1 | ✅ Cumplido | Panel HTML con sesión segura (CSRF): pedidos, productos, clientes, campañas, sync, chat |
| 5 | **Chatbot IA** (web, RAG, FAQ, seguimiento de pedidos, escalación) | M2 | ✅ Cumplido | Asistente con streaming SSE, RAG sobre catálogo, respuestas FAQ (horarios, envíos, pagos, facturación), rastreo por folio, escalación a WhatsApp; motor determinista de respaldo si no hay API key |
| 6 | **Portal B2B** (precios por lista SAE, pedidos masivos, historial) | M2 | ✅ Cumplido | Login por cliente con precio de su lista (lista 3 = vinatería, validado 10% bajo lista 1), catálogo B2B, pedidos masivos con captura rápida, historial de pedidos web |
| 7 | **CRM Base** (pipeline, 360°, segmentación, exportación) | M3 | ✅ Cumplido | Pipeline de oportunidades, vista 360° del cliente (SAE + pedidos web), segmentos automáticos, exportación CSV. La base de datos del CRM alimenta la analítica en SQLite |
| 8 | **Integración WhatsApp Business API** | M2 | ⚠️ Parcial | Webhook de Meta implementado (verificación + recepción con token, probado 403 sin token / 200 con token). **Falta:** cuenta verificada de Menguz en Meta Business Manager para activarlo (requisito del plan, Mes 2, item 8) |

**Extras cumplidos que el plan pedía transversalmente:**

- **"Un solo ejecutable en servidor propio, sin dependencias cloud"** (promesa central del anexo técnico): binario Go único, sin Docker, sin Cgo. ✅
- **"Interfaz de carga de catálogo con fichas técnicas y descripciones"**: fichas de vino por producto (uvas, bodega, notas de cata, maridaje, ocasiones) editables en panel, con generador de borradores. ✅
- **"Integración con SAE para actualización de precios e inventarios"**: ✅ (modo mock hasta entrega de credenciales).
- **Documentación técnica / runbooks**: Swagger UI en `/docs` (53 operaciones documentadas) + runbook de operación de Sofía. ✅

---

## 2. ❌ Qué NO está hecho todavía (Meses 4–12 del plan)

Estos módulos no se han iniciado. Ninguno bloquea la operación del mes 1–3; el orden sugerido respeta las fases del plan:

| Mes | Entregable pendiente | Dependencia para arrancar |
|---|---|---|
| M4 | Motor de Campañas (abandono de carrito, reactivación, cumpleaños, cross-sell; email + WhatsApp) | Cuenta de envío de email (SMTP) y WhatsApp Business activo |
| M4 | Programa de Lealtad (puntos, niveles Bronce/Plata/Oro, canje) | Definición de reglas de puntos con Menguz |
| M5 | Módulo de Logística (rutas, repartidores, tracking, foto/firma) | Proceso actual de reparto |
| M5 | App PWA para Repartidores | El anterior |
| M6 | Crédito y Cobranza (límites SAE, alertas, estados de cuenta) | Políticas de crédito |
| M6 | **Facturación CFDI 4.0** | **PAC contratado** (Finkok, SwSapiens… — requisito del plan, Mes 6, item 9) |
| M7 | CRM Inteligente: IA que sugiere ofertas por patrón de compra, alertas de churn | Histórico real de ventas en la plataforma |
| M8 | Dashboard de BI en tiempo real (rotación por SKU, rentabilidad, demanda, cohortes) | Datos reales acumulados |
| M8 | Alertas de Caducidad por lote (30/60/90 días) | Multialmacén/lotes configurados en SAE |
| M9 | Club de Mixología (suscripción, curaduría, recetas IA) | Modelo de negocio de suscripción |
| M10 | Eventos y Catas (cotizador, calendario, contrato digital) | — |
| M10 | Integración Marketplaces (Rappi, Uber Eats, Didi) | Cuentas de vendedor en cada marketplace |
| M11 | WhatsApp Commerce completo (catálogo y pagos en WhatsApp) | WhatsApp Business activo |
| M11–12 | Realidad Aumentada, chatbot v2 con voz/imágenes, ML avanzado (precios dinámicos), capacitación y plan de soporte | — |

---

## 3. 🆕 Qué se entregó que NO estaba en el plan (adiciones)

| Adición | Descripción | Por qué se hizo |
|---|---|---|
| **Sommelier Digital "Sofía"** | Asesor de vinos conversacional estilo Hedonism: DeepSeek con *tools* sobre el inventario vivo (precios/existencias reales) + fichas curadas; tarjetas de recomendación con foto/precio; fallback sin API key | Petición directa del cliente (similar a hedonism.co.uk); convierte el chatbot en vendedor, no solo en soporte |
| **DeepSeek en lugar de OpenAI** | El sommelier usa DeepSeek (`deepseek-chat`); el chatbot clásico sigue compatible con OpenAI | El cliente aportó la llave de DeepSeek; reduce costo y da tool-calling nativo. El plan pedía "API Key de OpenAI" — es intercambiable vía variable de entorno |
| **Swagger UI + OpenAPI 3.0** | Documentación viva de toda la API en `/docs` (53 operaciones) | Petición directa del cliente ("swagger y muy bien documentado") |
| **Paginación del catálogo** | 10 productos/página con controles numerados, conteo y scroll-assist (antes: scroll infinito) | Petición directa: 962 productos en una página eran imposibles de navegar |
| **Fondo blanco en catálogo** | Secciones Especiales y Sommelier pasaron de crema a blanco | Las fotos de producto tienen fondo blanco; en crema se veían "pegadas". Decisión visual del cliente |
| **Autoload de `.env`** | El binario carga `server/.env` al arrancar sin pisar variables del sistema | Robustez operativa (evita que un reinicio sin `source` pierda la llave de DeepSeek) |
| **Crédito a Darkvoice Center** | En el pie de página del sitio, como creadores | Petición directa del cliente |
| **Runbook de operación (español)** | Guía de arranque/paro, rotación de llaves, fallback, troubleshooting | Requisito de transición a operación del equipo Menguz |
| **Tag `v0.4.0` y repositorio GitHub** | 8 commits, repo privado `fromeroj/menguz` | Trazabilidad del entregable |

---

## 4. 🔀 Decisiones técnicas tomadas (y sus porqués)

1. **Referencia bancaria, NO pasarela de tarjeta.**
   El plan es **internamente contradictorio**: el PDF comercial (pág. 7) dice *"checkout con referencia para pagos bancarios (no pasarela de pagos con tarjeta)"*, pero el anexo técnico y la fase 1 dicen *"pasarela de pagos"*. Se siguió el **PDF comercial**: checkout que genera folio + referencia bancaria para transferencia. **Se necesita confirmación por escrito** si en algún mes se requiere pasarela (Mercado Pago/Stripe) — eso cambia el alcance del M1 y agrega comisiones.

2. **Stack: Go + Echo + Badger + SQLite, binario único.**
   Fiel al anexo técnico ("un solo ejecutable en un servidor propio"). Badger = operación en caliente (catálogo, pedidos, sesiones); SQLite = analítica/CRM. Sin servicios externos, costos predecibles, VPS de 2 GB/1 vCPU suficiente.

3. **Modo mock de SAE hasta entrega de accesos.**
   El plan lista como requisito del Mes 1: IP/puerto Firebird, usuario de solo lectura, ruta `.fdb`, configuración de multialmacén y contenido de `INVE_CLIB01`. **Ninguno ha sido entregado aún**, así que el desarrollo y las pruebas corren sobre un mock con el catálogo real de 962 productos/907 fotos. La integración real es activable por configuración (`SAE_*`), pero **las columnas de `internal/sae/queries.go` deben validarse contra el diccionario real de SAE del cliente** antes de producción.

4. **Chatbot con degradación elegante.**
   Con llave → LLM con streaming + RAG; sin llave → motor de reglas determinista sobre el mismo inventario. El sitio nunca se queda sin respuesta. Aplica igual al Sommelier (DeepSeek ↔ fallback).

5. **Autenticación JWT + panel con sesión cookie y CSRF.**
   La API admin usa JWT con refresh rotativo (15 min / 7 días); el panel HTML usa cookie con doble submit. Rate limiting por IP, headers de seguridad, bcrypt costo 12.

6. **Frontend: se aprovechó el diseño existente.**
   El plan pedía "rediseño mobile-first"; la carpeta de diseño entregada por el cliente ya era un rediseño completo (React + Tailwind), por lo que la inversión fue conectarlo al backend (promos, catálogo, sommelier) en lugar de rehacerlo.

7. **Nombre y persona "Sofía".**
   Elección de diseño de la sesión (no venía en el plan): nombre corto, cálido, natural en español. Aprobado por el cliente. Cambiable en 3 archivos.

8. **12 tarjetas máximo por respuesta del sommelier.**
   El modelo puede acumular decenas de resultados en varias rondas de herramientas; se capó el carrusel para no saturar al cliente.

---

## 5. 📋 Requisitos pendientes del lado de Menguz (bloqueantes para activar lo ya construido)

| # | Requisito (según plan) | Mes plan | Bloquea |
|---|---|---|---|
| 1 | Acceso Firebird SAE 8.0 (IP, puerto 3050, usuario solo-lectura) | M1 | Sync real (hoy: mock) |
| 2 | Ruta del `.fdb` y configuración de multialmacén | M1 | Sync real |
| 3 | Contenido de campos libres `INVE_CLIB01` (graduación, región, año, notas) | M1 | Fichas de vino automáticas completas |
| 4 | Datos bancarios reales (banco, CLABE, titular) | M1 | Referencias de pago correctas en producción |
| 5 | Dominio + VPS (2 GB/1 vCPU/50 GB) | M1 | Despliegue |
| 6 | Cuenta WhatsApp Business verificada (Meta Business Manager) | M2 | Chatbot y notificaciones por WhatsApp |
| 7 | PAC autorizado (Finkok, SwSapiens…) | M6 | Facturación CFDI 4.0 |

---

## 6. 🔍 Cómo verificar lo entregado

- **Sitio + Sofía:** scroll a *El Sommelier Digital*, escribir cualquier petición (ocasión, comida, presupuesto) → respuesta con tarjetas reales (foto/precio/existencia).
- **Catálogo:** 10 productos/página, filtros por categoría, búsqueda, conteo "Mostrando X–Y de Z".
- **Admin:** `/admin` → campañas (crear/publica sola), pedidos, fichas de vino, clientes B2B, CRM, chat.
- **API:** `/docs` (Swagger UI interactivo, 53 operaciones).
- **Repo:** `github.com/fromeroj/menguz`, tag `v0.4.0`.
- **Operación:** [RUNBOOK_SOFIA.md](RUNBOOK_SOFIA.md).

---

*Documento generado por Darkvoice Center — https://darkvoice.center/*
