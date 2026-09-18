import { useInView } from '@/hooks/useInView'

interface Props {
  src: string
  alt: string
  className?: string
  imgClassName?: string
}

/** Image that wipes in with a clip-path inset reveal. */
export default function RevealImage({ src, alt, className = '', imgClassName = '' }: Props) {
  const [ref, inView] = useInView<HTMLDivElement>(0.3)
  return (
    <div ref={ref} className={`clip-reveal overflow-hidden ${inView ? 'in' : ''} ${className}`}>
      <img src={src} alt={alt} loading="lazy" className={`h-full w-full object-cover ${imgClassName}`} />
    </div>
  )
}
