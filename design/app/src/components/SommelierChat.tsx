import { useCallback, useEffect, useRef, useState } from 'react'

/* ------------------------------------------------------------------ */
/* SommelierChat — floating chat widget (ChatKit-style launcher +      */
/* window). Mount once at app level; open it from anywhere via         */
/*   window.dispatchEvent(new CustomEvent('sommelier:open'))           */
/* ------------------------------------------------------------------ */

interface Rec {
  cve_art: string
  nombre: string
  precio: number
  existencia: number
  imagen: string
  categoria: string
  bodega?: string
  uvas?: string
  anada?: string
  maridaje?: string
}

type Msg = {
  id: number
  role: 'user' | 'assistant'
  text: string
  recs: Rec[]
  streaming: boolean
  error: boolean
}

const CHIPS = [
  '🥩 Para un asado de carne',
  '🎁 Un regalo impresionante',
  '🥂 Para brindis / celebración',
  '🐟 Con mariscos o sushi',
  '💰 Tinto bueno por menos de $400',
  '☕ Digestivo para cerrar la cena',
]

const WELCOME =
  '¡Hola! Soy Sofía, sommelier de Menguz 🍷 Cuéntame: ¿qué vas a comer, en qué ocasión estás pensando o cuál es tu presupuesto?'

const fmt = (p: number) =>
  `$${p.toLocaleString('es-MX', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

const waRec = (r: Rec) =>
  `https://wa.me/525511322231?text=${encodeURIComponent(`Me interesa: ${r.nombre} (${r.cve_art})`)}`

let nextId = 1

/* ------------------------------------------------------------------ */

function RecImage({ rec }: { rec: Rec }) {
  const [failed, setFailed] = useState(false)
  if (!rec.imagen || failed) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-[var(--wine-deep)]/40 px-3 text-center">
        <span className="text-3xl" aria-hidden>
          🍷
        </span>
        <span className="text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--cream)]/60">
          {rec.categoria}
        </span>
      </div>
    )
  }
  return (
    <img
      src={`/${rec.imagen}`}
      alt={rec.nombre}
      loading="lazy"
      decoding="async"
      onError={() => setFailed(true)}
      className="max-h-full w-auto object-contain transition-transform duration-500 group-hover:scale-[1.06]"
    />
  )
}

function RecCard({ rec }: { rec: Rec }) {
  const meta = [rec.bodega, rec.uvas ?? rec.anada].filter(Boolean).join(' · ')
  return (
    <a
      href={waRec(rec)}
      target="_blank"
      rel="noreferrer"
      data-testid="sommelier-card"
      className="group block w-36 shrink-0 overflow-hidden rounded-xl bg-[var(--cream)] text-[var(--ink)] shadow-[0_1px_2px_rgba(0,0,0,0.25)] transition-transform duration-300 hover:-translate-y-1"
    >
      <div className="relative flex aspect-[4/5] items-center justify-center bg-white p-3">
        <RecImage rec={rec} />
        {rec.existencia <= 0 && (
          <span className="absolute left-2 top-2 rounded-full bg-[var(--ink)] px-2 py-1 text-[9px] font-semibold uppercase tracking-[0.12em] text-[var(--cream)]">
            Agotado
          </span>
        )}
      </div>
      <div className="p-3">
        <h4 className="text-[12px] font-medium leading-snug [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:2] overflow-hidden">
          {rec.nombre}
        </h4>
        {meta && <p className="mt-1 truncate text-[11px] text-[var(--ink)]/55">{meta}</p>}
        <p className="font-serif-accent mt-1.5 text-base leading-none">{fmt(rec.precio)}</p>
      </div>
    </a>
  )
}

/* ------------------------------------------------------------------ */

