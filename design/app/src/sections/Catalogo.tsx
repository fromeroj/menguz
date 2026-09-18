import { CATALOG_GROUPS } from '@/lib/site'
import { useInView } from '@/hooks/useInView'
import SplitReveal from '@/components/SplitReveal'

export default function Catalogo() {
  const [ref, inView] = useInView<HTMLDivElement>(0.15)

  return (
    <section id="catalogo" className="grain relative bg-[var(--cellar-2)] py-20 md:py-32">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <p className="font-serif-it text-center text-xl text-[var(--gold)] md:text-2xl">
          Todo para tu barra, tu evento o tu negocio
        </p>
        <SplitReveal
          text="CATÁLOGO"
          as="h2"
          className="font-display mt-2 text-center text-[16vw] uppercase leading-[0.9] text-[var(--cream)] sm:text-8xl"
        />

        <div ref={ref} className="mt-14 md:mt-20">
          {CATALOG_GROUPS.map((g, i) => (
            <div
              key={g.n}
              className="border-t border-white/10 py-8 transition-all duration-700 md:py-10"
              style={{
                transitionDelay: `${i * 110}ms`,
                opacity: inView ? 1 : 0,
                transform: inView ? 'translateY(0)' : 'translateY(30px)',
              }}
            >
              <div className="flex flex-col gap-4 md:flex-row md:items-start md:gap-12">
                <div className="flex items-baseline gap-5 md:w-72 md:shrink-0">
                  <span className="font-serif-it text-lg text-[var(--wine)] md:text-xl">
                    ({g.n})
                  </span>
                  <h3 className="font-display text-5xl uppercase text-[var(--cream)] md:text-6xl">
                    {g.title}
                  </h3>
                </div>
                <ul className="flex flex-wrap gap-x-2 gap-y-2.5">
                  {g.items.map((item) => (
                    <li
                      key={item}
                      className="rounded-full border border-white/15 px-4 py-1.5 text-[13px] uppercase tracking-[0.12em] text-white/65 transition-colors duration-300 hover:border-[var(--wine)] hover:text-[var(--cream)]"
                    >
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          ))}
          <div className="border-t border-white/10" />
        </div>
      </div>
    </section>
  )
}
