import SplitReveal from '@/components/SplitReveal'

/* Static mock of the chat window — the real one is the floating
   SommelierChat widget (opens from the corner button or the CTA below). */
function ChatPreview() {
  return (
    <div className="relative w-full max-w-md overflow-hidden rounded-3xl bg-[var(--cellar)] shadow-[0_24px_60px_rgba(23,19,16,0.25)]">
      <div className="grain pointer-events-none absolute inset-0" />
      <div className="relative flex items-center gap-3 border-b border-white/10 px-5 py-4">
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-[var(--wine)] text-xl">
          🍷
        </span>
        <div>
          <p className="font-display text-sm uppercase tracking-[0.14em] text-[var(--cream)]">
            Sofía · Sommelier
          </p>
          <p className="flex items-center gap-1.5 text-[11px] text-[var(--cream)]/55">
            <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-green-500" />
            En línea · inventario real
          </p>
        </div>
        <span className="ml-auto flex h-8 w-8 items-center justify-center rounded-full text-[var(--cream)]/50">
          <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
            <path d="M18.3 5.71 12 12.01 5.7 5.71 4.29 7.12l6.3 6.3-6.3 6.3 1.41 1.41 6.3-6.3 6.3 6.3 1.41-1.41-6.3-6.3 6.3-6.3z" />
          </svg>
        </span>
      </div>
      <div className="relative flex flex-col gap-4 px-5 py-6">
        <div className="max-w-[85%] rounded-2xl rounded-bl-sm bg-[var(--cream-2)] px-4 py-3 text-sm leading-relaxed text-[var(--ink)]">
          ¡Hola! Soy Sofía 🍷 ¿Qué vas a comer o celebrar hoy?
        </div>
        <div className="flex justify-end">
          <p className="max-w-[80%] rounded-2xl rounded-br-sm bg-[var(--wine)] px-4 py-3 text-sm leading-relaxed text-[var(--cream)]">
            Una cena de aniversario, carne asada · máx $600
          </p>
        </div>
        <div className="max-w-[85%] rounded-2xl rounded-bl-sm bg-[var(--cream-2)] px-4 py-3 text-sm leading-relaxed text-[var(--ink)]">
          Con asado manda un tinto con estructura 🥩 Te tengo 5 opciones reales con precio y
          existencia…
        </div>
      </div>
      <div className="relative flex gap-2 overflow-hidden border-t border-white/10 px-5 py-3">
        {['🥩 Asado', '🎁 Regalo', '🥂 Brindis'].map((c) => (
          <span
            key={c}
            className="shrink-0 rounded-full border border-[var(--cream)]/40 bg-white/5 px-4 py-2 text-[12px] font-medium text-[var(--cream)]"
          >
            {c}
          </span>
        ))}
      </div>
    </div>
  )
}

export default function Sommelier() {
  const openChat = () => window.dispatchEvent(new CustomEvent('sommelier:open'))

  return (
    <section id="sommelier" className="bg-white py-20 text-[var(--ink)] md:py-32">
      <div className="mx-auto grid max-w-7xl items-center gap-12 px-5 md:grid-cols-2 md:px-10">
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
          <p className="mt-6 max-w-md text-[15px] leading-relaxed text-[var(--ink)]/65">
            Cuéntame qué vas a comer, celebrar o regalar y te recomiendo botellas que SÍ tenemos
            en existencia, con precio real — directo en el chat, desde cualquier página del sitio.
          </p>
          <button
            onClick={openChat}
            className="mt-8 inline-flex items-center gap-3 rounded-full bg-[var(--wine)] px-8 py-4 text-[13px] font-semibold uppercase tracking-[0.16em] text-[var(--cream)] transition-all hover:scale-[1.04] hover:bg-[var(--wine-deep)] active:scale-95"
          >
            Hablar con Sofía
            <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current" aria-hidden>
              <path d="M2.01 21 23 12 2.01 3 2 10l15 2-15 2z" />
            </svg>
          </button>
          <p className="mt-4 text-[13px] text-[var(--ink)]/50">
            También disponible en el botón inferior derecho. Venta exclusiva a mayores de 18 años.
          </p>
        </div>
        <div className="flex justify-center md:justify-end">
          <ChatPreview />
        </div>
      </div>
    </section>
  )
}