export default function SommelierChat() {
  const [open, setOpen] = useState(false)
  const [msgs, setMsgs] = useState<Msg[]>([
    { id: nextId++, role: 'assistant', text: WELCOME, recs: [], streaming: false, error: false },
  ])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [started, setStarted] = useState(false)
  const sessionId = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const currentId = useRef<number | null>(null)

  /* open on demand from any part of the site (CTAs, nav…) */
  useEffect(() => {
    const onOpen = () => setOpen(true)
    window.addEventListener('sommelier:open', onOpen)
    return () => window.removeEventListener('sommelier:open', onOpen)
  }, [])

  /* close with Escape */
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  /* focus input + scroll to newest when opened */
  useEffect(() => {
    if (!open) return
    const el = scrollRef.current
    if (el) el.scrollTop = el.scrollHeight
    inputRef.current?.focus()
  }, [open])

  /* auto-scroll to newest content */
  useEffect(() => {
    const el = scrollRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [msgs])

  const patchCurrent = useCallback((fn: (m: Msg) => Msg) => {
    /* capture the id now: updaters may run after send() has already
       reset currentId.current (all events can arrive in one chunk) */
    const id = currentId.current
    if (id == null) return
    setMsgs((prev) => prev.map((m) => (m.id === id ? fn(m) : m)))
  }, [])

  const handleEvent = useCallback(
    (event: string, data: string) => {
      if (event === 'session') {
        sessionId.current = data.trim()
      } else if (event === 'delta') {
        patchCurrent((m) => ({ ...m, text: m.text + data }))
      } else if (event === 'products') {
        try {
          const recs = JSON.parse(data) as Rec[]
          patchCurrent((m) => ({ ...m, recs }))
        } catch {
          /* malformed payload: ignore */
        }
      } else if (event === 'done') {
        patchCurrent((m) => ({ ...m, text: data, streaming: false }))
      }
    },
    [patchCurrent],
  )

  const send = async (raw: string) => {
    const text = raw.trim()
    if (!text || streaming) return

    setStarted(true)
    const user: Msg = {
      id: nextId++,
      role: 'user',
      text,
      recs: [],
      streaming: false,
      error: false,
    }
    const asst: Msg = {
      id: nextId++,
      role: 'assistant',
      text: '',
      recs: [],
      streaming: true,
      error: false,
    }
    currentId.current = asst.id
    setMsgs((prev) => [...prev, user, asst])
    setStreaming(true)
    setInput('')

    try {
      const res = await fetch('/api/sommelier/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: text,
          ...(sessionId.current ? { session_id: sessionId.current } : {}),
        }),
      })
      if (!res.ok || !res.body) throw new Error(`HTTP ${res.status}`)

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''

      const dispatch = (block: string) => {
        let event = 'message'
        const dataLines: string[] = []
        for (const line of block.split('\n')) {
          if (line.startsWith('event:')) event = line.slice(6).trim()
          else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''))
          else if (dataLines.length > 0) dataLines.push(line) /* multi-line data payload */
        }
        if (dataLines.length > 0) handleEvent(event, dataLines.join('\n'))
      }

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buf += decoder.decode(value, { stream: true })
        const blocks = buf.split(/\r?\n\r?\n/)
        buf = blocks.pop() ?? ''
        for (const block of blocks) dispatch(block)
      }
      buf += decoder.decode()
      if (buf.trim()) dispatch(buf)
    } catch {
      patchCurrent((m) =>
        m.text
          ? { ...m, streaming: false }
          : {
              ...m,
              streaming: false,
              error: true,
              text: 'Tuve un problema de conexión, intenta de nuevo.',
            },
      )
    } finally {
      setStreaming(false)
      currentId.current = null
    }
  }

  return (
    <>
      {/* launcher — occupies the classic FAB slot (replaces the WhatsApp FAB) */}
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={open ? 'Cerrar chat con la sommelier' : 'Abrir chat con la sommelier'}
        aria-expanded={open}
        data-testid="sommelier-launcher"
        className="fixed bottom-5 right-5 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--wine)] text-[var(--cream)] shadow-[0_10px_28px_rgba(0,0,0,0.35)] transition-transform duration-300 hover:scale-110 active:scale-95 md:bottom-8 md:right-8 md:h-16 md:w-16"
      >
        {open ? (
          <svg viewBox="0 0 24 24" className="h-6 w-6 fill-current" aria-hidden>
            <path d="M18.3 5.71 12 12.01 5.7 5.71 4.29 7.12l6.3 6.3-6.3 6.3 1.41 1.41 6.3-6.3 6.3 6.3 1.41-1.41-6.3-6.3 6.3-6.3z" />
          </svg>
        ) : (
          <svg viewBox="0 0 24 24" className="h-7 w-7 fill-current md:h-8 md:w-8" aria-hidden>
            <path d="M8.1 13.34c.13-.32.22-.65.28-.99-.06-.34-.15-.67-.28-.99-.38-.91-.28-2 .27-2.83.48-.72 1.26-1.16 2.09-1.23.83-.07 1.66.23 2.24.83.53.55.82 1.28.8 2.03-.02.75-.34 1.46-.88 1.98-.58.56-1.35.87-2.15.86-.8-.01-1.56-.33-2.14-.9l-.01-.01c-.33-.3-.6-.66-.8-1.07-.04 0-.07-.01-.11-.02-.15.35-.35.68-.6.98zm7.8 6.16c1.05.51 2.25.42 3.22-.24.88-.6 1.42-1.58 1.42-2.63 0-.97-.45-1.89-1.23-2.49-.35-.27-.74-.44-1.16-.5-.11-.02-.23-.02-.34 0 .07.23.12.47.15.71.02.2.03.4.02.6.03 1.51-.68 2.96-1.9 3.89-.24.18-.5.34-.77.46.12.14.25.27.4.38.67.52 1.49.8 2.33.8.24 0 .48-.03.71-.08.14-.03.28-.07.42-.11-.01-.04-.02-.08-.04-.11-.49-.09-.95-.31-1.32-.66l-.63.49zM12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z" />
          </svg>
        )}
      </button>

      {/* chat window */}
      {open && (
        <div
          data-testid="sommelier-window"
          role="dialog"
          aria-label="Chat con Sofía, sommelier de Menguz"
          className="fixed bottom-5 right-5 z-50 flex h-[min(620px,calc(100dvh-6rem))] w-[min(400px,calc(100vw-2.5rem))] flex-col overflow-hidden rounded-3xl bg-[var(--cellar)] shadow-[0_28px_70px_rgba(23,19,16,0.45)] md:bottom-8 md:right-8"
        >
          <div className="grain pointer-events-none absolute inset-0" />

          {/* header */}
          <div className="relative flex items-center gap-3 border-b border-white/10 px-5 py-4">
            <span className="flex h-10 w-10 items-center justify-center rounded-full bg-[var(--wine)] text-xl">
              🍷
            </span>
            <div className="flex-1">
              <p className="font-display text-sm uppercase tracking-[0.14em] text-[var(--cream)]">
                Sofía · Sommelier
              </p>
              <p className="text-[11px] text-[var(--cream)]/55">
                En línea · recomienda del inventario real
              </p>
            </div>
            <button
              onClick={() => setOpen(false)}
              aria-label="Cerrar chat"
              className="flex h-8 w-8 items-center justify-center rounded-full text-[var(--cream)]/70 transition-colors hover:bg-white/10 hover:text-[var(--cream)]"
            >
              <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
                <path d="M18.3 5.71 12 12.01 5.7 5.71 4.29 7.12l6.3 6.3-6.3 6.3 1.41 1.41 6.3-6.3 6.3 6.3 1.41-1.41-6.3-6.3 6.3-6.3z" />
              </svg>
            </button>
          </div>

          {/* messages */}
          <div
            ref={scrollRef}
            className="no-scrollbar relative flex flex-1 flex-col gap-4 overflow-y-auto px-4 py-5"
          >
            {msgs.map((m) =>
              m.role === 'user' ? (
                <div key={m.id} className="flex justify-end">
                  <p className="max-w-[80%] rounded-2xl rounded-br-sm bg-[var(--wine)] px-4 py-3 text-sm leading-relaxed text-[var(--cream)]">
                    {m.text}
                  </p>
                </div>
              ) : (
                <div key={m.id} className="flex justify-start">
                  <div className="max-w-[88%]">
                    <div className="rounded-2xl rounded-bl-sm bg-[var(--cream-2)] px-4 py-3 text-sm leading-relaxed text-[var(--ink)]">
                      {m.text}
                      {m.streaming && (
                        <span className="ml-1 inline-block h-4 w-2 animate-pulse bg-[var(--wine)] align-middle" />
                      )}
                    </div>
                    {m.recs.length > 0 && (
                      <div className="no-scrollbar mt-3 flex gap-3 overflow-x-auto pb-1">
                        {m.recs.map((r) => (
                          <RecCard key={r.cve_art} rec={r} />
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ),
            )}
          </div>

          {/* quick prompts — legible at rest: cream border/text on dark panel */}
          {!started && (
            <div className="no-scrollbar relative flex gap-2 overflow-x-auto border-t border-white/10 px-4 py-3">
              {CHIPS.map((c) => (
                <button
                  key={c}
                  onClick={() => void send(c)}
                  disabled={streaming}
                  className="shrink-0 rounded-full border border-[var(--cream)]/40 bg-white/5 px-4 py-2 text-[12px] font-medium text-[var(--cream)] transition-colors hover:border-[var(--gold)] hover:bg-white/10 hover:text-[var(--gold)] disabled:opacity-40"
                >
                  {c}
                </button>
              ))}
            </div>
          )}

          {/* input */}
          <form
            onSubmit={(e) => {
              e.preventDefault()
              void send(input)
            }}
            className="relative flex items-center gap-2 border-t border-white/10 px-4 py-3"
          >
            <input
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Ej. un tinto frutal para pasta, máx $500"
              aria-label="Mensaje para la sommelier"
              disabled={streaming}
              className="h-11 flex-1 rounded-full border border-[var(--cream)]/35 bg-white/5 px-4 text-sm text-[var(--cream)] outline-none transition-colors placeholder:text-[var(--cream)]/50 focus:border-[var(--gold)] disabled:opacity-50"
            />
            <button
              type="submit"
              disabled={streaming || !input.trim()}
              aria-label="Enviar mensaje"
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[var(--wine)] text-[var(--cream)] transition-all hover:bg-[var(--wine-deep)] disabled:opacity-40"
            >
              <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
                <path d="M2.01 21 23 12 2.01 3 2 10l15 2-15 2z" />
              </svg>
            </button>
          </form>

          <p className="relative bg-black/25 px-4 py-2 text-center text-[10px] tracking-wide text-[var(--cream)]/45">
            Venta exclusiva a mayores de 18 años
          </p>
        </div>
      )}
    </>
  )
}
