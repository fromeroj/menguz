import { useCallback, useEffect, useRef, useState } from 'react'
import promo1 from '@/assets/promo1.jpg'
import promo2 from '@/assets/promo2.jpg'
import promo3 from '@/assets/promo3.jpg'
import promo4 from '@/assets/promo4.jpg'
import { fetchCampanas, waLink, type Campana } from '@/lib/api'

/* Static fallback banners (used before the backend responds or if it fails) */
const STATIC_PROMOS = [
  { img: promo1, label: 'Kit Karaoke' },
  { img: promo2, label: 'Combo Verano' },
  { img: promo3, label: 'Don Julio 70 · Edición Mundial' },
  { img: promo4, label: 'Don Julio 1942 · Edición Mundial' },
]

const FALLBACK_IMGS = [promo1, promo2, promo3, promo4]

interface PromoSlide {
  img: string
  label: string
  wa: string
}

/** Live campaigns from the admin panel (Editor de Campañas); falls back to
 * the static banners above when the backend is unreachable. */
function usePromos(): PromoSlide[] {
  const [slides, setSlides] = useState<PromoSlide[] | null>(null)

  useEffect(() => {
    let alive = true
    fetchCampanas('carousel').then((camps: Campana[]) => {
      if (!alive) return
      if (camps.length === 0) {
        setSlides(
          STATIC_PROMOS.map((p) => ({
            ...p,
            wa: waLink(`Hola, me interesa ${p.label}. ¿Me pueden cotizar?`),
          })),
        )
        return
      }
      setSlides(
        camps.map((c, i) => ({
          img: c.imagen_url || FALLBACK_IMGS[i % FALLBACK_IMGS.length],
          label: c.titulo,
          wa: c.cta_url && c.cta_url.startsWith('http')
            ? c.cta_url
            : waLink(c.wa_mensaje || `Hola, me interesa ${c.titulo}. ¿Me pueden cotizar?`),
        })),
      )
    })
    return () => {
      alive = false
    }
  }, [])

  return slides ?? STATIC_PROMOS.map((p) => ({
    ...p,
    wa: waLink(`Hola, me interesa ${p.label}. ¿Me pueden cotizar?`),
  }))
}

const AUTOPLAY_MS = 5200

export default function Promociones() {
  const trackRef = useRef<HTMLDivElement>(null)
  const [active, setActive] = useState(0)
  const PROMOS = usePromos()
  const paused = useRef(false)
  const resumeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const goTo = useCallback((i: number) => {
    const track = trackRef.current
    if (!track) return
    const w = track.clientWidth
    track.scrollTo({ left: i * w, behavior: 'smooth' })
  }, [])

  /* autoplay */
  useEffect(() => {
    const id = setInterval(() => {
      if (paused.current) return
      setActive((a) => {
        const n = trackRef.current?.children.length || PROMOS.length
        const next = (a + 1) % n
        goTo(next)
        return next
      })
    }, AUTOPLAY_MS)
    return () => clearInterval(id)
  }, [goTo, PROMOS.length])

  /* keep dots in sync with manual swipes */
  useEffect(() => {
    const track = trackRef.current
    if (!track) return
    let raf = 0
    const onScroll = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(() => {
        const i = Math.round(track.scrollLeft / track.clientWidth)
        setActive((a) => (i === a ? a : i))
      })
    }
    track.addEventListener('scroll', onScroll, { passive: true })
    return () => {
      track.removeEventListener('scroll', onScroll)
      cancelAnimationFrame(raf)
    }
  }, [])

  const pauseTemporarily = () => {
    paused.current = true
    clearTimeout(resumeTimer.current)
    resumeTimer.current = setTimeout(() => {
      paused.current = false
    }, 9000)
  }

  return (
    <section id="promociones" className="bg-[var(--cellar)] pb-16 pt-4 md:pb-24">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <div className="mb-6 flex items-end justify-between">
          <h2 className="font-display text-4xl uppercase text-[var(--cream)] md:text-5xl">
            Promociones <span className="font-serif-it lowercase text-[var(--gold)]">vigentes</span>
          </h2>
          <div className="hidden gap-3 md:flex">
            <button
              aria-label="Promoción anterior"
              onClick={() => { pauseTemporarily(); goTo((active - 1 + PROMOS.length) % PROMOS.length) }}
              className="flex h-11 w-11 items-center justify-center rounded-full border border-white/25 text-[var(--cream)] transition-colors hover:border-[var(--cream)]"
            >
              ←
            </button>
            <button
              aria-label="Siguiente promoción"
              onClick={() => { pauseTemporarily(); goTo((active + 1) % PROMOS.length) }}
              className="flex h-11 w-11 items-center justify-center rounded-full border border-white/25 text-[var(--cream)] transition-colors hover:border-[var(--cream)]"
            >
              →
            </button>
          </div>
        </div>

        {/* carousel */}
        <div
          ref={trackRef}
          onPointerDown={pauseTemporarily}
          className="snap-row no-scrollbar relative flex overflow-x-auto rounded-2xl"
        >
          {PROMOS.map((p, i) => (
            <div
              key={i}
              className="relative h-[440px] w-full shrink-0 overflow-hidden rounded-2xl sm:h-[480px] md:h-[540px] lg:h-[620px]"
            >
              {/* blurred backdrop (mobile framing) */}
              <img
                src={p.img}
                alt=""
                aria-hidden
                className="absolute inset-0 h-full w-full scale-125 object-cover blur-2xl brightness-[0.4]"
              />
              {/* banner: contained on mobile so no text is cropped, full-bleed on desktop */}
              <img
                src={p.img}
                alt={`Promoción: ${p.label}`}
                loading={i === 0 ? 'eager' : 'lazy'}
                className="relative h-full w-full object-contain md:object-cover"
              />
              {/* overlay CTA */}
              <div className="absolute inset-x-0 bottom-0 flex items-center justify-between gap-3 bg-gradient-to-t from-black/70 via-black/25 to-transparent p-4 md:p-7">
                <span className="rounded-full border border-white/30 px-4 py-1.5 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/90 backdrop-blur-sm">
                  {p.label}
                </span>
                <a
                  href={p.wa}
                  target="_blank"
                  rel="noreferrer"
                  className="rounded-full bg-[var(--wine)] px-6 py-3 text-[12px] font-semibold uppercase tracking-[0.16em] text-[var(--cream)] transition-transform duration-300 hover:scale-[1.05] active:scale-95"
                >
                  Cotiza ahora
                </a>
              </div>
            </div>
          ))}
        </div>

        {/* dots */}
        <div className="mt-5 flex justify-center gap-2.5">
          {PROMOS.map((_, i) => (
            <button
              key={i}
              aria-label={`Ir a promoción ${i + 1}`}
              onClick={() => { pauseTemporarily(); goTo(i) }}
              className={`h-2 rounded-full transition-all duration-400 ${
                active === i ? 'w-8 bg-[var(--wine)]' : 'w-2 bg-white/30'
              }`}
            />
          ))}
        </div>
      </div>
    </section>
  )
}
