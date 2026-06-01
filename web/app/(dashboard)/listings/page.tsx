'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Listing, PaginatedResult } from '@/types'

const EMIRATES = ['Abu Dhabi', 'Dubai', 'Sharjah', 'Ajman', 'Umm Al Quwain', 'Ras Al Khaimah', 'Fujairah']
const PROPERTY_TYPES = ['apartment', 'villa', 'townhouse', 'commercial', 'land', 'studio', 'warehouse', 'office']
const FURNISHING = ['furnished', 'semi-furnished', 'unfurnished']

const blank = {
  title: '', description: '', property_type: 'apartment', listing_type: 'rent',
  price: 0, currency: 'AED', rent_period: '',
  area: '', community: '', subcommunity: '', city: 'Dubai', emirate: 'Dubai',
  bedrooms: 1, bathrooms: 1, total_sqft: 0, plot_sqft: 0, parking_spaces: 0,
  furnishing: '', amenities: [] as string[], year_built: undefined as number | undefined,
  cover_image_url: '', image_urls: [] as string[], virtual_tour_url: '', video_url: '',
  reference_number: '', available_from: '',
  owner_name: '', owner_phone: '', owner_email: '',
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  published: 'bg-green-100 text-green-700',
  sold: 'bg-blue-100 text-blue-700',
  rented: 'bg-purple-100 text-purple-700',
  expired: 'bg-amber-100 text-amber-700',
  withdrawn: 'bg-red-100 text-red-700',
}

