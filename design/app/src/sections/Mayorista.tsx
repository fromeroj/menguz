import { useState } from 'react'
import type { FormEvent } from 'react'
import SplitReveal from '@/components/SplitReveal'
import Marquee from '@/components/Marquee'

export default function Mayorista() {
  const [email, setEmail] = useState('')

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const text = encodeURIComponent(
      `Hola, soy vinatería o medio mayorista. Mi correo es ${email} y quiero recibir promociones y noticias.`,
    )
    window.open(`https://wa.me/525511322231?text=${text}`, '_blank')
  }

  return (
    <section className="bg-[var(--wine)] py-20 text-[var(--cream)] md:py-28">
      <Marquee slow className="border-b border-white/15 pb-6">
        {['Mayoreo', 'Vinaterías', 'Promociones', 'Precios especiales', 'Medio mayorista'].map((w) => (
          <span key={w} className="flex items-center">
            <span className="font-display px-6 text-3xl uppercase text-white/40 md:text-5xl">{w}</span>
            <span className="h-2 w-2 rounded-full bg-[var(--cream)]/50" aria-hidden />
          </span>
        ))}
      </Marquee>

      <div className="mx-auto max-w-4xl px-5 pt-16 text-center md:px-10">
        <p className="font-serif-it text-xl md:text-2xl">Acércate a nosotros</p>
        <SplitReveal
          text="¿ERES VINATERÍA O MEDIO MAYORISTA?"
          as="h2"
          className="font-display mx-auto mt-3 max-w-3xl text-[10vw] uppercase leading-[0.95] sm:text-5xl md:text-6xl"
        />
        <p className="mx-auto mt-5 max-w-md text-[15px] leading-relaxed text-white/75">
          Regístrate y recibe nuestras promociones y noticias antes que nadie.
        </p>

        <form onSubmit={onSubmit} className="mx-auto mt-9 flex max-w-md items-end gap-4">
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="tu@correo.com"
            className="w-full border-b border-white/40 bg-transparent py-3 text-base text-[var(--cream)] placeholder-white/45 outline-none transition-colors focus:border-[var(--cream)]"
          />
          <button
            type="submit"
            className="shrink-0 rounded-full bg-[var(--cellar)] px-7 py-3.5 text-[13px] font-semibold uppercase tracking-[0.16em] text-[var(--cream)] transition-transform duration-300 hover:scale-[1.04] active:scale-95"
          >
            Suscribirme
          </button>
        </form>
      </div>
    </section>
  )
}
