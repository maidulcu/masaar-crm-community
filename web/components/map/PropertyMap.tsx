'use client'

// Leaflet requires window — this component is always dynamically imported with ssr:false
import { useEffect, useRef, useState } from 'react'
import 'leaflet/dist/leaflet.css'

// ── Types ─────────────────────────────────────────────────────────────────────

export interface MapArea {
  id?: number
  name?: string
  slug?: string
  lat?: number
  lng?: number
  latitude?: number
  longitude?: number
  transaction_count?: number
  avg_price?: number
  avg_price_per_sqft?: number
}

export interface OwnListing {
  id: string
  title: string
  price: number
  currency: string
  property_type: string
  listing_type: string
  city?: string
  area?: string
  cover_image_url?: string
  status: string
  bos24_listing_uuid?: string
}

export interface POI {
  name: string
  category: string
  lat: number
  lng: number
  distance_km?: number
}

interface PropertyMapProps {
  areas: MapArea[]
  transactionAreas?: MapArea[]
  listings?: OwnListing[]
  pois?: POI[]
  onAreaClick?: (area: MapArea) => void
  onListingClick?: (listing: OwnListing) => void
  showHeatmap?: boolean
  height?: string
}

// Dubai / UAE default center
const DEFAULT_CENTER: [number, number] = [25.2048, 55.2708]
const DEFAULT_ZOOM = 11

// Price color scale — green (cheap) → red (expensive) for transaction heatmap
function priceToColor(price: number, min: number, max: number): string {
  if (max === min) return '#3b82f6'
  const ratio = (price - min) / (max - min)
  const r = Math.round(255 * ratio)
  const g = Math.round(255 * (1 - ratio))
  return `rgb(${r},${g},50)`
}

export default function PropertyMap({
  areas,
  transactionAreas,
  listings = [],
  pois = [],
  onAreaClick,
  onListingClick,
  showHeatmap = false,
  height = '100%',
}: PropertyMapProps) {
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstanceRef = useRef<import('leaflet').Map | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    // Leaflet must be imported dynamically (it uses window)
    let map: import('leaflet').Map | null = null

    async function init() {
      const L = (await import('leaflet')).default

      // Fix default marker icon paths broken by webpack
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      delete (L.Icon.Default.prototype as any)._getIconUrl
      L.Icon.Default.mergeOptions({
        iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
        iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
        shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
      })

      if (!mapRef.current || mapInstanceRef.current) return

      map = L.map(mapRef.current, {
        center: DEFAULT_CENTER,
        zoom: DEFAULT_ZOOM,
        zoomControl: true,
      })

      // Tile layer — OpenStreetMap (free, no API key needed)
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
        maxZoom: 19,
      }).addTo(map)

      mapInstanceRef.current = map

      // ── Area pins / transaction heatmap circles ─────────────────────────

      const source = transactionAreas && transactionAreas.length > 0 ? transactionAreas : areas
      const prices = source.map(a => a.avg_price ?? 0).filter(p => p > 0)
      const minPrice = prices.length ? Math.min(...prices) : 0
      const maxPrice = prices.length ? Math.max(...prices) : 1

      source.forEach(area => {
        const lat = area.lat ?? area.latitude
        const lng = area.lng ?? area.longitude
        if (!lat || !lng) return

        const name = area.name ?? area.slug ?? ''
        const count = area.transaction_count ?? 0
        const avgPrice = area.avg_price ?? 0

        if (showHeatmap && avgPrice > 0) {
          // Colored circle sized by transaction volume
          const radius = Math.max(300, Math.min(2000, count * 15))
          const color = priceToColor(avgPrice, minPrice, maxPrice)
          L.circle([lat, lng], {
            radius,
            color,
            fillColor: color,
            fillOpacity: 0.45,
            weight: 1,
          })
            .addTo(map!)
            .bindPopup(
              `<div class="text-sm font-medium">${name}</div>` +
              `<div class="text-xs text-gray-500 mt-1">${count} transactions</div>` +
              (avgPrice ? `<div class="text-xs">Avg: AED ${avgPrice.toLocaleString()}</div>` : '')
            )
            .on('click', () => onAreaClick?.(area))
        } else {
          // Simple circle marker
          L.circleMarker([lat, lng], {
            radius: 8,
            color: '#3b82f6',
            fillColor: '#3b82f6',
            fillOpacity: 0.6,
            weight: 2,
          })
            .addTo(map!)
            .bindTooltip(name, { direction: 'top' })
            .on('click', () => onAreaClick?.(area))
        }
      })

      // ── Own listings — custom colored markers ─────────────────────────

      const listingIcon = L.divIcon({
        className: '',
        html: `<div style="width:28px;height:28px;background:#16a34a;border:2px solid white;border-radius:50%;display:flex;align-items:center;justify-content:center;box-shadow:0 1px 4px rgba(0,0,0,.35)">
          <svg width="14" height="14" fill="white" viewBox="0 0 20 20"><path d="M10.707 2.293a1 1 0 00-1.414 0l-7 7a1 1 0 001.414 1.414L4 10.414V17a1 1 0 001 1h2a1 1 0 001-1v-2a1 1 0 011-1h2a1 1 0 011 1v2a1 1 0 001 1h2a1 1 0 001-1v-6.586l.293.293a1 1 0 001.414-1.414l-7-7z"/></svg>
        </div>`,
        iconSize: [28, 28],
        iconAnchor: [14, 14],
        popupAnchor: [0, -16],
      })

      listings.forEach(listing => {
        // Listings don't have lat/lng yet — show a placeholder in Dubai center offset slightly
        // When listings table gets coordinates, this will use them
        // For now, skip listings without geocoords
        // TODO: when lat/lng added to listings table, use listing.latitude/listing.longitude
        if (listing.area || listing.city) {
          const marker = L.marker(DEFAULT_CENTER, { icon: listingIcon })
          marker
            .addTo(map!)
            .bindPopup(
              `<div class="font-medium text-sm">${listing.title}</div>` +
              `<div class="text-xs text-gray-500">${listing.property_type} · ${listing.listing_type}</div>` +
              `<div class="text-xs font-semibold mt-1">AED ${listing.price.toLocaleString()}</div>` +
              `<a href="/listings/${listing.id}" class="text-xs text-blue-600 mt-1 block">View listing →</a>`
            )
            .on('click', () => onListingClick?.(listing))
        }
      })

      // ── POI markers ───────────────────────────────────────────────────

      pois.forEach(poi => {
        if (!poi.lat || !poi.lng) return
        const poiIcon = L.divIcon({
          className: '',
          html: `<div style="width:20px;height:20px;background:#f59e0b;border:2px solid white;border-radius:50%;box-shadow:0 1px 3px rgba(0,0,0,.3)"></div>`,
          iconSize: [20, 20],
          iconAnchor: [10, 10],
        })
        L.marker([poi.lat, poi.lng], { icon: poiIcon })
          .addTo(map!)
          .bindTooltip(`${poi.name} (${poi.category})`)
      })

      setReady(true)
    }

    init()

    return () => {
      if (mapInstanceRef.current) {
        mapInstanceRef.current.remove()
        mapInstanceRef.current = null
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Re-render heatmap when showHeatmap toggles (without full reinit)
  // Full reinit would clear all markers — handled by key prop in parent

  return (
    <div ref={mapRef} style={{ height, width: '100%' }} className="rounded-lg overflow-hidden z-0">
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100">
          <div className="w-6 h-6 border-2 border-brand-600 border-t-transparent rounded-full animate-spin" />
        </div>
      )}
    </div>
  )
}
