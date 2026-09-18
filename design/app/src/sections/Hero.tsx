import { HERO_BOTTLES, MARQUEE_ITEMS, WA_QUOTE } from '@/lib/site'
import { useParallax } from '@/hooks/useParallax'
import SplitReveal from '@/components/SplitReveal'
import Marquee from '@/components/Marquee'

const FLOATS = [
  { cls: 'left-[4%] top-[16%] w-14 md:w-24 lg:w-28', power: -3, rot: 4, anim: 'float-a', op: 'opacity-90' },
  { cls: 'right-[6%] top-[12%] w-16 md:w-28 lg:w-32', power: 2.4, rot: -5, anim: 'float-b', op: 'opacity-95' },
  { cls: 'left-[16%] bottom-[26%] w-12 md:w-20 lg:w-24', power: 3.2, rot: 6, anim: 'float-c', op: 'opacity-70' },
  { cls: 'right-[18%] bottom-[30%] w-12 md:w-24 lg:w-28 hidden sm:block', power: -2, rot: -4, anim: 'float-a', op: 'opacity-80' },
]

export default function Hero() {
  const ref = useParallax<HTMLElement>()

  return (
    <section
      id="inicio"
      ref={ref}
      className="grain relative flex min-h-[100svh] flex-col overflow-hidden bg-[var(--cellar)]"
    >
      {/* wine glow */}
      <div className="pointer-events-none absolute left-1/2 top-1/3 h-[70vmin] w-[110vmin] -translate-x-1/2 -translate-y-1/2 rounded-full bg-[radial-gradient(closest-side,rgba(131,48,63,0.5),transparent)] blur-2xl" />

      {/* floating bottle cutouts */}
      {FLOATS.map((f, i) => (
        <div
          key={i}
          className={`pointer-events-none absolute mix-blend-screen ${f.cls} ${f.op}`}
          data-power={f.power}
          data-rotation={f.rot}
        >
          <img
            src={HERO_BOTTLES[i]}
            alt=""
            className={`${f.anim} w-full`}
          />
        </div>
      ))}

      <div className="relative z-10 mx-auto flex w-full max-w-7xl flex-1 flex-col items-center justify-center px-5 pt-28 pb-16 text-center md:px-10">
        <p className="font-serif-it mb-5 text-lg text-[var(--gold)] md:text-2xl">
          Desde el corazón de la Ciudad de México
        </p>

        <h1 className="font-display leading-[0.86]">
          <SplitReveal
            text="VINOS"
            as="span"
            className="block text-[24vw] text-[var(--cream)] sm:text-[19vw] lg:text-[11rem]"
          />
          <span className="flex items-center justify-center gap-4 sm:gap-8">
            <span className="hidden h-[3px] w-16 bg-[var(--wine)] sm:block md:w-32" aria-hidden />
            <SplitReveal
              text="& LICORES"
              as="span"
              delay={200}
              className="block text-[24vw] text-transparent sm:text-[19vw] lg:text-[11rem] [-webkit-text-stroke:2px_var(--wine)]"
            />
            <span className="hidden h-[3px] w-16 bg-[var(--wine)] sm:block md:w-32" aria-hidden />
          </span>
        </h1>

        <p className="mt-7 max-w-md text-[15px] leading-relaxed text-white/70 md:max-w-xl md:text-lg">
          Las mejores marcas de vinos, tequila, mezcal, whisky y más.
          Atención personalizada y envío en 24 horas.
        </p>

        <div className="mt-9 flex flex-col items-center gap-3 sm:flex-row">
          <a
            href={WA_QUOTE}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center rounded-full bg-[var(--wine)] px-9 py-4 text-sm font-semibold uppercase tracking-[0.18em] text-[var(--cream)] transition-transform duration-300 hover:scale-[1.04] active:scale-95"
          >
            Cotiza ahora
          </a>
          <a
            href="#especiales"
            className="inline-flex items-center rounded-full border border-white/25 px-9 py-4 text-sm font-semibold uppercase tracking-[0.18em] text-[var(--cream)] transition-colors duration-300 hover:border-[var(--cream)]"
          >
            Ver especiales
          </a>
        </div>
      </div>

      {/* marquee */}
      <div className="relative z-10 border-t border-white/10 py-4">
        <Marquee>
          {MARQUEE_ITEMS.map((item) => (
            <span key={item} className="flex items-center">
              <span className="font-display px-5 text-2xl uppercase tracking-wide text-white/50 md:text-3xl">
                {item}
              </span>
              <span className="h-1.5 w-1.5 rounded-full bg-[var(--wine)]" aria-hidden />
            </span>
          ))}
        </Marquee>
      </div>
    </section>
  )
}
