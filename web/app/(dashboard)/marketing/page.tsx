'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Listing, PaginatedResult } from '@/types'

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  published: 'bg-green-100 text-green-700',
  sold: 'bg-blue-100 text-blue-700',
  rented: 'bg-purple-100 text-purple-700',
  expired: 'bg-amber-100 text-amber-700',
  withdrawn: 'bg-red-100 text-red-700',
}

export default function MarketingPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const [listings, setListings] = useState<Listing[]>([])
  const [loading, setLoading] = useState(true)
  const [copiedId, setCopiedId] = useState('')
  const BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

  useEffect(() => {
    api.listings.list({ page: 1, limit: 100 })
      .then((res) => setListings((res as PaginatedResult<Listing>).data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const published = listings.filter(l => l.status === 'published')

  const copyLink = async (id: string) => {
    const url = `${window.location.origin}/l/${id}`
    try {
      await navigator.clipboard.writeText(url)
      setCopiedId(id)
      setTimeout(() => setCopiedId(''), 2000)
    } catch {}
  }

  const formatPrice = (p: number, c: string, type: string) => {
    const fmt = new Intl.NumberFormat('en-US').format(p)
    return type === 'rent' ? `${c} ${fmt}/yr` : `${c} ${fmt}`
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('أدوات التسويق', 'Marketing Tools')} />

      <div className="flex-1 overflow-auto p-6">
        {loading ? (
          <div className="text-center py-12 text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
        ) : published.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-sm text-gray-500 mb-4">{t('لا توجد إعلانات منشورة بعد', 'No published listings yet')}</p>
            <Link href="/listings" className="text-sm text-brand-600 hover:underline font-medium">
              {t('اذهب إلى الإعلانات', 'Go to Listings')}
            </Link>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {published.map((listing) => {
              const shareUrl = `${typeof window !== 'undefined' ? window.location.origin : ''}/l/${listing.id}`
              return (
                <div key={listing.id} className="bg-white rounded-xl border border-gray-200 overflow-hidden hover:shadow-md transition-shadow">
                  {/* Cover image */}
                  <div className="relative h-40 bg-gray-100">
                    {listing.cover_image_url ? (
                      <img src={listing.cover_image_url} alt={listing.title} className="w-full h-full object-cover" />
                    ) : (
                      <div className="flex items-center justify-center h-full text-gray-300">
                        <svg className="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                      </div>
                    )}
                    <span className={`absolute top-2 left-2 text-[10px] font-semibold px-2 py-0.5 rounded-full ${STATUS_COLORS[listing.status] || 'bg-gray-100 text-gray-700'}`}>
                      {listing.status}
                    </span>
                  </div>

                  {/* Details */}
                  <div className="p-3 space-y-2">
                    <Link href={`/listings/${listing.id}`} className="block">
                      <h3 className="text-sm font-semibold text-gray-800 truncate">{listing.title}</h3>
                      <p className="text-[11px] text-gray-500 truncate">
                        {listing.bedrooms}BR · {listing.property_type} · {listing.area}
                      </p>
                      <p className="text-sm font-bold text-brand-600">
                        {formatPrice(listing.price, listing.currency, listing.listing_type)}
                      </p>
                    </Link>

                    {/* Actions */}
                    <div className="flex flex-wrap gap-1.5 pt-1 border-t border-gray-100">
                      <button onClick={() => copyLink(listing.id)}
                        className="inline-flex items-center gap-1 px-2.5 py-1.5 text-[11px] font-medium rounded-lg bg-gray-100 text-gray-600 hover:bg-gray-200 transition-colors">
                        {copiedId === listing.id ? (
                          <svg className="w-3.5 h-3.5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                          </svg>
                        ) : (
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M7.217 10.907a2.25 2.25 0 100 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186l9.566-5.314m-9.566 7.5l9.566 5.314m0 0a2.25 2.25 0 103.935 2.186 2.25 2.25 0 00-3.935-2.186zm0-12.814a2.25 2.25 0 103.933-2.185 2.25 2.25 0 00-3.933 2.185z" />
                          </svg>
                        )}
                        {copiedId === listing.id ? t('تم', 'Copied!') : t('نسخ الرابط', 'Copy Link')}
                      </button>

                      <a href={api.marketing.brochureUrl(listing.id)} target="_blank" rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2.5 py-1.5 text-[11px] font-medium rounded-lg bg-gray-100 text-gray-600 hover:bg-gray-200 transition-colors">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
                        </svg>
                        PDF
                      </a>

                      <a href={api.marketing.qrUrl(listing.id)} target="_blank" rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 px-2.5 py-1.5 text-[11px] font-medium rounded-lg bg-gray-100 text-gray-600 hover:bg-gray-200 transition-colors">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 013.75 9.375v-4.5zM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5zM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0113.5 9.375v-4.5z" />
                          <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 01-1.125-1.125v-4.5z" />
                        </svg>
                        QR
                      </a>

                      <Link href={`/listings/${listing.id}`}
                        className="inline-flex items-center gap-1 px-2.5 py-1.5 text-[11px] font-medium rounded-lg bg-gray-100 text-gray-600 hover:bg-gray-200 transition-colors ml-auto">
                        {t('فتح', 'Open')}
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
                        </svg>
                      </Link>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
