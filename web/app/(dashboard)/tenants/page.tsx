'use client'
import { useEffect, useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { Tenant, PaginatedResult } from '@/types'

const ID_TYPES = ['emirates_id', 'passport', 'visa', 'driving_license']
const EMP_STATUS = ['employed', 'self_employed', 'unemployed', 'student', 'retired']

const blank = {
  full_name_en: '', full_name_ar: '', email: '', phone_wa: '',
  nationality: '', id_type: 'emirates_id', id_number: '', id_expiry_date: '',
  employment_status: 'employed', employer_name: '', annual_income: 0,
  emergency_contact_name: '', emergency_contact_phone: '', notes: '',
}

export default function TenantsPage() {
  const { t } = useLang()
  const [tenants, setTenants] = useState<Tenant[]>([])
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
      const res = await api.tenants.list({ page, limit }) as PaginatedResult<Tenant>
      setTenants(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch { setTenants([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: any) => setForm(f => ({ ...f, [k]: v }))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.tenants.create(form)
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
      <Header title={t('المستأجرون', 'Tenants')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('المستأجرون', 'Tenants')} ({total})
          </h2>
          <button
            onClick={() => setOpen(true)}
            className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors"
          >
            + {t('مستأجر جديد', 'New Tenant')}
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            {tenants.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-gray-400 text-sm mb-3">{t('لا يوجد مستأجرون حتى الآن', 'No tenants yet')}</p>
                <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">
                  + {t('أضف أول مستأجر', 'Add your first tenant')}
                </button>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    {[t('الاسم','Name'), t('البريد الإلكتروني','Email'), t('الهاتف','Phone'), t('الجنسية','Nationality'), t('التحقق','Verified'), t('الحالة','Status')].map(h => (
                      <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {tenants.map(ten => (
                    <tr key={ten.id} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium text-gray-900">{ten.full_name_en}</td>
                      <td className="px-4 py-3 text-gray-600 text-xs">{ten.email}</td>
                      <td className="px-4 py-3 text-gray-600">{ten.phone_wa}</td>
                      <td className="px-4 py-3 text-gray-600">{ten.nationality}</td>
                      <td className="px-4 py-3">
                        {ten.is_verified
                          ? <span className="px-2 py-1 bg-green-100 text-green-700 rounded text-xs font-medium">✓ {t('محقق', 'Verified')}</span>
                          : <span className="px-2 py-1 bg-yellow-100 text-yellow-700 rounded text-xs font-medium">{t('قيد الانتظار', 'Pending')}</span>}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${ten.status === 'active' ? 'bg-green-100 text-green-700' : ten.status === 'blacklisted' ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600'}`}>
                          {ten.status}
                        </span>
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
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">{t('السابق', 'Prev')}</button>
            <span className="flex items-center px-4 text-sm text-gray-500">{page} / {Math.ceil(total / limit)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page * limit >= total} className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">{t('التالي', 'Next')}</button>
          </div>
        )}
      </div>

      <Modal open={open} onClose={() => { setOpen(false); setError('') }} title={t('مستأجر جديد', 'New Tenant')}>
        <form onSubmit={handleCreate} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('الاسم (إنجليزي)', 'Full Name (EN)')}>
              <input required className={inputCls} value={form.full_name_en} onChange={e => set('full_name_en', e.target.value)} placeholder="Ahmed Al Mansouri" />
            </FormField>
            <FormField label={t('الاسم (عربي)', 'Full Name (AR)')}>
              <input className={inputCls} dir="rtl" value={form.full_name_ar} onChange={e => set('full_name_ar', e.target.value)} placeholder="أحمد المنصوري" />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('البريد الإلكتروني', 'Email')}>
              <input type="email" required className={inputCls} value={form.email} onChange={e => set('email', e.target.value)} />
            </FormField>
            <FormField label={t('رقم واتساب', 'WhatsApp Number')}>
              <input className={inputCls} value={form.phone_wa} onChange={e => set('phone_wa', e.target.value)} placeholder="+971501234567" />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('الجنسية', 'Nationality')}>
              <input className={inputCls} value={form.nationality} onChange={e => set('nationality', e.target.value)} placeholder="Emirati" />
            </FormField>
            <FormField label={t('حالة التوظيف', 'Employment Status')}>
              <select className={inputCls} value={form.employment_status} onChange={e => set('employment_status', e.target.value)}>
                {EMP_STATUS.map(s => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
              </select>
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('نوع الهوية', 'ID Type')}>
              <select className={inputCls} value={form.id_type} onChange={e => set('id_type', e.target.value)}>
                {ID_TYPES.map(s => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
              </select>
            </FormField>
            <FormField label={t('رقم الهوية', 'ID Number')}>
              <input required className={inputCls} value={form.id_number} onChange={e => set('id_number', e.target.value)} />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('تاريخ انتهاء الهوية', 'ID Expiry')}>
              <input type="date" className={inputCls} value={form.id_expiry_date} onChange={e => set('id_expiry_date', e.target.value)} />
            </FormField>
            <FormField label={t('جهة العمل', 'Employer')}>
              <input className={inputCls} value={form.employer_name} onChange={e => set('employer_name', e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('الدخل السنوي (AED)', 'Annual Income (AED)')}>
            <input type="number" min={0} className={inputCls} value={form.annual_income} onChange={e => set('annual_income', +e.target.value)} />
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('جهة الاتصال للطوارئ', 'Emergency Contact')}>
              <input className={inputCls} value={form.emergency_contact_name} onChange={e => set('emergency_contact_name', e.target.value)} />
            </FormField>
            <FormField label={t('هاتف الطوارئ', 'Emergency Phone')}>
              <input className={inputCls} value={form.emergency_contact_phone} onChange={e => set('emergency_contact_phone', e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('ملاحظات', 'Notes')}>
            <textarea rows={2} className={inputCls} value={form.notes} onChange={e => set('notes', e.target.value)} />
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
