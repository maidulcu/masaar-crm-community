'use client'

import { useState, useEffect, useCallback } from 'react'
import dynamic from 'next/dynamic'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { MapArea, OwnListing } from '@/components/map/PropertyMap'

// Leaflet must not be SSR'd (it uses window)
const PropertyMap = dynamic(() => import('@/components/map/PropertyMap'), {
  ssr: false,
  loading: () => (
    <div className="flex-1 flex items-center justify-center bg-gray-100 rounded-lg">
      <div className="w-8 h-8 border-2 border-brand-600 border-t-transparent rounded-full animate-spin" />
    </div>
  ),
})

// ── Types ────────────────────────────────────────────────────────────────────

interface AreaStat {
  name: string
  avg_price: number
  avg_price_per_sqft: number
  transaction_count: number
  change_pct?: number
}

// ── Filters ──────────────────────────────────────────────────────────────────

const PROPERTY_TYPES = ['', 'Apartment', 'Villa', 'Townhouse', 'Commercial', 'Land']
const TRANS_TYPES = ['', 'Sell', 'Rent']

// ── Component ─────────────────────────────────────────────────────────────────

export default function MapPage() {
  const { t } = useLang()

  const [areas, setAreas] = useState<MapArea[]>([])
  const [transAreas, setTransAreas] = useState<MapArea[]>([])
  const [listings, setListings] = useState<OwnListing[]>([])
  const [loading, setLoading] = useState(true)
  const [bos24Unavailable, setBos24Unavailable] = useState(false)

  // Filters
  const [propertyType, setPropertyType] = useState('')
  const [transType, setTransType] = useState('')

  // UI state
  const [showHeatmap, setShowHeatmap] = useState(true)
  const [showListings, setShowListings] = useState(true)
  const [selectedArea, setSelectedArea] = useState<AreaStat | null>(null)
  const [selectedListing, setSelectedListing] = useState<OwnListing | null>(null)

  // Reload map areas when filters change (used as key to force remount)
  const [mapKey, setMapKey] = useState(0)

  const loadData = useCallback(async () => {
    setLoading(true)
    setBos24Unavailable(false)

    try {
      // Load BOS24 area pins and transaction heatmap in parallel
      const [areasRes, transRes] = await Promise.allSettled([
        api.bos24.map.areas() as Promise<{ areas: MapArea[] }>,
        api.bos24.map.transactionAreas(propertyType || undefined, transType || undefined) as Promise<{ areas: MapArea[] }>,
      ])

      if (areasRes.status === 'fulfilled') {
        setAreas(areasRes.value?.areas ?? [])
      } else {
        setBos24Unavailable(true)
      }
      if (transRes.status === 'fulfilled') {
        setTransAreas(transRes.value?.areas ?? [])
      }
    } catch {
      setBos24Unavailable(true)
    }

    // Load own listings
    try {
      const r = await api.listings.list({ page: 1, limit: 200 }) as any
      const raw: OwnListing[] = r?.data ?? r?.listings ?? []
      setListings(raw.filter(l => l.status === 'active' || l.status === 'published'))
    } catch {
      // ok if listings fail
    }

    setLoading(false)
    setMapKey(k => k + 1) // force map remount with fresh data
  }, [propertyType, transType])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handleAreaClick = (area: MapArea) => {
    setSelectedArea({
      name: area.name ?? area.slug ?? '',
      avg_price: area.avg_price ?? 0,
      avg_price_per_sqft: area.avg_price_per_sqft ?? 0,
      transaction_count: area.transaction_count ?? 0,
    })
    setSelectedListing(null)
  }

  const handleListingClick = (listing: OwnListing) => {
    setSelectedListing(listing)
    setSelectedArea(null)
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('خريطة العقارات', 'Property Map')} />

      <div className="flex flex-1 min-h-0 overflow-hidden">

        {/* ── Left sidebar: filters + info panel ───────────────────────── */}
        <aside className="w-72 shrink-0 border-r border-gray-200 bg-white flex flex-col overflow-y-auto">

          {/* Filters */}
          <div className="p-4 border-b border-gray-100 space-y-3">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide">
              {t('تصفية', 'Filters')}
            </p>

            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">
                {t('نوع العقار', 'Property Type')}
              </label>
              <select
                value={propertyType}
                onChange={e => setPropertyType(e.target.value)}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500"
              >
                {PROPERTY_TYPES.map(v => (
                  <option key={v} value={v}>{v || t('الكل', 'All types')}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">
                {t('نوع المعاملة', 'Transaction Type')}
              </label>
              <select
                value={transType}
                onChange={e => setTransType(e.target.value)}
                className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500"
              >
                {TRANS_TYPES.map(v => (
                  <option key={v} value={v}>{v || t('الكل', 'All')}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Layer toggles */}
          <div className="p-4 border-b border-gray-100 space-y-2">
            <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">
              {t('الطبقات', 'Layers')}
            </p>
            <label className="flex items-center gap-2.5 cursor-pointer">
              <div
                onClick={() => setShowHeatmap(!showHeatmap)}
                className={`w-8 h-4 rounded-full transition-colors ${showHeatmap ? 'bg-brand-600' : 'bg-gray-300'}`}
              >
                <div className={`w-4 h-4 rounded-full bg-white shadow transition-transform ${showHeatmap ? 'translate-x-4' : 'translate-x-0'}`} />
              </div>
              <span className="text-sm text-gray-700">{t('خريطة حرارية', 'Transaction Heatmap')}</span>
            </label>
            <label className="flex items-center gap-2.5 cursor-pointer">
              <div
                onClick={() => setShowListings(!showListings)}
                className={`w-8 h-4 rounded-full transition-colors ${showListings ? 'bg-green-600' : 'bg-gray-300'}`}
              >
                <div className={`w-4 h-4 rounded-full bg-white shadow transition-transform ${showListings ? 'translate-x-4' : 'translate-x-0'}`} />
              </div>
              <span className="text-sm text-gray-700">{t('قوائمي', 'My Listings')}</span>
            </label>
          </div>

          {/* Legend */}
          {showHeatmap && !bos24Unavailable && (
            <div className="p-4 border-b border-gray-100">
              <p className="text-xs font-semibold text-gray-500 mb-2">
                {t('مفتاح الألوان', 'Price Legend')}
              </p>
              <div className="flex items-center gap-1">
                <span className="text-[10px] text-gray-500">{t('منخفض', 'Low')}</span>
                <div className="flex-1 h-2 rounded-full"
                  style={{ background: 'linear-gradient(to right, rgb(0,255,50), rgb(255,0,50))' }} />
                <span className="text-[10px] text-gray-500">{t('مرتفع', 'High')}</span>
              </div>
              <p className="text-[10px] text-gray-400 mt-1">Average transaction price per area</p>
            </div>
          )}

          {/* BOS24 unavailable notice */}
          {bos24Unavailable && (
            <div className="p-4">
              <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 text-xs text-amber-800">
                <p className="font-semibold mb-1">BOS24 not configured</p>
                <p>Area data and transaction heatmap require a BOS24 API key.</p>
                <a href="/settings/api" className="text-amber-700 underline font-medium mt-1 block">
                  Configure in Settings →
                </a>
              </div>
            </div>
          )}

          {/* Selected area stats */}
          {selectedArea && (
            <div className="p-4 space-y-2 border-b border-gray-100">
              <div className="flex items-center justify-between">
                <p className="text-sm font-semibold text-gray-800">{selectedArea.name}</p>
                <button onClick={() => setSelectedArea(null)} className="text-gray-400 hover:text-gray-600 text-xs">✕</button>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="bg-gray-50 rounded-lg p-2.5">
                  <p className="text-[10px] text-gray-400 uppercase tracking-wide">Transactions</p>
                  <p className="text-sm font-bold text-gray-800">{selectedArea.transaction_count.toLocaleString()}</p>
                </div>
                <div className="bg-gray-50 rounded-lg p-2.5">
                  <p className="text-[10px] text-gray-400 uppercase tracking-wide">Avg Price</p>
                  <p className="text-sm font-bold text-gray-800">
                    {selectedArea.avg_price > 0
                      ? `AED ${(selectedArea.avg_price / 1000).toFixed(0)}K`
                      : '—'}
                  </p>
                </div>
                {selectedArea.avg_price_per_sqft > 0 && (
                  <div className="bg-gray-50 rounded-lg p-2.5 col-span-2">
                    <p className="text-[10px] text-gray-400 uppercase tracking-wide">Avg per sqft</p>
                    <p className="text-sm font-bold text-gray-800">
                      AED {selectedArea.avg_price_per_sqft.toLocaleString()}
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Selected listing */}
          {selectedListing && (
            <div className="p-4 border-b border-gray-100">
              <div className="flex items-center justify-between mb-2">
                <p className="text-sm font-semibold text-gray-800 truncate pr-2">{selectedListing.title}</p>
                <button onClick={() => setSelectedListing(null)} className="text-gray-400 hover:text-gray-600 text-xs shrink-0">✕</button>
              </div>
              {selectedListing.cover_image_url && (
                <img src={selectedListing.cover_image_url} alt={selectedListing.title}
                  className="w-full h-28 object-cover rounded-lg mb-2" />
              )}
              <div className="space-y-1 text-xs text-gray-600">
                <p><span className="text-gray-400">Type:</span> {selectedListing.property_type} · {selectedListing.listing_type}</p>
                <p><span className="text-gray-400">Price:</span> <strong>AED {selectedListing.price.toLocaleString()}</strong></p>
                {selectedListing.area && <p><span className="text-gray-400">Area:</span> {selectedListing.area}</p>}
              </div>
              <a href={`/listings/${selectedListing.id}`}
                className="block mt-3 text-center text-xs font-semibold text-brand-600 border border-brand-200 rounded-lg py-2 hover:bg-brand-50 transition-colors">
                {t('عرض القائمة', 'View Listing')} →
              </a>
            </div>
          )}

          {/* Stats summary */}
          {!bos24Unavailable && areas.length > 0 && (
            <div className="p-4 mt-auto">
              <p className="text-[10px] text-gray-400">
                {areas.length} areas loaded · {listings.length} own listings
              </p>
            </div>
          )}
        </aside>

        {/* ── Map area ─────────────────────────────────────────────────────── */}
        <div className="flex-1 relative p-3">
          {loading ? (
            <div className="absolute inset-0 flex items-center justify-center bg-gray-50">
              <div className="text-center">
                <div className="w-8 h-8 border-2 border-brand-600 border-t-transparent rounded-full animate-spin mx-auto mb-3" />
                <p className="text-sm text-gray-500">{t('جارٍ تحميل بيانات الخريطة...', 'Loading map data...')}</p>
              </div>
            </div>
          ) : (
            <PropertyMap
              key={`${mapKey}-${showHeatmap}-${showListings}`}
              areas={areas}
              transactionAreas={showHeatmap ? transAreas : []}
              listings={showListings ? listings : []}
              showHeatmap={showHeatmap}
              onAreaClick={handleAreaClick}
              onListingClick={handleListingClick}
              height="100%"
            />
          )}
        </div>
      </div>
    </div>
  )
}
