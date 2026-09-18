import { useEffect, useState } from 'react'
import { NAV_LINKS, CONTACT, SOCIAL } from '@/lib/site'
import { useParallax } from '@/hooks/useParallax'
import logo from '@/assets/logo.png'
import logoCream from '@/assets/logo-cream.png'
import bottle from '@/assets/bottle3.png'

export default function Header() {
  const [scrolled, setScrolled] = useState(false)
  const [open, setOpen] = useState(false)
  const menuRef = useParallax<HTMLDivElement>()

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 32)
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    document.body.style.overflow = open ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [open])

  return (
    <>
      <header
        className={`fixed inset-x-0 top-0 z-50 transition-all duration-500 ${
          scrolled ? 'header-glass' : 'bg-[var(--cream)]'
        }`}
      >
        <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-3 md:px-10">
          <a href="#inicio" aria-label="Menguz inicio" className="relative z-[60]">
            <img src={logo} alt="Menguz — Vinos & Licores" className="h-11 w-auto md:h-14" />
          </a>

          {/* desktop nav */}
          <nav className="hidden items-center gap-8 lg:flex">
            {NAV_LINKS.map((l) => (
              <a
                key={l.href}
                href={l.href}
                className="link-line text-[13px] font-medium uppercase tracking-[0.14em] text-[var(--ink)]"
              >
                {l.label}
              </a>
            ))}
          </nav>

          {/* hamburger */}
          <button
            onClick={() => setOpen(!open)}
            aria-label={open ? 'Cerrar menú' : 'Abrir menú'}
            aria-expanded={open}
            className="relative z-[60] flex h-11 w-11 flex-col items-center justify-center gap-[7px] lg:hidden"
          >
            <span
              className={`h-[2px] w-7 transition-all duration-300 ${
                open ? 'translate-y-[4.5px] rotate-45 bg-[var(--cream)]' : 'bg-[var(--ink)]'
              }`}
            />
            <span
              className={`h-[2px] w-7 transition-all duration-300 ${
                open ? '-translate-y-[4.5px] -rotate-45 bg-[var(--cream)]' : 'bg-[var(--ink)]'
              }`}
            />
          </button>
        </div>
      </header>

      {/* full-screen overlay menu */}
      <div
        ref={menuRef}
        className={`fixed inset-0 z-40 flex flex-col bg-[var(--cellar)] transition-[clip-path] duration-700 ease-[cubic-bezier(0.22,1,0.36,1)] lg:hidden ${
          open ? '[clip-path:inset(0_0_0_0)]' : 'pointer-events-none [clip-path:inset(0_0_100%_0)]'
        }`}
      >
        <div className="grain pointer-events-none absolute inset-0" />
        <div
          className="pointer-events-none absolute -right-10 top-24 w-36 opacity-60 mix-blend-screen"
          data-power="2"
        >
          <img src={bottle} alt="" className="float-b w-full" />
        </div>
        <div
          className="pointer-events-none absolute -left-8 bottom-40 w-24 opacity-40 mix-blend-screen"
          data-power="-2"
        >
          <img src={bottle} alt="" className="float-c w-full scale-x-[-1]" />
        </div>

        <nav className="relative z-10 mt-28 flex flex-1 flex-col px-8">
          {NAV_LINKS.map((l, i) => (
            <a
              key={l.href}
              href={l.href}
              onClick={() => setOpen(false)}
              className="group flex items-baseline gap-4 border-b border-white/10 py-4"
              style={{
                transitionDelay: `${open ? 150 + i * 70 : 0}ms`,
                transform: open ? 'translateY(0)' : 'translateY(24px)',
                opacity: open ? 1 : 0,
                transition: 'transform .6s cubic-bezier(0.22,1,0.36,1), opacity .5s ease',
              }}
            >
              <span className="font-serif-it text-sm text-[var(--gold)]">0{i + 1}</span>
              <span className="font-display text-5xl uppercase text-[var(--cream)] transition-colors group-active:text-[var(--wine)]">
                {l.label}
              </span>
            </a>
          ))}
        </nav>

        <div className="relative z-10 flex items-end justify-between px-8 pb-10">
          <div>
            <img src={logoCream} alt="Menguz" className="mb-3 w-32 opacity-90" />
            <p className="text-sm text-white/60">{CONTACT.phones.join(' · ')}</p>
          </div>
          <div className="flex gap-5 text-sm text-white/70">
            <a href={SOCIAL.instagram} target="_blank" rel="noreferrer" className="link-line">
              IG
            </a>
            <a href={SOCIAL.facebook} target="_blank" rel="noreferrer" className="link-line">
              FB
            </a>
          </div>
        </div>
      </div>
    </>
  )
}
