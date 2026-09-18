import { useEffect, useRef } from 'react'

/**
 * Pointer + scroll parallax for absolutely positioned children
 * carrying a data-power attribute (negative = move against pointer).
 * Movement is lerped in a rAF loop for buttery motion.
 */
export function useParallax<T extends HTMLElement>() {
  const ref = useRef<T | null>(null)

  useEffect(() => {
    const root = ref.current
    if (!root) return
    const items = Array.from(
      root.querySelectorAll<HTMLElement>('[data-power]'),
    )
    if (!items.length) return

    const fine = window.matchMedia('(pointer: fine)').matches
    let tx = 0
    let ty = 0
    let cx = 0
    let cy = 0
    let scrollFactor = 0
    let raf = 0

    const onMove = (e: PointerEvent) => {
      const r = root.getBoundingClientRect()
      tx = (e.clientX - (r.left + r.width / 2)) / (r.width / 2)
      ty = (e.clientY - (r.top + r.height / 2)) / (r.height / 2)
    }

    const onScroll = () => {
      const r = root.getBoundingClientRect()
      // -1 .. 1 progress of the hero through the viewport
      scrollFactor = Math.max(
        -1,
        Math.min(1, -r.top / Math.max(r.height, 1)),
      )
    }

    const loop = () => {
      cx += (tx - cx) * 0.06
      cy += (ty - cy) * 0.06
      for (const el of items) {
        const p = parseFloat(el.dataset.power || '0')
        const rot = parseFloat(el.dataset.rotation || '0')
        const px = fine ? cx * p * 10 : 0
        const py = (fine ? cy * p * 8 : 0) + scrollFactor * p * 26
        el.style.transform = `translate3d(${px}px, ${py}px, 0) rotate(${
          rot * (fine ? cx : 0)
        }deg)`
      }
      raf = requestAnimationFrame(loop)
    }

    if (fine) root.addEventListener('pointermove', onMove)
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
    raf = requestAnimationFrame(loop)

    return () => {
      root.removeEventListener('pointermove', onMove)
      window.removeEventListener('scroll', onScroll)
      cancelAnimationFrame(raf)
    }
  }, [])

  return ref
}

/** Subtle rotation driven by page scroll (for circular product images). */
export function useScrollRotate<T extends HTMLElement>(speed = 0.04) {
  const ref = useRef<T | null>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    let raf = 0
    const update = () => {
      el.style.transform = `rotate(${window.scrollY * speed}deg)`
      raf = 0
    }
    const onScroll = () => {
      if (!raf) raf = requestAnimationFrame(update)
    }
    window.addEventListener('scroll', onScroll, { passive: true })
    update()
    return () => {
      window.removeEventListener('scroll', onScroll)
      if (raf) cancelAnimationFrame(raf)
    }
  }, [speed])

  return ref
}
