import RevealImage from '@/components/RevealImage'
import SplitReveal from '@/components/SplitReveal'
import { waProduct } from '@/lib/site'
import casadragones from '@/assets/casadragones.jpg'

export default function CasaDragones() {
  return (
    <section className="bg-[var(--cream)] py-20 text-[var(--ink)] md:py-32">
      <div className="mx-auto max-w-7xl px-5 md:px-10">
        <div className="flex flex-col-reverse items-center gap-12 lg:flex-row lg:gap-20">
          <div className="text-center lg:text-left">
            <p className="font-serif-it text-xl text-[var(--wine)] md:text-2xl">Tequila</p>
            <SplitReveal
              text="CASA"
              as="h2"
              className="font-display mt-2 text-[15vw] leading-[0.9] uppercase sm:text-7xl lg:text-8xl"
            />
            <SplitReveal
              text="DRAGONES"
              as="h2"
              delay={200}
              className="font-display text-stroke-ink text-[15vw] leading-[0.9] uppercase sm:text-7xl lg:text-8xl"
            />
            <p className="font-serif-accent mt-4 text-2xl text-[var(--ink)]/80">
              Añejo · 100% Agave Azul
            </p>
            <p className="mx-auto mt-5 max-w-lg text-[15px] leading-relaxed text-[var(--ink)]/65 lg:mx-0 md:text-base">
              Añejado en dos barricas diferentes —roble francés y roble americano
              nuevas— seleccionadas por sus sabores y características singulares.
              Un maridaje artesanal que celebra notas de agave elegantes y
              suaves, con un perfil infinitamente rico y matizado.
            </p>
            <a
              href={waProduct('Tequila Casa Dragones Añejo')}
              target="_blank"
              rel="noreferrer"
              className="link-line mt-8 inline-block text-sm font-semibold uppercase tracking-[0.18em] text-[var(--wine)]"
            >
              Cotizar esta botella
            </a>
          </div>

          <RevealImage
            src={casadragones}
            alt="Tequila Casa Dragones Añejo"
            className="aspect-square w-full max-w-md shrink-0 lg:max-w-lg"
          />
        </div>
      </div>
    </section>
  )
}