export default function ListingsPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const [listings, setListings] = useState<Listing[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(blank)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const limit = 20

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.listings.list({ page, limit }) as PaginatedResult<Listing>
      setListings(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch { setListings([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: unknown) => setForm(f => ({ ...f, [k]: v }))

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation(); e.preventDefault()
    if (!confirm(t('حذف هذا الإعلان؟', 'Delete this listing? This cannot be undone.'))) return
    try { await api.listings.delete(id); load() } catch {}
  }

  const formatPrice = (p: number, c: string, type: string) => {
    const fmt = new Intl.NumberFormat('en-US').format(p)
    return type === 'rent' ? `${c} ${fmt}/year` : `${c} ${fmt}`
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.listings.create({
        ...form,
        amenities: form.amenities.filter(Boolean),
        image_urls: form.image_urls.filter(Boolean),
        year_built: form.year_built || null,
        available_from: form.available_from || null,
      })
      setOpen(false)
      setForm(blank)
      load()
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally { setSaving(false) }
  }

  const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('القوائم', 'Listings')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('القوائم', 'Listings')} ({total})
          </h2>
          {isAgent && (
            <button
              onClick={() => setOpen(true)}
              className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors"
            >
              + {t('إعلان جديد', 'New Listing')}
            </button>
          )}
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : listings.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
            <p className="text-gray-400 text-sm mb-3">{t('لا توجد إعلانات حتى الآن', 'No listings yet')}</p>
            {isAgent && (
              <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">
                + {t('أضف أول إعلان', 'Add your first listing')}
              </button>
            )}
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {listings.map(l => (
              <Link
                key={l.id}
                href={`/listings/${l.id}`}
                className="block bg-white rounded-xl border border-gray-200 overflow-hidden hover:shadow-card-hover transition-shadow group"
              >
                <div className="h-40 bg-gradient-to-br from-gray-100 to-gray-200 relative">
                  {l.cover_image_url ? (
                    <img src={l.cover_image_url} alt={l.title} className="w-full h-full object-cover" />
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-300">
                      <svg className="w-10 h-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                      </svg>
                    </div>
                  )}
                  <div className="absolute top-2 left-2 flex gap-1">
                    <span className={`px-2 py-0.5 rounded text-[11px] font-medium ${STATUS_COLORS[l.status] || 'bg-gray-100 text-gray-600'}`}>
                      {l.status}
                    </span>
                    {l.featured && (
                      <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-gold-100 text-gold-700">
                        {t('مميز', 'Featured')}
                      </span>
                    )}
                  </div>
                  <div className="absolute bottom-2 right-2">
                    <span className="px-2 py-0.5 rounded text-[11px] font-medium bg-white/90 text-gray-700 capitalize">
                      {l.listing_type}
                    </span>
                  </div>
                </div>
                <div className="p-3 space-y-1.5">
                  <h3 className="font-semibold text-gray-900 text-sm truncate group-hover:text-brand-600 transition-colors">
                    {l.title}
                  </h3>
                  <p className="text-xs text-gray-500">
                    {l.bedrooms} bed · {l.bathrooms} bath · {l.total_sqft > 0 ? `${l.total_sqft.toLocaleString()} sqft` : ''} · {l.area}
                  </p>
                  <p className="text-sm font-bold text-brand-700">
                    {formatPrice(l.price, l.currency, l.listing_type)}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        )}

        {total > limit && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('السابق', 'Prev')}
            </button>
            <span className="flex items-center px-4 text-sm text-gray-500">{page} / {Math.ceil(total / limit)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>

      <Modal open={open} onClose={() => { setOpen(false); setError('') }} title={t('إعلان جديد', 'New Listing')}>
        <form onSubmit={handleCreate} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <FormField label={t('عنوان الإعلان', 'Title')}>
            <input required className={inputCls} value={form.title} onChange={e => set('title', e.target.value)} placeholder="Brand new villa in Palm Jumeirah" />
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('النوع', 'Type')}>
              <select className={inputCls} value={form.property_type} onChange={e => set('property_type', e.target.value)}>
                {PROPERTY_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </FormField>
            <FormField label={t('نوع الإعلان', 'Listing Type')}>
              <select className={inputCls} value={form.listing_type} onChange={e => set('listing_type', e.target.value)}>
                <option value="rent">{t('إيجار', 'Rent')}</option>
                <option value="sale">{t('بيع', 'Sale')}</option>
              </select>
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('السعر', 'Price')}>
              <input type="number" min={0} required className={inputCls} value={form.price || ''} onChange={e => set('price', +e.target.value)} />
            </FormField>
            <FormField label={t('العملة', 'Currency')}>
              <select className={inputCls} value={form.currency} onChange={e => set('currency', e.target.value)}>
                <option value="AED">AED</option>
                <option value="USD">USD</option>
                <option value="EUR">EUR</option>
              </select>
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('الإمارة', 'Emirate')}>
              <select className={inputCls} value={form.emirate} onChange={e => set('emirate', e.target.value)}>
                {EMIRATES.map(em => <option key={em} value={em}>{em}</option>)}
              </select>
            </FormField>
            <FormField label={t('المنطقة', 'Area')}>
              <input className={inputCls} value={form.area} onChange={e => set('area', e.target.value)} placeholder="Dubai Marina" />
            </FormField>
          </div>

          <div className="grid grid-cols-3 gap-3">
            <FormField label={t('غرف النوم', 'Beds')}>
              <input type="number" min={0} className={inputCls} value={form.bedrooms} onChange={e => set('bedrooms', +e.target.value)} />
            </FormField>
            <FormField label={t('الحمامات', 'Baths')}>
              <input type="number" min={0} className={inputCls} value={form.bathrooms} onChange={e => set('bathrooms', +e.target.value)} />
            </FormField>
            <FormField label={t('المساحة', 'Sqft')}>
              <input type="number" min={0} className={inputCls} value={form.total_sqft || ''} onChange={e => set('total_sqft', +e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('الوصف', 'Description')}>
            <textarea rows={3} className={inputCls} value={form.description} onChange={e => set('description', e.target.value)} />
          </FormField>

          <FormField label={t('اسم المالك', 'Owner Name')}>
            <input className={inputCls} value={form.owner_name} onChange={e => set('owner_name', e.target.value)} />
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('هاتف المالك', 'Owner Phone')}>
              <input className={inputCls} value={form.owner_phone} onChange={e => set('owner_phone', e.target.value)} placeholder="+971 50 123 4567" />
            </FormField>
            <FormField label={t('بريد المالك', 'Owner Email')}>
              <input type="email" className={inputCls} value={form.owner_email} onChange={e => set('owner_email', e.target.value)} />
            </FormField>
          </div>

          {error && <FormError error={error} />}

          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setOpen(false)}
              className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
              {t('إلغاء', 'Cancel')}
            </button>
            <button type="submit" disabled={saving}
              className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
              {saving ? t('جاري الحفظ...', 'Saving...') : t('إنشاء', 'Create')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
