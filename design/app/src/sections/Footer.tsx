import { CONTACT, NAV_LINKS, SOCIAL, WA_MAIN } from '@/lib/site'
import logo from '@/assets/logo.png'

export default function Footer() {
  return (
    <footer id="contacto" className="bg-[var(--cream)] pt-16 text-[var(--ink)] md:pt-24">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <div className="grid gap-12 pb-16 md:grid-cols-3">
          {/* contacto */}
          <div>
            <h3 className="font-display text-2xl uppercase tracking-wide">Contacto</h3>
            <ul className="mt-5 space-y-2.5 text-[15px] text-[var(--ink)]/75">
              {CONTACT.phones.map((p) => (
                <li key={p}>
                  <a href={`tel:+52${p.replace(/\s/g, '')}`} className="link-line">
                    {p}
                  </a>
                </li>
              ))}
              <li>
                <a href={WA_MAIN} target="_blank" rel="noreferrer" className="link-line">
                  WhatsApp {CONTACT.mobile}
                </a>
              </li>
            </ul>
            <div className="mt-6 flex gap-6">
              <a
                href={SOCIAL.instagram}
                target="_blank"
                rel="noreferrer"
                className="link-line text-[13px] font-semibold uppercase tracking-[0.16em]"
              >
                Instagram
              </a>
              <a
                href={SOCIAL.facebook}
                target="_blank"
                rel="noreferrer"
                className="link-line text-[13px] font-semibold uppercase tracking-[0.16em]"
              >
                Facebook
              </a>
            </div>
          </div>

          {/* horarios */}
          <div>
            <h3 className="font-display text-2xl uppercase tracking-wide">Horarios</h3>
            <ul className="mt-5 space-y-2.5 text-[15px] text-[var(--ink)]/75">
              {CONTACT.hours.map((h) => (
                <li key={h.d} className="flex justify-between gap-6 border-b border-[var(--ink)]/10 pb-2.5">
                  <span>{h.d}</span>
                  <span className="font-serif-accent text-base">{h.h}</span>
                </li>
              ))}
            </ul>
          </div>

          {/* navegación */}
          <div>
            <h3 className="font-display text-2xl uppercase tracking-wide">Menú</h3>
            <ul className="mt-5 space-y-2.5">
              {NAV_LINKS.map((l) => (
                <li key={l.href}>
                  <a href={l.href} className="link-line text-[15px] text-[var(--ink)]/75">
                    {l.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

      {/* massive logo */}
      <div className="border-t border-[var(--ink)]/10 px-5 pb-8 pt-10 md:px-10">
        <img
          src={logo}
          alt="Menguz — Vinos & Licores"
          className="mx-auto w-full max-w-4xl"
        />
        <div className="mx-auto mt-8 flex max-w-7xl flex-col items-center justify-between gap-3 text-[13px] text-[var(--ink)]/50 md:flex-row">
          <p>© 2024 Menguz · Vinos y Licores</p>
          <p className="flex gap-6">
            <a href="#contacto" className="link-line">Términos y Condiciones</a>
            <a href="#contacto" className="link-line">Política de Privacidad</a>
          </p>
        </div>
      </div>
    </footer>
  )
}
