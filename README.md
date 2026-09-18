# Menguz — Transformación Digital (Meses 1–3)

Workspace del proyecto **Menguz Vinos & Licores** según la propuesta de
Transformación Digital (Konen.guru, agosto 2026). Este repositorio contiene
la implementación de los **tres primeros meses**: sitio B2C con catálogo
sincronizado a Aspel SAE 8.0, editor de campañas/promociones, chatbot IA,
portal mayorista B2B y CRM base.

## Contenido

| Ruta | Qué es |
|------|--------|
| [server/](server/) | Backend completo en Go (un solo binario): API, panel admin, sync SAE, chatbot, B2B, CRM. **Empezar por su README.** |
| [design/app/](design/app/) | Storefront React/Vite mobile-first (rediseño entregado). En producción lo sirve el binario Go; en desarrollo corre con `npm run dev` y proxy a la API. |
| [docs/](docs/) | Propuesta comercial (PDF), anexo técnico (HTML) y el paquete original del rediseño. |

## Arranque en 3 comandos

```bash
cd server && make frontend   # compila el storefront
make run                     # levanta todo en :8080 (datos demo de catálogo)
```

Panel admin: `http://localhost:8080/admin` (credenciales en el log del primer arranque).

## Estado por mes (fase 1 — Fundamentos & Venta Directa)

- **M1 Cimientos** ✅ storefront API + sync SAE nocturna (Firebird/mock) + checkout con referencia bancaria + panel admin.
- **M2 Atención & B2B** ✅ editor de campañas + chatbot IA (OpenAI/RAG con fallback) + WhatsApp Business webhook + portal B2B con listas de precios SAE.
- **M3 CRM base** ✅ pipeline de ventas + vista 360° del cliente + segmentación + exportación estructurada.

Los meses 4–12 (motor de campañas automatizado, lealtad, logística, cobranza,
CFDI 4.0, BI, Club de Mixología…) se construyen sobre esta base siguiendo el
roadmap del anexo técnico.
