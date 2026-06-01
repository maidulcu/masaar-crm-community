'use client'
import { useEffect, useState, useCallback, useRef } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Listing } from '@/types'
import clsx from 'clsx'

// ── Offer types (local) ────────────────────────────────────────────────────
interface Offer {
  id: string
  contact: { id: string; full_name: string; phone_wa: string }
  offer_amount: number
  currency: string
  status: string
  terms: string
  notes: string
  valid_until: string | null
  deal_id: string | null
  parent_offer_id: string | null
  created_at: string
}

const OFFER_STATUS_COLORS: Record<string, string> = {
  submitted:    'bg-blue-100 text-blue-700',
  under_review: 'bg-amber-100 text-amber-700',
  countered:    'bg-purple-100 text-purple-700',
  accepted:     'bg-green-100 text-green-700',
  rejected:     'bg-red-100 text-red-700',
  expired:      'bg-gray-100 text-gray-500',
}
const OFFER_STATUS_LABELS: Record<string, string> = {
  submitted: 'Submitted', under_review: 'Under Review', countered: 'Countered',
  accepted: 'Accepted', rejected: 'Rejected', expired: 'Expired',
}

const EMIRATES = ['Abu Dhabi', 'Dubai', 'Sharjah', 'Ajman', 'Umm Al Quwain', 'Ras Al Khaimah', 'Fujairah']
const PROPERTY_TYPES = ['apartment', 'villa', 'townhouse', 'commercial', 'land', 'studio', 'warehouse', 'office']
const FURNISHING = ['furnished', 'semi-furnished', 'unfurnished']
const STATUS_OPTIONS = ['draft', 'published', 'sold', 'rented', 'expired', 'withdrawn']

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  published: 'bg-green-100 text-green-700',
  sold: 'bg-blue-100 text-blue-700',
  rented: 'bg-purple-100 text-purple-700',
  expired: 'bg-amber-100 text-amber-700',
  withdrawn: 'bg-red-100 text-red-700',
}

