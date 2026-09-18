# Runbook — Sofía, la Sommelier Digital

Guía de operación para el equipo de Menguz. Sofía es la asesora de vinos del
sitio (sección **El Sommelier Digital**): responde con el modelo **DeepSeek**
razonando sobre el inventario real de SAE (precios y existencias de la sync
nocturna) y las fichas de vino curadas en el panel.

---

## 1. Arranque y paro

```bash
cd server
make run        # build + sirve en :8080 (carga server/.env automáticamente)
```

Binario directo (p. ej. después de copiar `bin/menguz-server` al VPS):

```bash
./bin/menguz-server --port=8080 --static-dir=../design/app/dist
```

En producción (systemd) ya está la unidad `deploy/menguz.service`:

```bash
sudo systemctl start menguz
sudo systemctl status menguz
journalctl -u menguz -f
```

Paro: `Ctrl-C` en desarrollo, `sudo systemctl stop menguz` en producción.

## 2. Dónde vive la llave de DeepSeek

- Archivo: **`server/.env`** (está en `.gitignore`; **nunca** se sube al repo).
- Variables: `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL` (default `deepseek-chat`),
  `DEEPSEEK_BASE_URL` (default `https://api.deepseek.com`).
- El binario carga `.env` al arrancar **sin pisar** variables que ya existan
  en el sistema — en systemd se puede poner la llave en el unit y manda esa.

### Cambiar / rotar la llave

1. Edita `server/.env` (o el `Environment=` del unit en producción).
2. Reinicia el servidor. No hay nada más que tocar.

## 3. Cambiar entre DeepSeek y el fallback gratuito

- **Con llave** → Sofía usa DeepSeek con herramientas (búsqueda en inventario,
  fichas). Es el modo recomendado.
- **Sin llave** (`DEEPSEEK_API_KEY=` vacía) → motor determinista interno: mismo
  inventario, mismas garantías, pero respuestas de plantilla. Útil si se acaba
  la saldo de la cuenta o cae la API.

**Cómo detectar en qué modo está:** en el fallback, todas sus respuestas
empiezan con frases como *"Déjame revisar el inventario actual 🍷"* o
*"Con carne/asado manda un tinto con estructura 🥩"*. Si la conversación suena
"humana" y varía, está en DeepSeek. También en los logs del servidor no hay
llamadas externas en modo fallback.

## 4. Fichas de vino (lo que Sofía "sabe" de cada botella)

- Panel admin → **Productos** → columna **"ficha"** por artículo.
- Campos: tipo, uvas, bodega, región, país, añada, crianza, notas de cata,
  maridaje, descripción, ocasiones y tags.
- Botón **"⚡ generar fichas borrador"**: detecta tipo/uvas/añada desde el
  nombre del catálogo. **No pisa** fichas ya editadas a mano — es seguro
  re-correrlo.
- Las fichas sobreviven las syncs nocturnas (SAE solo toca precio/existencia).

## 5. Revisar lo que Sofía respondió

Panel admin → **Chat**. Las conversaciones del sommelier aparecen con
`canal: sommelier`. Sirve para auditar precios citados y detectar preguntas
frecuentes que conviene convertir en ficha o campaña.

## 6. Límites y costos

- El chat del sommelier tiene **rate limit por IP** (bucket del chatbot):
  ~30 mensajes/minuto por visitante. Si un cliente ve "demasiados mensajes",
  es eso; espera un minuto.
- Cada mensaje con DeepSeek = 1+ llamadas a la API (hasta 4 rondas de
  herramientas). Monitorea el consumo en la consola de DeepSeek.
- Timeout de respuesta: 60 s por mensaje; si DeepSeek falla, Sofía degrada
  automáticamente al fallback **sin caerse** (el cliente nunca ve un error).

## 7. Troubleshooting

| Síntoma | Causa probable | Acción |
|---|---|---|
| Responde "de plantilla" | Sin llave o llave inválida | Revisar `DEEPSEEK_API_KEY` en `.env` y reiniciar |
| "demasiados mensajes" | Rate limit del visitante | Normal; esperar 1 min |
| Recomienda cosas raras / sin fotos | Catálogo mock o sync vieja | Verificar `SAE_MOCK` y fecha en `/healthz` (`last_sync`) |
| No menciona notas de cata de una botella | Ficha vacía | Completar la ficha en Productos → ficha |
| Precio distinto al del mostrador | Sync nocturna pendiente | Ejecutar sync manual: panel → Sync → "ejecutar ahora" |
| `/docs` no carga | Build viejo | Recompilar: `cd server && make build` |

## 8. Soporte técnico

Desarrollo y soporte: **Darkvoice Center** — https://darkvoice.center/
API documentada en `/docs` (Swagger) cuando el servidor está corriendo.
