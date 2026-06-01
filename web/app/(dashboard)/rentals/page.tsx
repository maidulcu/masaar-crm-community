'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { RentalProperty, PaginatedResult } from '@/types'

const EMIRATES = ['Abu Dhabi', 'Dubai', 'Sharjah', 'Ajman', 'Umm Al Quwain', 'Ras Al Khaimah', 'Fujairah']
const PROPERTY_TYPES = ['apartment', 'villa', 'commercial', 'townhouse', 'studio', 'warehouse', 'land', 'office']

const blank = {
  name: '', property_type: 'apartment', area: '', city: '', emirate: 'Dubai',
  street_address: '', units_count: 1, bedrooms: 1, bathrooms: 1,
  total_sqft: 0, parking_spaces: 0, description: '',
}

export default function RentalsPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'agent' || isAdmin
  const [properties, setProperties] = useState<RentalProperty[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState(blank)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const limit = 20

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation(); e.preventDefault()
    if (!confirm(t('حذف هذا العقار؟', 'Delete this property? This cannot be undone.'))) return
    try { await api.rentalProperties.delete(id); load() } catch {}
  }

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.rentalProperties.list({ page, limit }) as PaginatedResult<RentalProperty>
      setProperties(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch { setProperties([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: any) => setForm(f => ({ ...f, [k]: v }))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.rentalProperties.create(form)
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
      <Header title={t('العقارات للإيجار', 'Rental Properties')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('العقارات', 'Properties')} ({total})
          </h2>
          <button
            onClick={() => setOpen(true)}
            className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors"
          >
            + {t('عقار جديد', 'New Property')}
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            {properties.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-gray-400 text-sm mb-3">{t('لا توجد عقارات حتى الآن', 'No properties yet')}</p>
                <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">
                  + {t('أضف أول عقار', 'Add your first property')}
                </button>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    {[t('الاسم','Name'), t('النوع','Type'), t('المنطقة','Area'), t('الإمارة','Emirate'), t('الوحدات','Units'), t('الحالة','Status')].map(h => (
                      <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {properties.map(p => (
                    <tr key={p.id} className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer">
                      <td className="px-4 py-3">
                        <Link href={`/rentals/${p.id}`} className="font-medium text-gray-900 hover:text-brand-600">
                          {p.name}
                        </Link>
                      </td>
                      <td className="px-4 py-3 text-gray-600 capitalize">{p.property_type}</td>
                      <td className="px-4 py-3 text-gray-600">{p.area}</td>
                      <td className="px-4 py-3 text-gray-600">{p.emirate}</td>
                      <td className="px-4 py-3 text-gray-600">{p.total_occupied_units}/{p.units_count}</td>
                      <td className="px-4 py-3 flex items-center gap-2">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${p.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>
                          {p.status}
                        </span>
                        {isAdmin && (
                          <button onClick={e => handleDelete(p.id, e)}
                            className="text-xs text-red-400 hover:text-red-600 font-medium ml-2">
                            {t('حذف', 'Delete')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {total > limit && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('السابق', 'Prev')}
            </button>
            <span className="flex items-center px-4 text-sm text-gray-500">{page} / {Math.ceil(total / limit)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page * limit >= total} className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>

      <Modal open={open} onClose={() => { setOpen(false); setError('') }} title={t('عقار جديد', 'New Property')}>
        <form onSubmit={handleCreate} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <FormField label={t('اسم العقار', 'Property Name')}>
            <input required className={inputCls} value={form.name} onChange={e => set('name', e.target.value)} placeholder="Al Noor Tower" />
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('النوع', 'Type')}>
              <select className={inputCls} value={form.property_type} onChange={e => set('property_type', e.target.value)}>
                {PROPERTY_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </FormField>
            <FormField label={t('الإمارة', 'Emirate')}>
              <select className={inputCls} value={form.emirate} onChange={e => set('emirate', e.target.value)}>
                {EMIRATES.map(em => <option key={em} value={em}>{em}</option>)}
              </select>
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('المنطقة', 'Area')}>
              <input className={inputCls} value={form.area} onChange={e => set('area', e.target.value)} placeholder="Downtown" />
            </FormField>
            <FormField label={t('المدينة', 'City')}>
              <input className={inputCls} value={form.city} onChange={e => set('city', e.target.value)} placeholder="Dubai" />
            </FormField>
          </div>

          <FormField label={t('العنوان', 'Street Address')}>
            <input className={inputCls} value={form.street_address} onChange={e => set('street_address', e.target.value)} placeholder="Sheikh Zayed Rd" />
          </FormField>

          <div className="grid grid-cols-3 gap-3">
            <FormField label={t('الوحدات', 'Units')}>
              <input type="number" min={1} className={inputCls} value={form.units_count} onChange={e => set('units_count', +e.target.value)} />
            </FormField>
            <FormField label={t('غرف النوم', 'Beds')}>
              <input type="number" min={0} className={inputCls} value={form.bedrooms} onChange={e => set('bedrooms', +e.target.value)} />
            </FormField>
            <FormField label={t('الحمامات', 'Baths')}>
              <input type="number" min={0} className={inputCls} value={form.bathrooms} onChange={e => set('bathrooms', +e.target.value)} />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('المساحة (قدم²)', 'Total Sqft')}>
              <input type="number" min={0} className={inputCls} value={form.total_sqft} onChange={e => set('total_sqft', +e.target.value)} />
            </FormField>
            <FormField label={t('مواقف السيارات', 'Parking')}>
              <input type="number" min={0} className={inputCls} value={form.parking_spaces} onChange={e => set('parking_spaces', +e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('الوصف', 'Description')}>
            <textarea rows={2} className={inputCls} value={form.description} onChange={e => set('description', e.target.value)} />
          </FormField>

          {error && <FormError error={error} />}

          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setOpen(false)} className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
              {t('إلغاء', 'Cancel')}
            </button>
            <button type="submit" disabled={saving} className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
              {saving ? t('جاري الحفظ...', 'Saving...') : t('إنشاء', 'Create')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
