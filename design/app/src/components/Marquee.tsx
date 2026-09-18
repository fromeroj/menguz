import type { ReactNode } from 'react'

interface Props {
  children: ReactNode
  className?: string
  slow?: boolean
}

/** Infinite horizontal marquee (duplicated track, seamless loop). */
export default function Marquee({ children, className = '', slow = false }: Props) {
  return (
    <div className={`overflow-hidden ${className}`} aria-hidden>
      <div className={`marquee-track ${slow ? 'slow' : ''} flex w-max items-center`}>
        <div className="flex items-center shrink-0">{children}</div>
        <div className="flex items-center shrink-0">{children}</div>
      </div>
    </div>
  )
}
