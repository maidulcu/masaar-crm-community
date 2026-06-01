'use client'
import { useEffect, useState } from 'react'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { Payment, Lease, PaginatedResult } from '@/types'

const PAYMENT_METHODS = ['bank_transfer', 'cash', 'cheque', 'credit_card', 'online']

const blank = {
  lease_id: '', amount: 0, currency: 'AED',
  due_date: '', payment_method: 'bank_transfer', notes: '',
}

export default function PaymentsPage() {
  const { t } = useLang()
  const [payments, setPayments] = useState<Payment[]>([])
  const [leases, setLeases] = useState<Lease[]>([])
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
      const [pRes, lRes] = await Promise.all([
        api.payments.list({ page, limit }) as Promise<PaginatedResult<Payment>>,
        api.leases.list({ limit: 100 }) as Promise<PaginatedResult<Lease>>,
      ])
      setPayments(pRes.data ?? [])
      setTotal(pRes.total ?? 0)
      setLeases(lRes.data ?? [])
    } catch { setPayments([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: any) => setForm(f => ({ ...f, [k]: v }))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await api.payments.create(form)
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
    received: 'bg-green-100 text-green-700',
    pending: 'bg-yellow-100 text-yellow-700',
    overdue: 'bg-red-100 text-red-700',
    failed: 'bg-red-100 text-red-700',
    refunded: 'bg-blue-100 text-blue-700',
  }[s] ?? 'bg-gray-100 text-gray-600')

  const statusLabel = (s: string) => t(
    { received: 'مستلمة', pending: 'قيد الانتظار', overdue: 'متأخرة', failed: 'فشلت', refunded: 'مرتجعة' }[s] ?? s,
    { received: 'Received', pending: 'Pending', overdue: 'Overdue', failed: 'Failed', refunded: 'Refunded' }[s] ?? s,
  )

  const leaseLabel = (l: Lease) => {
    const tenant = l.tenant?.full_name_en ?? ''
    const property = l.property?.name ?? ''
    return [tenant, property, l.property?.unit_number].filter(Boolean).join(' · ')
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الدفعات', 'Payments')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">{t('الدفعات', 'Payments')} ({total})</h2>
          <button onClick={() => setOpen(true)} className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
            + {t('دفعة جديدة', 'New Payment')}
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            {payments.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-gray-400 text-sm mb-3">{t('لا توجد دفعات حتى الآن', 'No payments yet')}</p>
                <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">
                  + {t('سجل أول دفعة', 'Record your first payment')}
                </button>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    {[t('المبلغ', 'Amount'), t('تاريخ الاستحقاق', 'Due Date'), t('طريقة الدفع', 'Method'), t('تاريخ الاستلام', 'Received'), t('الحالة', 'Status')].map(h => (
                      <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {payments.map(p => (
                    <tr key={p.id} className="border-b border-gray-100 hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium text-gray-900">{p.currency} {p.amount.toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(p.due_date)}</td>
                      <td className="px-4 py-3 text-gray-600 capitalize text-xs">{p.payment_method?.replace('_', ' ')}</td>
                      <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(p.paid_date)}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${statusColor(p.status)}`}>{statusLabel(p.status)}</span>
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

      <Modal open={open} onClose={() => { setOpen(false); setError('') }} title={t('دفعة جديدة', 'New Payment')}>
        <form onSubmit={handleCreate} className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
          <FormField label={t('العقد', 'Lease')}>
            <select required className={inputCls} value={form.lease_id} onChange={e => set('lease_id', e.target.value)}>
              <option value="">{t('اختر عقد...', 'Select lease...')}</option>
              {leases.map(l => <option key={l.id} value={l.id}>{leaseLabel(l)}</option>)}
            </select>
          </FormField>

          <div className="grid grid-cols-2 gap-3">
            <FormField label={t('المبلغ (AED)', 'Amount (AED)')}>
              <input type="number" min={0} required className={inputCls} value={form.amount} onChange={e => set('amount', +e.target.value)} />
            </FormField>
            <FormField label={t('تاريخ الاستحقاق', 'Due Date')}>
              <input type="date" required className={inputCls} value={form.due_date} onChange={e => set('due_date', e.target.value)} />
            </FormField>
          </div>

          <FormField label={t('طريقة الدفع', 'Payment Method')}>
            <select className={inputCls} value={form.payment_method} onChange={e => set('payment_method', e.target.value)}>
              {PAYMENT_METHODS.map(m => <option key={m} value={m}>{m.replace(/_/g, ' ')}</option>)}
            </select>
          </FormField>

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
