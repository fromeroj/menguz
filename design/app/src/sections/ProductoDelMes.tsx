import RevealImage from '@/components/RevealImage'
import SplitReveal from '@/components/SplitReveal'
import { useInView } from '@/hooks/useInView'
import { WA_QUOTE } from '@/lib/site'
import { fetchCampanas, waLink } from '@/lib/api'
import { useEffect, useState } from 'react'
import donjulio from '@/assets/donjulio.jpg'

/** Headline words rendered with the split animation. The live "Producto del
 * mes" campaign (from the admin panel) overrides the static copy. */
function headline(titulo: string): [string, string] {
  const words = titulo.trim().split(/\s+/)
  if (words.length === 1) return [words[0].toUpperCase(), '']
  return [words[0].toUpperCase(), words.slice(1).join(' ').toUpperCase()]
}

export default function ProductoDelMes() {
  const [ref, inView] = useInView<HTMLDivElement>(0.35)
  const [camp, setCamp] = useState<Awaited<ReturnType<typeof fetchCampanas>>[number] | null>(null)

  useEffect(() => {
    let alive = true
    fetchCampanas('producto_mes').then((camps) => {
      if (alive && camps.length > 0) setCamp(camps[0])
    })
    return () => {
      alive = false
    }
  }, [])

  const [word1, word2] = camp ? headline(camp.titulo) : ['DON JULIO', 'CENIZA']
  const descripcion = camp?.descripcion ||
    'El nuevo tequila de Don Julio cuenta con un perfil único de sabor suave, dulce y ahumado. Una sofisticada mezcla de roble carbonizado, cacao oscuro aterciopelado y suaves notas de humo, perfectamente equilibrado con agave tostado. Ideal solo o en cocteles como la nueva Paloma Negra.'
  const cta = camp
    ? waLink(camp.wa_mensaje || `Hola, me interesa ${camp.titulo}. ¿Me pueden cotizar?`)
    : WA_QUOTE
  const precio = camp?.precio_oferta
    ? `$${camp.precio_oferta.toLocaleString('es-MX', { minimumFractionDigits: 2 })}`
    : null

  return (
    <section id="destacado" className="grain relative overflow-hidden bg-[var(--cellar)] py-20 md:py-32">
      <div className="pointer-events-none absolute right-0 top-0 h-[50vmin] w-[80vmin] rounded-full bg-[radial-gradient(closest-side,rgba(131,48,63,0.35),transparent)] blur-2xl" />

      {/* animated wine-glass line drawing */}
      <div ref={ref} className={`draw-svg ${inView ? 'in' : ''} pointer-events-none absolute left-4 top-8 w-20 opacity-50 md:left-16 md:w-28`}>
        <svg viewBox="0 0 100 220" fill="none" stroke="var(--gold)" strokeWidth="1.5">
          <path d="M20 12 C20 70 34 96 50 96 C66 96 80 70 80 12" />
          <path className="d2" d="M14 12 L86 12" />
          <path className="d2" d="M50 96 L50 190" />
          <path className="d3" d="M24 206 C24 190 76 190 76 206" />
          <path className="d3" d="M30 42 C40 52 60 52 70 42" stroke="var(--wine)" />
        </svg>
      </div>

      <div className="relative mx-auto max-w-7xl px-5 md:px-10">
        <div className="flex flex-col items-center gap-12 lg:flex-row lg:gap-20">
          <RevealImage
            src={donjulio}
            alt="Tequila Don Julio Ceniza"
            className="aspect-square w-full max-w-md shrink-0 lg:max-w-lg"
          />

          <div className="text-center lg:text-left">
            <p className="font-serif-it text-xl text-[var(--gold)] md:text-2xl">
              Producto del mes
              {precio && (
                <span className="ml-3 rounded-full border border-[var(--gold)] px-3 py-0.5 text-sm not-italic text-[var(--gold)]">
                  {precio}
                </span>
              )}
            </p>
            <SplitReveal
              text={word1}
              as="h2"
              className="font-display mt-2 text-[15vw] leading-[0.9] uppercase text-[var(--cream)] sm:text-7xl lg:text-8xl"
            />
            {word2 && (
              <SplitReveal
                text={word2}
                as="h2"
                delay={200}
                className="font-display text-stroke-cream text-[15vw] leading-[0.9] uppercase sm:text-7xl lg:text-8xl"
              />
            )}
            <p className="mt-6 max-w-lg text-[15px] leading-relaxed text-white/70 md:text-base">
              {descripcion}
            </p>
            <a
              href={cta}
              target="_blank"
              rel="noreferrer"
              className="mt-8 inline-flex items-center rounded-full bg-[var(--cream)] px-9 py-4 text-sm font-semibold uppercase tracking-[0.18em] text-[var(--ink)] transition-transform duration-300 hover:scale-[1.04] active:scale-95"
            >
              {camp?.cta_texto || 'Cotiza ahora'}
            </a>
          </div>
        </div>
      </div>
    </section>
  )
}
