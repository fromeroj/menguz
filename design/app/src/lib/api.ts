// Thin client for the menguz-server backend.
// In production the Go server serves the storefront, so relative paths work;
// in dev, vite proxies /api and /data to the backend (see vite.config.ts).

export interface Campana {
  id: string
  titulo: string
  tipo: 'carousel' | 'producto_mes' | 'especial'
  imagen_url: string
  descripcion: string
  cve_art?: string
  precio_oferta?: number
  cta_texto?: string
  cta_url?: string
  wa_mensaje?: string
  inicio: string
  fin: string
  activa: boolean
}

export const WA_NUMBER = '525511322231'

export const waLink = (message: string) =>
  `https://wa.me/${WA_NUMBER}?text=${encodeURIComponent(message)}`

/** Fetches active campaigns by type; resolves [] on any failure so the
 * storefront always renders with its static fallback content. */
export async function fetchCampanas(tipo: string): Promise<Campana[]> {
  try {
    const r = await fetch(`/api/promos?tipo=${encodeURIComponent(tipo)}`, {
      headers: { Accept: 'application/json' },
    })
    if (!r.ok) return []
    const data = (await r.json()) as Campana[]
    return Array.isArray(data) ? data : []
  } catch {
    return []
  }
}
