import { useEffect, useRef, useState } from 'react'
import SplitReveal from '@/components/SplitReveal'

/* ------------------------------------------------------------------ */
/* types                                                               */
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

/* ------------------------------------------------------------------ */
/* constants                                                           */
/* ------------------------------------------------------------------ */

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
/* product image with graceful fallback                                */
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

/* ------------------------------------------------------------------ */
/* recommendation card                                                 */
/* ------------------------------------------------------------------ */

function RecCard({ rec }: { rec: Rec }) {
  const meta = [rec.bodega, rec.uvas ?? rec.anada].filter(Boolean).join(' · ')
  return (
    <a
      href={waRec(rec)}
      target="_blank"
      rel="noreferrer"
      data-testid="sommelier-card"
      className="group block w-40 shrink-0 overflow-hidden rounded-xl bg-[var(--cream)] text-[var(--ink)] shadow-[0_1px_2px_rgba(0,0,0,0.25)] transition-transform duration-300 hover:-translate-y-1 sm:w-44"
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
/* section                                                             */
/* ------------------------------------------------------------------ */

export default function Sommelier() {
  const [msgs, setMsgs] = useState<Msg[]>([
    { id: nextId++, role: 'assistant', text: WELCOME, recs: [], streaming: false, error: false },
  ])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [started, setStarted] = useState(false)
  const sessionId = useRef<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const currentId = useRef<number | null>(null)

  /* auto-scroll to newest content */
  useEffect(() => {
    const el = scrollRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [msgs])

  const patchCurrent = (fn: (m: Msg) => Msg) => {
    /* capture the id now: updaters may run after send() has already
       reset currentId.current (all events can arrive in one chunk) */
    const id = currentId.current
    if (id == null) return
    setMsgs((prev) => prev.map((m) => (m.id === id ? fn(m) : m)))
  }

  const handleEvent = (event: string, data: string) => {
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
  }

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
    <section id="sommelier" className="bg-[var(--cream)] py-20 text-[var(--ink)] md:py-32">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        {/* intro */}
        <div className="md:flex md:items-end md:justify-between md:gap-12">
          <div>
            <p className="font-serif-it text-xl text-[var(--wine)] md:text-2xl">
              Tu asesora personal de vinos
            </p>
            <SplitReveal
              text="EL SOMMELIER"
              as="h2"
              className="font-display mt-2 text-[14vw] uppercase leading-[0.9] sm:text-7xl lg:text-8xl"
            />
            <SplitReveal
              text="DIGITAL"
              as="h2"
              delay={220}
              className="font-display text-stroke-ink text-[14vw] uppercase leading-[0.9] sm:text-7xl lg:text-8xl"
            />
          </div>
          <p className="mt-5 max-w-sm text-[15px] leading-relaxed text-[var(--ink)]/65 md:pb-3 md:text-right">
            Cuéntame qué vas a comer, celebrar o regalar y te recomiendo botellas
            que SÍ tenemos en existencia, con precio real.
          </p>
        </div>

        {/* chat panel */}
        <div className="relative mt-12 overflow-hidden rounded-3xl bg-[var(--cellar)] shadow-[0_24px_60px_rgba(23,19,16,0.25)]">
          <div className="grain pointer-events-none absolute inset-0" />
          <div className="relative flex flex-col">
            {/* messages */}
            <div
              ref={scrollRef}
              className="no-scrollbar flex max-h-[420px] min-h-[280px] flex-col gap-4 overflow-y-auto px-5 py-6 md:px-8"
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
                      <p className="mb-1.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-[var(--gold)]">
                        Sofía · Sommelier
                      </p>
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

            {/* quick prompts */}
            {!started && (
              <div className="no-scrollbar flex gap-2 overflow-x-auto border-t border-white/10 px-5 py-3 md:px-8">
                {CHIPS.map((c) => (
                  <button
                    key={c}
                    onClick={() => void send(c)}
                    disabled={streaming}
                    className="shrink-0 rounded-full border border-white/20 px-4 py-2 text-[12px] text-[var(--cream)]/85 transition-colors hover:border-[var(--gold)] hover:text-[var(--gold)] disabled:opacity-40"
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
              className="flex items-center gap-3 border-t border-white/10 px-5 py-4 md:px-8"
            >
              <input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Ej. quiero un tinto frutal para pasta, máx $500"
                aria-label="Mensaje para la sommelier"
                disabled={streaming}
                className="h-12 flex-1 rounded-full border border-white/20 bg-transparent px-5 text-sm text-[var(--cream)] outline-none transition-colors placeholder:text-[var(--cream)]/40 focus:border-[var(--gold)] disabled:opacity-50"
              />
              <button
                type="submit"
                disabled={streaming || !input.trim()}
                className="flex h-12 shrink-0 items-center gap-2 rounded-full bg-[var(--wine)] px-6 text-[12px] font-semibold uppercase tracking-[0.14em] text-[var(--cream)] transition-all hover:bg-[var(--wine-deep)] disabled:opacity-40"
              >
                Enviar
                <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden>
                  <path d="M2.01 21 23 12 2.01 3 2 10l15 2-15 2z" />
                </svg>
              </button>
            </form>
          </div>
        </div>

        <p className="mt-4 text-center text-[13px] text-[var(--ink)]/50">
          Sofía, nuestra sommelier digital, recomienda solo del inventario real.
          Venta exclusiva a mayores de 18 años.
        </p>
      </div>
    </section>
  )
}
