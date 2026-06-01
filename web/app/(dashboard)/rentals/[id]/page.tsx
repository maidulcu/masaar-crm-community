'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { RentalProperty } from '@/types'
import clsx from 'clsx'

const EMIRATES = ['Abu Dhabi', 'Dubai', 'Sharjah', 'Ajman', 'Umm Al Quwain', 'Ras Al Khaimah', 'Fujairah']
const PROPERTY_TYPES = ['apartment', 'villa', 'commercial', 'townhouse', 'studio', 'warehouse', 'land', 'office']

export default function RentalPropertyDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [property, setProperty] = useState<RentalProperty | null>(null)
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [form, setForm] = useState({ name: '', property_type: 'apartment', area: '', city: '', emirate: 'Dubai', street_address: '', building_number: '', unit_number: '', units_count: 1, bedrooms: 1, bathrooms: 1, total_sqft: 0, parking_spaces: 0, description: '', purchase_price: 0, market_value: 0, currency: 'AED', title_deed_number: '', municipality_registration: '' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const r = await api.rentalProperties.get(id) as RentalProperty
      setProperty(r)
      setForm({
        name: r.name, property_type: r.property_type, area: r.area, city: r.city, emirate: r.emirate,
        street_address: r.street_address, building_number: r.building_number, unit_number: r.unit_number,
        units_count: r.units_count, bedrooms: r.bedrooms, bathrooms: r.bathrooms,
        total_sqft: r.total_sqft, parking_spaces: r.parking_spaces, description: r.description,
        purchase_price: r.purchase_price, market_value: r.market_value, currency: r.currency,
        title_deed_number: r.title_deed_number, municipality_registration: r.municipality_registration,
      })
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setSuccess(''); setSaving(true)
    try {
      await api.rentalProperties.update(id, form)
      setSuccess(t('تم الحفظ', 'Saved'))
      setEditOpen(false)
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Error'))
    } finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!confirm(t('حذف هذا العقار؟', 'Delete this property? This cannot be undone.'))) return
    try { await api.rentalProperties.delete(id); router.push('/rentals') } catch {}
  }

  const set = (k: string, v: unknown) => setForm(f => ({ ...f, [k]: v }))

  const fmtAmount = (v: number) => `${property?.currency || 'AED'} ${v.toLocaleString()}`
  const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'
  const labelCls = 'block text-xs font-medium text-gray-600 mb-1'

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  }
  if (!property) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Property not found')}</div>
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={property.name} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/rentals" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">
          ← {t('العودة', 'Back to Properties')}
        </Link>

        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        {/* Status */}
        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', property.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600')}>
            {property.status}
          </span>
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', property.occupancy_status === 'occupied' ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-600')}>
            {property.occupancy_status}
          </span>
        </div>

        {/* Key details */}
        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('النوع', 'Type')}</p>
            <p className="font-semibold text-gray-900 capitalize">{property.property_type}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المنطقة', 'Area')}</p>
            <p className="font-medium text-gray-700">{property.area}, {property.emirate}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('العنوان', 'Address')}</p>
            <p className="font-medium text-gray-700">{property.street_address || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الوحدات', 'Units')}</p>
            <p className="font-medium text-gray-700">{property.total_occupied_units}/{property.units_count} {t('مشغولة', 'occupied')}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('غرف النوم', 'Bedrooms')}</p>
            <p className="font-medium text-gray-700">{property.bedrooms}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الحمامات', 'Bathrooms')}</p>
            <p className="font-medium text-gray-700">{property.bathrooms}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المساحة', 'Sqft')}</p>
            <p className="font-medium text-gray-700">{property.total_sqft.toLocaleString()} ft²</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('مواقف', 'Parking')}</p>
            <p className="font-medium text-gray-700">{property.parking_spaces}</p>
          </div>
        </div>

        {/* Financial info */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('المعلومات المالية', 'Financial')}</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-5">
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('سعر الشراء', 'Purchase Price')}</p>
              <p className="font-semibold text-gray-900">{property.purchase_price ? fmtAmount(property.purchase_price) : '-'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('القيمة السوقية', 'Market Value')}</p>
              <p className="font-semibold text-gray-900">{property.market_value ? fmtAmount(property.market_value) : '-'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('تاريخ الشراء', 'Purchase Date')}</p>
              <p className="font-medium text-gray-700">{property.purchase_date ? new Date(property.purchase_date).toLocaleDateString() : '-'}</p>
            </div>
          </div>
        </div>

        {/* Documents & Registration */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('التسجيل والمستندات', 'Registration & Documents')}</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 text-sm">
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('رقم صك الملكية', 'Title Deed')}</p>
              <p className="font-medium text-gray-700">{property.title_deed_number || '- '}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('رقم البلدية', 'Municipality Reg')}</p>
              <p className="font-medium text-gray-700">{property.municipality_registration || '-'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('رقم المبنى', 'Building No')}</p>
              <p className="font-medium text-gray-700">{property.building_number || '-'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('رقم الوحدة', 'Unit No')}</p>
              <p className="font-medium text-gray-700">{property.unit_number || '-'}</p>
            </div>
          </div>
          {property.property_deed_url && (
            <a href={property.property_deed_url} target="_blank" rel="noopener noreferrer"
              className="inline-block mt-3 text-xs text-brand-600 hover:underline font-medium">
              {t('عرض صك الملكية', 'View Deed Document →')}
            </a>
          )}
        </div>

        {/* Amenities */}
        {property.amenities?.length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('وسائل الراحة', 'Amenities')}</h3>
            <div className="flex flex-wrap gap-2">
              {property.amenities.map((a, i) => (
                <span key={i} className="text-xs bg-gray-100 text-gray-700 px-2.5 py-1 rounded-full">{a}</span>
              ))}
            </div>
          </div>
        )}

        {/* Description */}
        {property.description && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-2">{t('الوصف', 'Description')}</h3>
            <p className="text-sm text-gray-700 whitespace-pre-wrap">{property.description}</p>
          </div>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-3">
          <button onClick={() => setEditOpen(true)}
            className="px-5 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
            {t('تعديل', 'Edit Property')}
          </button>
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
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('تعديل العقار', 'Edit Property')}</h3>
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className={labelCls}>{t('اسم العقار', 'Property Name')}</label>
                <input required className={inputCls} value={form.name} onChange={e => set('name', e.target.value)} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('النوع', 'Type')}</label>
                  <select className={inputCls} value={form.property_type} onChange={e => set('property_type', e.target.value)}>
                    {PROPERTY_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                  </select>
                </div>
                <div>
                  <label className={labelCls}>{t('الإمارة', 'Emirate')}</label>
                  <select className={inputCls} value={form.emirate} onChange={e => set('emirate', e.target.value)}>
                    {EMIRATES.map(em => <option key={em} value={em}>{em}</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('المنطقة', 'Area')}</label>
                  <input className={inputCls} value={form.area} onChange={e => set('area', e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('المدينة', 'City')}</label>
                  <input className={inputCls} value={form.city} onChange={e => set('city', e.target.value)} />
                </div>
              </div>
              <div>
                <label className={labelCls}>{t('العنوان', 'Street Address')}</label>
                <input className={inputCls} value={form.street_address} onChange={e => set('street_address', e.target.value)} />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className={labelCls}>{t('رقم المبنى', 'Building No')}</label>
                  <input className={inputCls} value={form.building_number} onChange={e => set('building_number', e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('رقم الوحدة', 'Unit No')}</label>
                  <input className={inputCls} value={form.unit_number} onChange={e => set('unit_number', e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('الوحدات', 'Units')}</label>
                  <input type="number" min={1} className={inputCls} value={form.units_count} onChange={e => set('units_count', +e.target.value)} />
                </div>
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className={labelCls}>{t('غرف النوم', 'Beds')}</label>
                  <input type="number" min={0} className={inputCls} value={form.bedrooms} onChange={e => set('bedrooms', +e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('الحمامات', 'Baths')}</label>
                  <input type="number" min={0} className={inputCls} value={form.bathrooms} onChange={e => set('bathrooms', +e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('المساحة (قدم²)', 'Sqft')}</label>
                  <input type="number" min={0} className={inputCls} value={form.total_sqft} onChange={e => set('total_sqft', +e.target.value)} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('سعر الشراء', 'Purchase Price')}</label>
                  <input type="number" min={0} className={inputCls} value={form.purchase_price} onChange={e => set('purchase_price', +e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('القيمة السوقية', 'Market Value')}</label>
                  <input type="number" min={0} className={inputCls} value={form.market_value} onChange={e => set('market_value', +e.target.value)} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('مواقف السيارات', 'Parking')}</label>
                  <input type="number" min={0} className={inputCls} value={form.parking_spaces} onChange={e => set('parking_spaces', +e.target.value)} />
                </div>
                <div>
                  <label className={labelCls}>{t('رقم صك الملكية', 'Title Deed')}</label>
                  <input className={inputCls} value={form.title_deed_number} onChange={e => set('title_deed_number', e.target.value)} />
                </div>
              </div>
              <div>
                <label className={labelCls}>{t('رقم سجل البلدية', 'Municipality Reg')}</label>
                <input className={inputCls} value={form.municipality_registration} onChange={e => set('municipality_registration', e.target.value)} />
              </div>
              <div>
                <label className={labelCls}>{t('الوصف', 'Description')}</label>
                <textarea rows={2} className={inputCls} value={form.description} onChange={e => set('description', e.target.value)} />
              </div>
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
