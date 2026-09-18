import { useEffect, useMemo, useRef, useState } from 'react'
import SplitReveal from '@/components/SplitReveal'

interface Product {
  n: string // name
  p: number // price
  c: string // category
  g: string // group
  i: string // image path
}

const PAGE = 24

const fmt = (p: number) =>
  `$${p.toLocaleString('es-MX', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`

/** Placeholder prices from the source catalog are quoted via WhatsApp. */
const PriceTag = ({ p }: { p: number }) =>
  p > 1 ? (
    <p className="font-serif-accent text-lg leading-none">{fmt(p)}</p>
  ) : (
    <p className="font-serif-it text-base leading-none text-[var(--wine)]">A cotizar</p>
  )

const waProduct = (name: string) =>
  `https://wa.me/525511322231?text=${encodeURIComponent(
    `Hola, me interesa ${name}. ¿Me pueden cotizar?`,
  )}`

export default function Especiales() {
  const [items, setItems] = useState<Product[]>([])
  const [query, setQuery] = useState('')
  const [cat, setCat] = useState('Todos')
  const [limit, setLimit] = useState(PAGE)
  const sentinelRef = useRef<HTMLDivElement>(null)

  /* load catalog on first scroll into the section */
  const sectionRef = useRef<HTMLElement>(null)
  const loaded = useRef(false)
  useEffect(() => {
    const el = sectionRef.current
    if (!el) return
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting) && !loaded.current) {
          loaded.current = true
          fetch('/data/products.json')
            .then((r) => r.json())
            .then((d: Product[]) => {
              // biggest categories first so the grid opens with wines & spirits
              const counts = new Map<string, number>()
              for (const p of d) counts.set(p.c, (counts.get(p.c) ?? 0) + 1)
              const sorted = [...d].sort(
                (a, b) =>
                  (counts.get(b.c) ?? 0) - (counts.get(a.c) ?? 0) ||
                  a.n.localeCompare(b.n, 'es'),
              )
              setItems(sorted)
            })
            .catch(() => setItems([]))
          io.disconnect()
        }
      },
      { rootMargin: '600px' },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [])

  const categories = useMemo(() => {
    const counts = new Map<string, number>()
    for (const p of items) counts.set(p.c, (counts.get(p.c) ?? 0) + 1)
    return ['Todos', ...[...counts.entries()].sort((a, b) => b[1] - a[1]).map(([c]) => c)]
  }, [items])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return items.filter(
      (p) =>
        (cat === 'Todos' || p.c === cat) &&
        (!q || p.n.toLowerCase().includes(q) || p.c.toLowerCase().includes(q)),
    )
  }, [items, query, cat])

  const visible = filtered.slice(0, limit)

  /* infinite scroll */
  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setLimit((l) => (l < filtered.length ? l + PAGE : l))
        }
      },
      { rootMargin: '800px' },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [filtered.length])

  /* reset pagination when filters change */
  useEffect(() => {
    setLimit(PAGE)
  }, [query, cat])

  return (
    <section id="especiales" ref={sectionRef} className="bg-[var(--cream)] pb-20 text-[var(--ink)] md:pb-28">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <div className="md:flex md:items-end md:justify-between md:gap-12">
          <div>
            <p className="font-serif-it text-xl text-[var(--wine)] md:text-2xl">
              Los favoritos de nuestros clientes
            </p>
            <SplitReveal
              text="ESPECIALES"
              as="h2"
              className="font-display mt-2 text-[16vw] uppercase leading-[0.9] sm:text-7xl lg:text-8xl"
            />
            <SplitReveal
              text="MENGUZ WEB"
              as="h2"
              delay={220}
              className="font-display text-stroke-ink text-[16vw] uppercase leading-[0.9] sm:text-7xl lg:text-8xl"
            />
          </div>
          <p className="mt-5 max-w-sm text-[15px] leading-relaxed text-[var(--ink)]/65 md:pb-3 md:text-right">
            {items.length > 0 ? `${items.length} productos en catálogo. ` : ''}
            Cotiza cualquier producto por WhatsApp y recíbelo en 24 horas.
          </p>
        </div>
      </div>

      {/* sticky toolbar: search + category chips */}
      <div className="sticky top-[68px] z-30 mt-10 border-y border-[var(--ink)]/10 bg-[var(--cream)]/95 backdrop-blur-md md:top-[80px]">
        <div className="mx-auto max-w-7xl px-5 py-3 md:px-10">
          <div className="flex items-center gap-3">
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar producto…"
              aria-label="Buscar producto"
              className="h-11 w-44 shrink-0 rounded-full border border-[var(--ink)]/20 bg-transparent px-4 text-sm outline-none transition-colors placeholder:text-[var(--ink)]/40 focus:border-[var(--wine)] sm:w-56"
            />
            <div className="no-scrollbar flex gap-2 overflow-x-auto py-1">
              {categories.map((c) => (
                <button
                  key={c}
                  onClick={() => setCat(c)}
                  className={`h-10 shrink-0 rounded-full border px-4 text-[12px] font-semibold uppercase tracking-[0.1em] transition-colors ${
                    cat === c
                      ? 'border-[var(--wine)] bg-[var(--wine)] text-[var(--cream)]'
                      : 'border-[var(--ink)]/20 text-[var(--ink)]/70 hover:border-[var(--wine)] hover:text-[var(--wine)]'
                  }`}
                >
                  {c}
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-7xl px-5 pt-8 md:px-10">
        {items.length === 0 ? (
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {Array.from({ length: 10 }).map((_, i) => (
              <div key={i} className="aspect-[3/4.4] animate-pulse rounded-xl bg-[var(--ink)]/5" />
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <p className="py-20 text-center text-[var(--ink)]/60">
            No encontramos «{query}» {cat !== 'Todos' ? `en ${cat}` : ''}. Intenta con otra búsqueda.
          </p>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-x-4 gap-y-8 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
              {visible.map((p) => (
                <article key={p.i} className="group flex flex-col">
                  <div className="relative mb-3 flex aspect-[4/5] items-center justify-center overflow-hidden rounded-xl bg-white p-4 shadow-[0_1px_2px_rgba(30,26,22,0.06)] transition-shadow duration-300 group-hover:shadow-[0_10px_28px_rgba(30,26,22,0.12)]">
                    <img
                      src={`/${p.i}`}
                      alt={p.n}
                      loading="lazy"
                      decoding="async"
                      className="max-h-full w-auto object-contain transition-transform duration-500 group-hover:scale-[1.06]"
                    />
                    <span className="absolute left-2.5 top-2.5 rounded-full bg-[var(--cream)] px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-[var(--wine)]">
                      {p.c}
                    </span>
                  </div>
                  <h4 className="text-[13px] font-medium leading-snug text-[var(--ink)] [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:2] overflow-hidden">
                    {p.n}
                  </h4>
                  <div className="mt-1.5 flex items-center justify-between gap-2">
                    <PriceTag p={p.p} />
                    <a
                      href={waProduct(p.n)}
                      target="_blank"
                      rel="noreferrer"
                      aria-label={`Cotizar ${p.n} por WhatsApp`}
                      className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-[var(--cellar)] text-[var(--cream)] transition-transform duration-300 hover:scale-110 active:scale-95"
                    >
                      <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden>
                        <path d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38a9.9 9.9 0 0 0 4.74 1.21h.01c5.46 0 9.9-4.45 9.9-9.91A9.85 9.85 0 0 0 12.04 2zm4.52 12.01c-.25-.13-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.13-.16.24-.64.8-.78.97-.14.16-.29.18-.54.06-.25-.13-1.05-.39-1.99-1.23-.74-.66-1.23-1.47-1.38-1.72-.14-.25-.01-.38.11-.51.11-.11.25-.29.37-.43.12-.14.16-.25.25-.41.08-.17.04-.31-.02-.43-.06-.13-.56-1.34-.76-1.84-.2-.48-.41-.42-.56-.43h-.48c-.17 0-.43.06-.66.31-.22.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.67-1.18.21-.58.21-1.07.15-1.18-.06-.1-.23-.16-.48-.29z" />
                      </svg>
                    </a>
                  </div>
                </article>
              ))}
            </div>

            {limit < filtered.length && (
              <div ref={sentinelRef} className="flex justify-center pt-10">
                <button
                  onClick={() => setLimit((l) => l + PAGE)}
                  className="rounded-full border border-[var(--ink)]/25 px-8 py-3.5 text-[12px] font-semibold uppercase tracking-[0.16em] text-[var(--ink)] transition-colors hover:border-[var(--wine)] hover:text-[var(--wine)]"
                >
                  Ver más ({filtered.length - limit} restantes)
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </section>
  )
}
