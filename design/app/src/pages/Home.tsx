import Header from '@/sections/Header'
import Hero from '@/sections/Hero'
import Promociones from '@/sections/Promociones'
import ValueProps from '@/sections/ValueProps'
import Especiales from '@/sections/Especiales'
import ProductoDelMes from '@/sections/ProductoDelMes'
import Sommelier from '@/sections/Sommelier'
import CasaDragones from '@/sections/CasaDragones'
import Catalogo from '@/sections/Catalogo'
import Eventos from '@/sections/Eventos'
import Mayorista from '@/sections/Mayorista'
import Footer from '@/sections/Footer'
import WhatsAppFab from '@/sections/WhatsAppFab'

export default function Home() {
  return (
    <main>
      <Header />
      <Hero />
      <Promociones />
      <ValueProps />
      <Especiales />
      <ProductoDelMes />
      <Sommelier />
      <CasaDragones />
      <Catalogo />
      <Eventos />
      <Mayorista />
      <Footer />
      <WhatsAppFab />
    </main>
  )
}
