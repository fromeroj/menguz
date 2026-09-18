import type { CSSProperties, ElementType, ReactNode } from 'react'
import { useInView } from '@/hooks/useInView'

interface Props {
  text: string
  as?: ElementType
  className?: string
  /** ms before the whole reveal starts */
  delay?: number
}

/**
 * Splits text into words/chars and reveals them with a 3D rotateX
 * character animation once scrolled into view.
 */
export default function SplitReveal({
  text,
  as: Tag = 'span',
  className = '',
  delay = 0,
}: Props) {
  const [ref, inView] = useInView<HTMLElement>(0.3)
  const words = text.split(' ')
  let charIndex = 0

  const content: ReactNode = words.map((word, wi) => (
    <span className="word" key={wi} style={{ '--word-index': wi } as CSSProperties}>
      {Array.from(word).map((c, ci) => {
        const idx = charIndex++
        return (
          <span
            className="char"
            key={ci}
            style={
              {
                '--char-index': idx,
                transitionDelay: `${delay + idx * 28}ms`,
              } as CSSProperties
            }
          >
            {c}
          </span>
        )
      })}
      {wi < words.length - 1 ? '\u00A0' : ''}
    </span>
  ))

  return (
    <Tag
      ref={ref}
      className={`char-reveal ${inView ? 'in' : ''} ${className}`}
      aria-label={text}
    >
      {content}
    </Tag>
  )
}
