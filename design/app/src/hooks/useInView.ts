import { useEffect, useRef, useState } from 'react'

/** Observes an element once; returns [ref, inView]. */
export function useInView<T extends HTMLElement>(threshold = 0.25) {
  const ref = useRef<T | null>(null)
  const [inView, setInView] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    // Immediate fallback: if already in the viewport at mount, reveal now
    const r = el.getBoundingClientRect()
    if (r.top < window.innerHeight * 0.95 && r.bottom > 0) {
      setInView(true)
      return
    }
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            setInView(true)
            io.disconnect()
          }
        })
      },
      { threshold },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [threshold])

  return [ref, inView] as const
}