export default function ListingDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [listing, setListing] = useState<Listing | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [form, setForm] = useState({
    title: '', description: '', property_type: 'apartment', listing_type: 'rent',
    price: 0, currency: 'AED', rent_period: '',
    area: '', community: '', subcommunity: '', city: 'Dubai', emirate: 'Dubai',
    bedrooms: 1, bathrooms: 1, total_sqft: 0, plot_sqft: 0, parking_spaces: 0,
    furnishing: '', amenities: [] as string[],
    cover_image_url: '', virtual_tour_url: '', video_url: '',
    reference_number: '', available_from: '',
    owner_name: '', owner_phone: '', owner_email: '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  // ── Offers state ────────────────────────────────────────────────────────
  const [offers, setOffers] = useState<Offer[]>([])
  const [offersLoading, setOffersLoading] = useState(false)
  const [showOfferForm, setShowOfferForm] = useState(false)
  const [offerForm, setOfferForm] = useState({ contact_phone: '', offer_amount: '', currency: 'AED', terms: '', valid_until: '' })
  const [offerSubmitting, setOfferSubmitting] = useState(false)
  const [offerError, setOfferError] = useState('')
  const [offerActing, setOfferActing] = useState<string | null>(null)

  const loadOffers = useCallback(async () => {
    if (!id) return
    setOffersLoading(true)
    try {
      const r = await api.offers.list({ listing_id: id, limit: 50 }) as any
      setOffers(r.data ?? [])
    } catch { setOffers([]) }
    finally { setOffersLoading(false) }
  }, [id])

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const l = await api.listings.get(id) as Listing
      setListing(l)
      setForm({
        title: l.title, description: l.description, property_type: l.property_type,
        listing_type: l.listing_type, price: l.price, currency: l.currency, rent_period: l.rent_period || '',
        area: l.area, community: l.community, subcommunity: l.subcommunity, city: l.city, emirate: l.emirate,
        bedrooms: l.bedrooms, bathrooms: l.bathrooms, total_sqft: l.total_sqft, plot_sqft: l.plot_sqft,
        parking_spaces: l.parking_spaces, furnishing: l.furnishing || '',
        amenities: l.amenities || [],
        cover_image_url: l.cover_image_url, virtual_tour_url: l.virtual_tour_url, video_url: l.video_url,
        reference_number: l.reference_number, available_from: l.available_from || '',
        owner_name: l.owner_name, owner_phone: l.owner_phone, owner_email: l.owner_email,
      })
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load(); loadOffers() }, [id, loadOffers])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setSuccess(''); setSaving(true)
    try {
      await api.listings.update(id, form)
      setSuccess(t('تم الحفظ', 'Saved'))
      setEditOpen(false)
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Error'))
    } finally { setSaving(false) }
  }

  const handleStatusChange = async (status: string) => {
    try {
      await api.listings.updateStatus(id, status)
      setSuccess(t('تم تحديث الحالة', 'Status updated'))
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch { setError(t('حدث خطأ', 'Error')) }
  }

  const handleDelete = async () => {
    if (!confirm(t('حذف هذا الإعلان؟', 'Delete this listing? This cannot be undone.'))) return
    try { await api.listings.delete(id); router.push('/listings') } catch {}
  }

  const set = (k: string, v: unknown) => setForm(f => ({ ...f, [k]: v }))

  const formatPrice = (p: number, c: string, type: string) => {
    const fmt = new Intl.NumberFormat('en-US').format(p)
    return type === 'rent' ? `${c} ${fmt}/year` : `${c} ${fmt}`
  }

  const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  }
  if (!listing) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Listing not found')}</div>
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={listing.title} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/listings" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">
          ← {t('العودة', 'Back to Listings')}
        </Link>

        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        {/* Cover image */}
        <div className="h-56 bg-gradient-to-br from-gray-100 to-gray-200 rounded-xl overflow-hidden">
          {listing.cover_image_url ? (
            <img src={listing.cover_image_url} alt={listing.title} className="w-full h-full object-cover" />
          ) : (
            <div className="flex items-center justify-center h-full text-gray-300">
              <svg className="w-16 h-16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
          )}
        </div>

        {/* Status + actions */}
        <div className="flex items-center gap-3 flex-wrap">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', STATUS_COLORS[listing.status])}>
            {listing.status}
          </span>
          {listing.featured && (
            <span className="px-3 py-1 rounded-full text-sm font-medium bg-gold-100 text-gold-700">
              {t('مميز', 'Featured')}
            </span>
          )}
          {isAgent && (
            <select
              value={listing.status}
              onChange={e => handleStatusChange(e.target.value)}
              className="ml-auto px-3 py-1.5 border border-gray-200 rounded-lg text-sm"
            >
              {STATUS_OPTIONS.map(s => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          )}
        </div>

        {/* Key details */}
        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('السعر', 'Price')}</p>
            <p className="font-bold text-brand-700 text-lg">{formatPrice(listing.price, listing.currency, listing.listing_type)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('النوع', 'Type')}</p>
            <p className="font-semibold text-gray-900 capitalize">{listing.property_type} · {listing.listing_type}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الموقع', 'Location')}</p>
            <p className="font-medium text-gray-700">{listing.area}{listing.community ? `, ${listing.community}` : ''}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الإمارة', 'Emirate')}</p>
            <p className="font-medium text-gray-700">{listing.emirate}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('غرف النوم', 'Bedrooms')}</p>
            <p className="font-medium text-gray-700">{listing.bedrooms}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الحمامات', 'Bathrooms')}</p>
            <p className="font-medium text-gray-700">{listing.bathrooms}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المساحة', 'Sqft')}</p>
            <p className="font-medium text-gray-700">{listing.total_sqft > 0 ? `${listing.total_sqft.toLocaleString()} ft²` : '-'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('مواقف', 'Parking')}</p>
            <p className="font-medium text-gray-700">{listing.parking_spaces}</p>
          </div>
        </div>

        {/* Description */}
        {listing.description && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-2">{t('الوصف', 'Description')}</h3>
            <p className="text-sm text-gray-700 whitespace-pre-wrap">{listing.description}</p>
          </div>
        )}

        {/* Amenities */}
        {listing.amenities?.length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('وسائل الراحة', 'Amenities')}</h3>
            <div className="flex flex-wrap gap-2">
              {listing.amenities.map((a, i) => (
                <span key={i} className="text-xs bg-gray-100 text-gray-700 px-2.5 py-1 rounded-full">{a}</span>
              ))}
            </div>
          </div>
        )}

        {/* Owner info */}
        {(listing.owner_name || listing.owner_phone) && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('المالك', 'Owner')}</h3>
            <div className="grid grid-cols-3 gap-4 text-sm">
              {listing.owner_name && (
                <div>
                  <p className="text-xs text-gray-400 mb-1">{t('الاسم', 'Name')}</p>
                  <p className="font-medium text-gray-700">{listing.owner_name}</p>
                </div>
              )}
              {listing.owner_phone && (
                <div>
                  <p className="text-xs text-gray-400 mb-1">{t('الهاتف', 'Phone')}</p>
                  <p className="font-medium text-gray-700">{listing.owner_phone}</p>
                </div>
              )}
              {listing.owner_email && (
                <div>
                  <p className="text-xs text-gray-400 mb-1">{t('البريد', 'Email')}</p>
                  <p className="font-medium text-gray-700">{listing.owner_email}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── Offers section ─────────────────────────────────────────────── */}
        <div className="bg-white rounded-xl border border-gray-200">
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
            <h3 className="text-sm font-semibold text-gray-700">
              {t('العروض', 'Offers')}
              {offers.length > 0 && (
                <span className="ml-2 px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 rounded-full">{offers.length}</span>
              )}
            </h3>
            {isAgent && (
              <button onClick={() => setShowOfferForm(!showOfferForm)}
                className="px-3 py-1.5 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 transition-colors">
                {showOfferForm ? t('إلغاء', 'Cancel') : t('+ عرض جديد', '+ New Offer')}
              </button>
            )}
          </div>

          {/* New offer form */}
          {showOfferForm && (
            <form onSubmit={async e => {
              e.preventDefault(); setOfferError(''); setOfferSubmitting(true)
              try {
                // Look up contact by phone
                const list = await api.contacts.list({ search: offerForm.contact_phone, page: 1, limit: 1 }) as any
                const contactId = list?.data?.[0]?.id
                if (!contactId) { setOfferError('Contact not found. Please add them in Contacts first.'); return }
                await api.offers.create({
                  listing_id: id,
                  contact_id: contactId,
                  offer_amount: parseFloat(offerForm.offer_amount),
                  currency: offerForm.currency,
                  terms: offerForm.terms,
                  valid_until: offerForm.valid_until || undefined,
                })
                setShowOfferForm(false)
                setOfferForm({ contact_phone: '', offer_amount: '', currency: 'AED', terms: '', valid_until: '' })
                loadOffers()
              } catch (err: any) { setOfferError(err.message || 'Failed') }
              finally { setOfferSubmitting(false) }
            }} className="p-5 border-b border-gray-100 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('رقم هاتف المشتري', 'Buyer Phone')}</label>
                  <input value={offerForm.contact_phone} onChange={e => setOfferForm(p => ({ ...p, contact_phone: e.target.value }))}
                    placeholder="+971501234567" required
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('مبلغ العرض', 'Offer Amount (AED)')}</label>
                  <input type="number" min={1} value={offerForm.offer_amount} onChange={e => setOfferForm(p => ({ ...p, offer_amount: e.target.value }))}
                    placeholder="1500000" required
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الشروط', 'Terms')}</label>
                  <input value={offerForm.terms} onChange={e => setOfferForm(p => ({ ...p, terms: e.target.value }))}
                    placeholder="e.g. Cash, 30-day completion"
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('صالح حتى', 'Valid Until')}</label>
                  <input type="date" value={offerForm.valid_until} onChange={e => setOfferForm(p => ({ ...p, valid_until: e.target.value }))}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              {offerError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{offerError}</p>}
              <button type="submit" disabled={offerSubmitting}
                className="w-full py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50">
                {offerSubmitting ? t('جارٍ الإرسال...', 'Submitting...') : t('إرسال العرض', 'Submit Offer')}
              </button>
            </form>
          )}

          {/* Offers list */}
          {offersLoading ? (
            <div className="p-6 text-center text-xs text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
          ) : offers.length === 0 ? (
            <div className="p-6 text-center text-xs text-gray-400">{t('لا توجد عروض بعد', 'No offers yet')}</div>
          ) : (
            <div className="divide-y divide-gray-100">
              {offers.map(offer => (
                <div key={offer.id} className="px-5 py-3 flex items-center gap-3">
                  <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full shrink-0 ${OFFER_STATUS_COLORS[offer.status]}`}>
                    {OFFER_STATUS_LABELS[offer.status]}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-gray-800">
                      {offer.currency} {offer.offer_amount.toLocaleString()}
                    </p>
                    <p className="text-xs text-gray-500">
                      {offer.contact?.full_name} · {offer.contact?.phone_wa}
                      {offer.terms && ` · ${offer.terms}`}
                      {offer.valid_until && ` · valid until ${new Date(offer.valid_until).toLocaleDateString()}`}
                    </p>
                  </div>
                  {offer.deal_id && (
                    <Link href="/deals" className="text-xs text-green-600 hover:underline shrink-0">Deal →</Link>
                  )}
                  {isAgent && !['accepted','rejected','expired'].includes(offer.status) && (
                    <div className="flex items-center gap-1 shrink-0">
                      <button onClick={async () => {
                        setOfferActing(offer.id)
                        try { await api.offers.accept(offer.id); loadOffers() }
                        catch (e: any) { alert(e.message) }
                        finally { setOfferActing(null) }
                      }} disabled={offerActing === offer.id}
                        className="text-xs px-2 py-1 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50">
                        {t('قبول', 'Accept')}
                      </button>
                      <button onClick={async () => {
                        setOfferActing(offer.id)
                        try { await api.offers.updateStatus(offer.id, 'rejected'); loadOffers() }
                        catch {}
                        finally { setOfferActing(null) }
                      }} disabled={offerActing === offer.id}
                        className="text-xs px-2 py-1 text-red-500 hover:bg-red-50 rounded-md">
                        {t('رفض', 'Reject')}
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* ── Marketing panel ──────────────────────────────────────── */}
        <MarketingPanel listingId={id} listingTitle={listing.title} />

        {/* Actions */}
        <div className="flex flex-wrap gap-3">
          {isAgent && (
            <button onClick={() => setEditOpen(true)}
              className="px-5 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
              {t('تعديل', 'Edit Listing')}
            </button>
          )}
          {isAdmin && (
            <button onClick={handleDelete}
              className="px-5 py-2.5 bg-red-500 text-white text-sm font-medium rounded-lg hover:bg-red-600 transition-colors">
              {t('حذف', 'Delete')}
            </button>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {editOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setEditOpen(false)}>
          <div className="bg-white rounded-xl p-6 w-full max-w-lg mx-4 max-h-[85vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('تعديل الإعلان', 'Edit Listing')}</h3>
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('العنوان', 'Title')}</label>
                <input className={inputCls} value={form.title} onChange={e => set('title', e.target.value)} required />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('النوع', 'Type')}</label>
                  <select className={inputCls} value={form.property_type} onChange={e => set('property_type', e.target.value)}>
                    {PROPERTY_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('نوع الإعلان', 'Listing Type')}</label>
                  <select className={inputCls} value={form.listing_type} onChange={e => set('listing_type', e.target.value)}>
                    <option value="rent">{t('إيجار', 'Rent')}</option>
                    <option value="sale">{t('بيع', 'Sale')}</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('السعر', 'Price')}</label>
                  <input type="number" min={0} className={inputCls} value={form.price} onChange={e => set('price', +e.target.value)} required />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('العملة', 'Currency')}</label>
                  <select className={inputCls} value={form.currency} onChange={e => set('currency', e.target.value)}>
                    <option value="AED">AED</option>
                    <option value="USD">USD</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الإمارة', 'Emirate')}</label>
                  <select className={inputCls} value={form.emirate} onChange={e => set('emirate', e.target.value)}>
                    {EMIRATES.map(em => <option key={em} value={em}>{em}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('المنطقة', 'Area')}</label>
                  <input className={inputCls} value={form.area} onChange={e => set('area', e.target.value)} />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('غرف النوم', 'Beds')}</label>
                  <input type="number" min={0} className={inputCls} value={form.bedrooms} onChange={e => set('bedrooms', +e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الحمامات', 'Baths')}</label>
                  <input type="number" min={0} className={inputCls} value={form.bathrooms} onChange={e => set('bathrooms', +e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('المساحة', 'Sqft')}</label>
                  <input type="number" min={0} className={inputCls} value={form.total_sqft} onChange={e => set('total_sqft', +e.target.value)} />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('الوصف', 'Description')}</label>
                <textarea rows={3} className={inputCls} value={form.description} onChange={e => set('description', e.target.value)} />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('صورة الغلاف', 'Cover Image URL')}</label>
                  <input className={inputCls} value={form.cover_image_url} onChange={e => set('cover_image_url', e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الرقم المرجعي', 'Ref Number')}</label>
                  <input className={inputCls} value={form.reference_number} onChange={e => set('reference_number', e.target.value)} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('اسم المالك', 'Owner Name')}</label>
                  <input className={inputCls} value={form.owner_name} onChange={e => set('owner_name', e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('هاتف المالك', 'Owner Phone')}</label>
                  <input className={inputCls} value={form.owner_phone} onChange={e => set('owner_phone', e.target.value)} />
                </div>
              </div>

              {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

              <div className="flex gap-2 pt-2">
                <button type="button" onClick={() => setEditOpen(false)}
                  className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={saving}
                  className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
                  {saving ? t('جاري الحفظ...', 'Saving...') : t('حفظ', 'Save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Marketing Panel ────────────────────────────────────────────────────────────

function MarketingPanel({ listingId, listingTitle }: { listingId: string; listingTitle: string }) {
  const BASE = process.env.NEXT_PUBLIC_API_URL || ''
  const shareUrl = typeof window !== 'undefined' ? `${window.location.origin}/l/${listingId}` : ''
  const [copied, setCopied] = useState(false)
  const [qrSrc, setQrSrc] = useState<string | null>(null)
  const [showEmailForm, setShowEmailForm] = useState(false)
  const [emailSearch, setEmailSearch] = useState('')
  const [emailMsg, setEmailMsg] = useState(`I thought you might be interested in this property: ${listingTitle}`)
  const [sending, setSending] = useState(false)
  const [emailResult, setEmailResult] = useState<{ sent: number; failed: number } | null>(null)
  const [contacts, setContacts] = useState<{ id: string; full_name: string; email: string }[]>([])
  const [selectedIds, setSelectedIds] = useState<string[]>([])

  // Generate QR client-side
  useEffect(() => {
    if (!shareUrl) return
    import('qrcode').then(QRCode => {
      QRCode.toDataURL(shareUrl, { width: 200, margin: 1 }).then(setQrSrc)
    }).catch(() => {})
  }, [shareUrl])

  const copyLink = () => {
    navigator.clipboard.writeText(shareUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const searchContacts = async (q: string) => {
    if (!q.trim()) { setContacts([]); return }
    try {
      const r = await api.contacts.list({ search: q, page: 1, limit: 20 }) as any
      setContacts((r.data ?? []).filter((c: any) => c.email))
    } catch { setContacts([]) }
  }

  const toggleContact = (id: string) =>
    setSelectedIds(p => p.includes(id) ? p.filter(x => x !== id) : [...p, id])

  const sendCampaign = async () => {
    if (selectedIds.length === 0) return
    setSending(true)
    try {
      const r = await api.marketing.emailCampaign(listingId, {
        contact_ids: selectedIds,
        subject: `Property: ${listingTitle}`,
        message: emailMsg,
      }) as any
      setEmailResult({ sent: r.sent, failed: r.failed })
      setSelectedIds([])
    } catch {}
    finally { setSending(false) }
  }

  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <div className="px-5 py-4 border-b border-gray-100">
        <h3 className="text-sm font-semibold text-gray-700">📢 Marketing Tools</h3>
      </div>
      <div className="p-5 grid grid-cols-1 md:grid-cols-3 gap-4">

        {/* Share link */}
        <div className="space-y-2">
          <p className="text-xs font-semibold text-gray-600">Share Link</p>
          <div className="flex gap-2">
            <input readOnly value={shareUrl}
              className="flex-1 text-xs border border-gray-200 rounded-lg px-2 py-1.5 bg-gray-50 text-gray-600 truncate" />
            <button onClick={copyLink}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors shrink-0 ${
                copied ? 'bg-green-600 text-white' : 'border border-gray-200 text-gray-600 hover:bg-gray-50'
              }`}>
              {copied ? '✓ Copied' : 'Copy'}
            </button>
          </div>
          <a href={`/api/v1/listings/${listingId}/brochure`} target="_blank"
            className="flex items-center gap-1.5 text-xs text-brand-600 hover:underline font-medium">
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Download PDF Brochure
          </a>
          <a href={`/l/${listingId}`} target="_blank"
            className="flex items-center gap-1.5 text-xs text-brand-600 hover:underline font-medium">
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            Public Listing Page ↗
          </a>
        </div>

        {/* QR Code */}
        <div className="flex flex-col items-center gap-2">
          <p className="text-xs font-semibold text-gray-600 self-start">QR Code</p>
          {qrSrc
            ? <img src={qrSrc} alt="QR code" className="w-28 h-28 border border-gray-200 rounded-lg" />
            : <div className="w-28 h-28 bg-gray-100 rounded-lg animate-pulse" />
          }
          {qrSrc && (
            <a href={qrSrc} download={`qr-${listingId}.png`}
              className="text-xs text-brand-600 hover:underline">Download QR</a>
          )}
        </div>

        {/* Email campaign */}
        <div className="space-y-2">
          <p className="text-xs font-semibold text-gray-600">Email Campaign</p>
          {!showEmailForm ? (
            <button onClick={() => setShowEmailForm(true)}
              className="w-full py-2 border border-brand-300 text-brand-600 text-xs font-semibold rounded-lg hover:bg-brand-50">
              Send to Contacts
            </button>
          ) : (
            <div className="space-y-2">
              <input placeholder="Search contacts by name/phone..."
                value={emailSearch}
                onChange={e => { setEmailSearch(e.target.value); searchContacts(e.target.value) }}
                className="w-full text-xs border border-gray-200 rounded-lg px-2 py-1.5 focus:outline-none focus:ring-1 focus:ring-brand-500" />
              {contacts.length > 0 && (
                <div className="max-h-28 overflow-y-auto border border-gray-200 rounded-lg divide-y divide-gray-100">
                  {contacts.map(c => (
                    <label key={c.id} className="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-50 cursor-pointer">
                      <input type="checkbox" checked={selectedIds.includes(c.id)} onChange={() => toggleContact(c.id)}
                        className="accent-brand-600" />
                      <span className="text-xs truncate">{c.full_name} <span className="text-gray-400">{c.email}</span></span>
                    </label>
                  ))}
                </div>
              )}
              <textarea rows={2} value={emailMsg} onChange={e => setEmailMsg(e.target.value)}
                placeholder="Personal message..."
                className="w-full text-xs border border-gray-200 rounded-lg px-2 py-1.5 resize-none focus:outline-none focus:ring-1 focus:ring-brand-500" />
              {emailResult && (
                <p className={`text-xs ${emailResult.failed > 0 ? 'text-amber-600' : 'text-green-600'}`}>
                  Sent {emailResult.sent}, failed {emailResult.failed}
                </p>
              )}
              <div className="flex gap-1.5">
                <button onClick={() => setShowEmailForm(false)}
                  className="flex-1 py-1.5 text-xs border border-gray-200 rounded-lg text-gray-600 hover:bg-gray-50">
                  Cancel
                </button>
                <button onClick={sendCampaign} disabled={sending || selectedIds.length === 0}
                  className="flex-1 py-1.5 text-xs bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 font-semibold">
                  {sending ? '...' : `Send (${selectedIds.length})`}
                </button>
              </div>
            </div>
          )}
        </div>

      </div>
    </div>
  )
}
