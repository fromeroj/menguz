import RevealImage from '@/components/RevealImage'
import SplitReveal from '@/components/SplitReveal'
import { WA_EVENT } from '@/lib/site'
import event1 from '@/assets/event1.jpg'
import event2 from '@/assets/event2.jpg'
import event3 from '@/assets/event3.jpg'
import event4 from '@/assets/event4.jpg'

const EVENTS = [event1, event2, event3, event4]

export default function Eventos() {
  return (
    <section id="eventos" className="bg-[var(--cellar)] py-20 md:py-32">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="font-serif-it text-xl text-[var(--gold)] md:text-2xl">
              Bodas y eventos privados
            </p>
            <SplitReveal
              text="SERVICIOS"
              as="h2"
              className="font-display mt-2 text-[14vw] uppercase leading-[0.9] text-[var(--cream)] sm:text-7xl lg:text-8xl"
            />
            <SplitReveal
              text="PARA EVENTOS"
              as="h2"
              delay={200}
              className="font-display text-stroke-cream text-[14vw] uppercase leading-[0.9] sm:text-7xl lg:text-8xl"
            />
          </div>
          <p className="max-w-sm text-[15px] leading-relaxed text-white/65 md:pb-3 md:text-right">
            Para tu evento, surtimos las botellas que necesitas.
            Consulta el presupuesto con nosotros.
          </p>
        </div>
      </div>

      {/* mobile: snap slider / desktop: 4-col grid */}
      <div className="snap-row no-scrollbar mt-12 flex gap-4 overflow-x-auto px-5 pb-2 md:px-10 lg:mx-auto lg:grid lg:max-w-7xl lg:grid-cols-4 lg:overflow-visible">
        {EVENTS.map((src, i) => (
          <RevealImage
            key={i}
            src={src}
            alt={`Evento Menguz ${i + 1}`}
            className="aspect-[3/4] w-[72vw] shrink-0 sm:w-[46vw] lg:w-auto"
          />
        ))}
      </div>

      <div className="mx-auto mt-12 max-w-7xl px-5 text-center md:px-10">
        <a
          href={WA_EVENT}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center rounded-full bg-[var(--wine)] px-10 py-4 text-sm font-semibold uppercase tracking-[0.18em] text-[var(--cream)] transition-transform duration-300 hover:scale-[1.04] active:scale-95"
        >
          Cotizar evento
        </a>
      </div>
    </section>
  )
}
