import { useInView } from '@/hooks/useInView'

const PROPS = [
  { n: '01', title: 'Atención Personalizada', text: 'Nuestro servicio es el mejor del mercado.' },
  { n: '02', title: 'Calidad Asegurada', text: 'Trabajamos con las mejores marcas.' },
  { n: '03', title: 'Envío Gratis', text: 'Recibe tus productos en 24 hrs.' },
]

export default function ValueProps() {
  const [ref, inView] = useInView<HTMLDivElement>(0.2)

  return (
    <section className="bg-[var(--cream)] text-[var(--ink)]">
      <div ref={ref} className="mx-auto max-w-7xl px-5 py-16 md:px-10 md:py-24">
        {PROPS.map((p, i) => (
          <div
            key={p.n}
            className="group flex flex-col gap-2 border-t border-[var(--ink)]/15 py-7 transition-all duration-700 md:flex-row md:items-baseline md:gap-12 md:py-9"
            style={{
              transitionDelay: `${i * 120}ms`,
              opacity: inView ? 1 : 0,
              transform: inView ? 'translateY(0)' : 'translateY(28px)',
            }}
          >
            <span className="font-serif-it text-xl text-[var(--wine)] md:w-16 md:text-2xl">
              ({p.n})
            </span>
            <h3 className="font-display flex-1 text-4xl uppercase md:text-6xl">
              {p.title}
            </h3>
            <p className="font-serif-accent text-xl text-[var(--ink)]/70 md:w-72 md:text-right md:text-2xl">
              {p.text}
            </p>
          </div>
        ))}
        <div className="border-t border-[var(--ink)]/15" />
      </div>
    </section>
  )
}
