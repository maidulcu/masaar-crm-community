'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { Lease, Tenant, RentalProperty, PaginatedResult } from '@/types'

const FREQUENCIES = ['monthly', 'quarterly', 'semi_annual', 'annual', 'bi_monthly']

const blank = {
  tenant_id: '', property_id: '', unit_number: '',
  start_date: '', end_date: '',
  monthly_rent: 0, currency: 'AED', security_deposit: 0,
  payment_frequency: 'monthly', payment_day_of_month: 1,
  auto_generate_payments: true, ejari_number: '', notes: '',
}

export default function LeasesPage() {
  const { t } = useLang()
  const [leases, setLeases] = useState<Lease[]>([])
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [properties, setProperties] = useState<RentalProperty[]>([])
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
      const [lRes, tRes, pRes] = await Promise.all([
        api.leases.list({ page, limit }) as Promise<PaginatedResult<Lease>>,
        api.tenants.list({ limit: 100 }) as Promise<PaginatedResult<Tenant>>,
        api.rentalProperties.list({ limit: 100 }) as Promise<PaginatedResult<RentalProperty>>,
      ])
      setLeases(lRes.data ?? [])
      setTotal(lRes.total ?? 0)
      setTenants(tRes.data ?? [])
      setProperties(pRes.data ?? [])
    } catch { setLeases([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: any) => setForm(f => ({ ...f, [k]: v }))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.leases.create(form)
      setOpen(false)
      setForm(blank)
      load()
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally { setSaving(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString() : '-'
  const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'

  const statusColor = (s: string) => ({
    active: 'bg-green-100 text-green-700',
    expired: 'bg-gray-100 text-gray-600',
    terminated: 'bg-red-100 text-red-700',
    pending: 'bg-blue-100 text-blue-700',
  }[s] ?? 'bg-gray-100 text-gray-600')

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('العقود', 'Leases')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">{t('العقود', 'Leases')} ({total})</h2>
          <button onClick={() => setOpen(true)} className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
            + {t('عقد جديد', 'New Lease')}
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            {leases.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-gray-400 text-sm mb-3">{t('لا توجد عقود حتى الآن', 'No leases yet')}</p>
                <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">
                  + {t('أنشئ أول عقد', 'Create your first lease')}
                </button>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    {[t('المستأجر','Tenant'), t('العقار','Property'), t('الإيجار','Rent/mo'), t('البداية','Start'), t('النهاية','End'), t('الحالة','Status')].map(h => (
                      <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {leases.map(l => (
                    <tr key={l.id} className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer">
                      <td className="px-4 py-3">
                        <Link href={`/leases/${l.id}`} className="font-medium text-gray-900 hover:text-brand-600">{l.tenant?.full_name_en || '-'}</Link>
                      </td>
                      <td className="px-4 py-3 text-gray-600">{l.property?.name || '-'}</td>
                      <td className="px-4 py-3 text-gray-700 font-medium">{l.currency} {l.monthly_rent.toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(l.start_date)}</td>
                      <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(l.end_date)}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${statusColor(l.status)}`}>{l.status}</span>
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

      <Modal open={open} onClose={() => { setOpen(false); setError('') }} title={t('عقد إيجار جديد', 'New Lease')}>
        <form onSubmit={handleCreate} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <FormField label={t('المستأجر', 'Tenant')}>
            <select required className={inputCls} value={form.tenant_id} onChange={e => set('tenant_id', e.target.value)}>
              <option value="">{t('اختر مستأجر', 'Select tenant...')}</option>
              {tenants.map(ten => <option key={ten.id} value={ten.id}>{ten.full_name_en}</option>)}
            </select>
          </FormField>

          <FormField label={t('العقار', 'Property')}>
            <select required className={inputCls} value={form.property_id} onChange={e => set('property_id', e.target.value)}>
              <option value="">{t('اختر عقار', 'Select property...')}</option>
              {properties.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </FormField>

          <FormField label={t('رقم الوحدة', 'Unit Number')}>
            <input className={inputCls} value={form.unit_number} onChange={e => set('unit_number', e.target.value)} placeholder="A-101" />
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('تاريخ البداية', 'Start Date')}>
              <input type="date" required className={inputCls} value={form.start_date} onChange={e => set('start_date', e.target.value)} />
            </FormField>
            <FormField label={t('تاريخ النهاية', 'End Date')}>
              <input type="date" required className={inputCls} value={form.end_date} onChange={e => set('end_date', e.target.value)} />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('الإيجار الشهري (AED)', 'Monthly Rent (AED)')}>
              <input type="number" min={0} required className={inputCls} value={form.monthly_rent} onChange={e => set('monthly_rent', +e.target.value)} />
            </FormField>
            <FormField label={t('مبلغ التأمين (AED)', 'Security Deposit (AED)')}>
              <input type="number" min={0} className={inputCls} value={form.security_deposit} onChange={e => set('security_deposit', +e.target.value)} />
            </FormField>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('دورية الدفع', 'Payment Frequency')}>
              <select className={inputCls} value={form.payment_frequency} onChange={e => set('payment_frequency', e.target.value)}>
                {FREQUENCIES.map(f => <option key={f} value={f}>{f.replace('_', ' ')}</option>)}
              </select>
            </FormField>
            <FormField label={t('يوم الدفع من الشهر', 'Payment Day')}>
              <input type="number" min={1} max={28} className={inputCls} value={form.payment_day_of_month} onChange={e => set('payment_day_of_month', +e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('رقم إيجاري', 'Ejari Number')}>
            <input className={inputCls} value={form.ejari_number} onChange={e => set('ejari_number', e.target.value)} />
          </FormField>

          <FormField label={t('ملاحظات', 'Notes')}>
            <textarea rows={2} className={inputCls} value={form.notes} onChange={e => set('notes', e.target.value)} />
          </FormField>

          <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
            <input type="checkbox" checked={form.auto_generate_payments} onChange={e => set('auto_generate_payments', e.target.checked)} className="rounded" />
            {t('إنشاء جدول دفعات تلقائياً', 'Auto-generate payment schedule')}
          </label>

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
