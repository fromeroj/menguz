import bottle1 from '@/assets/bottle1.png'
import bottle2 from '@/assets/bottle2.png'
import bottle3 from '@/assets/bottle3.png'
import bottle4 from '@/assets/bottle4.png'

export const WA_MAIN = 'https://wa.me/525511322231'
export const WA_QUOTE =
  'https://wa.me/525511322231?text=Hola%2C%20quiero%20cotizar%20un%20pedido'
export const WA_EVENT =
  'https://wa.me/525511322231?text=Hola%2C%20quiero%20cotizar%20para%20un%20evento'
export const WA_WHOLESALE =
  'https://wa.me/525511322231?text=Hola%2C%20soy%20vinater%C3%ADa%20o%20medio%20mayorista%20y%20quiero%20informaci%C3%B3n'

export const SOCIAL = {
  facebook: 'https://www.facebook.com/Menguz-Vinos-y-Licores-109106668405994/',
  instagram: 'https://www.instagram.com/menguz_vl/',
}

export const CONTACT = {
  phones: ['55 7098 3876', '55 7098 3913'],
  mobile: '55 3483 6109',
  hours: [
    { d: 'Lunes a viernes', h: '9:00 – 18:30 hrs' },
    { d: 'Sábados', h: '9:00 – 13:30 hrs' },
  ],
}

export interface Product {
  name: string
  category: string
  price: string
  img: string
}

export const PRODUCTS: Product[] = [
  { name: 'Burgundy', category: 'Vino Tinto', price: '$229.50', img: bottle1 },
  { name: 'Red Wine', category: 'Vino Tinto', price: '$299.70', img: bottle2 },
  { name: 'Goose Berry', category: 'Vino Blanco', price: '$299.70', img: bottle3 },
  { name: 'La Katina', category: 'Vino Tinto', price: '$259.70', img: bottle4 },
]

export const HERO_BOTTLES = [bottle1, bottle3, bottle4, bottle2]

export const CATALOG_GROUPS = [
  {
    n: '01',
    title: 'Vinos',
    items: ['Tinto', 'Blanco', 'Rosado', 'Espumoso', 'Champagne'],
  },
  {
    n: '02',
    title: 'Licores',
    items: [
      'Tequila', 'Mezcal', 'Whisky', 'Ron', 'Vodka', 'Ginebra', 'Brandy',
      'Cognac', 'Anís', 'Rompope', 'Aguardiente', 'Aperitivo', 'Destilado',
      'Licor', 'Cooler', 'Jerez', 'Oporto', 'Sidra', 'Sangrita',
    ],
  },
  {
    n: '03',
    title: 'Mezcladores',
    items: ['Jarabe', 'Refresco', 'Energizante', 'Agua', 'Jugo'],
  },
  {
    n: '04',
    title: 'Abarrotes',
    items: ['Abarrotes', 'Quesos'],
  },
]

export const MARQUEE_ITEMS = [
  'Vino Tinto', 'Tequila', 'Mezcal', 'Whisky', 'Ron', 'Vodka',
  'Vino Blanco', 'Espumoso', 'Ginebra', 'Cerveza', 'Rompope', 'Cognac',
]

export const NAV_LINKS = [
  { label: 'Inicio', href: '#inicio' },
  { label: 'Promociones', href: '#promociones' },
  { label: 'Especiales', href: '#especiales' },
  { label: 'Producto del mes', href: '#destacado' },
  { label: 'Sommelier', href: '#sommelier' },
  { label: 'Catálogo', href: '#catalogo' },
  { label: 'Eventos', href: '#eventos' },
  { label: 'Contacto', href: '#contacto' },
]

export const waProduct = (name: string) =>
  `https://wa.me/525511322231?text=${encodeURIComponent(
    `Hola, me interesa ${name}. ¿Me pueden cotizar?`,
  )}`
