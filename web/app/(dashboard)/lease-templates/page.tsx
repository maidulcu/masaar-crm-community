'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { LeaseTemplate, PaymentFrequency } from '@/types'

const freqLabel: Record<PaymentFrequency, { en: string; ar: string }> = {
  monthly:     { en: 'Monthly',     ar: 'شهري' },
  quarterly:   { en: 'Quarterly',   ar: 'ربع سنوي' },
  semi_annual: { en: 'Semi-Annual', ar: 'نصف سنوي' },
  annual:      { en: 'Annual',      ar: 'سنوي' },
}

const statusColor: Record<string, string> = {
  active:   'bg-green-100 text-green-700',
  inactive: 'bg-gray-100 text-gray-600',
  archived: 'bg-yellow-100 text-yellow-700',
}

const empty: Partial<LeaseTemplate> = {
  name: '',
  description: '',
  is_default: false,
  payment_frequency: 'monthly',
  payment_day_of_month: 1,
  auto_generate_payments: true,
  default_security_deposit_percent: 5,
  default_utility_charges: 0,
  default_late_fee_percent: 2,
  default_lease_duration_months: 12,
  default_notice_period_days: 30,
  default_renewal_duration_months: 12,
  template_document_url: '',
  terms_conditions: '',
  status: 'active',
}

export default function LeaseTemplatesPage() {
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [templates, setTemplates] = useState<LeaseTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [showModal, setShowModal] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState<Partial<LeaseTemplate>>({ ...empty })

  const [deleting, setDeleting] = useState<string | null>(null)

  useEffect(() => {
    loadTemplates()
  }, [page])

  const loadTemplates = async () => {
    setLoading(true)
    try {
      const res: any = await api.leaseTemplates.list({ page, limit })
      setTemplates(res?.data ?? res ?? [])
      setTotal(res?.total ?? 0)
    } catch {
      setTemplates([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const openCreate = () => {
    setForm({ ...empty })
    setEditingId(null)
    setShowModal(true)
  }

  const openEdit = (tpl: LeaseTemplate) => {
    setForm({ ...tpl })
    setEditingId(tpl.id)
    setShowModal(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.name?.trim()) return
    setSaving(true)
    try {
      const payload = { ...form }
      if (editingId) {
        await api.leaseTemplates.update(editingId, payload)
      } else {
        await api.leaseTemplates.create(payload)
      }
      setShowModal(false)
      setPage(1)
      loadTemplates()
    } catch {
      alert(t('حدث خطأ في الحفظ', 'Failed to save template'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('هل أنت متأكد من حذف هذا القالب؟', 'Delete this template?'))) return
    setDeleting(id)
    try {
      await api.leaseTemplates.delete(id)
      loadTemplates()
    } catch {
      alert(t('فشل الحذف', 'Failed to delete'))
    } finally {
      setDeleting(null)
    }
  }

  const setField = (field: string, value: unknown) =>
    setForm(prev => ({ ...prev, [field]: value }))

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('قوالب عقود الإيجار', 'Lease Templates')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : templates.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد قوالب بعد', 'No templates yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('أنشئ قالباً لبدء استخدام عقود الإيجار', 'Create a template to start using lease agreements')}</p>
            </div>
            {isAdmin && (
              <button onClick={openCreate}
                className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
                {t('+ قالب جديد', '+ New Template')}
              </button>
            )}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{total} {t('قالب', 'templates')}</p>
              {isAdmin && (
                <button onClick={openCreate}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                  {t('+ قالب جديد', '+ New Template')}
                </button>
              )}
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('الاسم', 'Name')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('التكرار', 'Frequency')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('المدة', 'Duration')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الإيداع', 'Deposit')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('افتراضي', 'Default')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {templates.map(tpl => (
                    <tr key={tpl.id} className="hover:bg-gray-50 transition-colors cursor-pointer" onClick={() => openEdit(tpl)}>
                      <td className="px-5 py-3.5 font-medium text-gray-900">{tpl.name}</td>
                      <td className="px-5 py-3.5 text-gray-600">
                        {lang === 'ar' ? freqLabel[tpl.payment_frequency]?.ar : freqLabel[tpl.payment_frequency]?.en}
                      </td>
                      <td className="px-5 py-3.5 text-gray-600">
                        {tpl.default_lease_duration_months} {t('شهر', 'months')}
                      </td>
                      <td className="px-5 py-3.5 text-gray-600">
                        {tpl.default_security_deposit_percent}%
                      </td>
                      <td className="px-5 py-3.5">
                        {tpl.is_default
                          ? <span className="text-xs font-medium text-brand-600 bg-brand-50 px-2 py-0.5 rounded-full">{t('نعم', 'Yes')}</span>
                          : <span className="text-xs text-gray-400">—</span>}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', statusColor[tpl.status])}>
                          {tpl.status}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-end" onClick={e => e.stopPropagation()}>
                        {isAdmin && (
                          <button
                            onClick={() => handleDelete(tpl.id)}
                            disabled={deleting === tpl.id}
                            className="text-xs text-red-500 hover:text-red-700 font-medium disabled:opacity-30"
                          >
                            {t('حذف', 'Delete')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-6">
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('السابق', 'Prev')}
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(p => (
                  <button key={p} onClick={() => setPage(p)}
                    className={clsx('px-3 py-1.5 text-sm rounded-lg', p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50')}>
                    {p}
                  </button>
                ))}
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowModal(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-2xl mx-4 shadow-xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">
              {editingId ? t('تعديل القالب', 'Edit Template') : t('قالب جديد', 'New Template')}
            </h2>
            <form onSubmit={handleSave} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className="text-xs text-gray-500 mb-1 block">{t('الاسم', 'Name')} *</label>
                  <input value={form.name || ''} onChange={e => setField('name', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
                </div>
                <div className="col-span-2">
                  <label className="text-xs text-gray-500 mb-1 block">{t('الوصف', 'Description')}</label>
                  <textarea value={form.description || ''} onChange={e => setField('description', e.target.value)} rows={2}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('تكرار الدفع', 'Payment Frequency')}</label>
                  <select value={form.payment_frequency || 'monthly'} onChange={e => setField('payment_frequency', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                    <option value="monthly">{t('شهري', 'Monthly')}</option>
                    <option value="quarterly">{t('ربع سنوي', 'Quarterly')}</option>
                    <option value="semi_annual">{t('نصف سنوي', 'Semi-Annual')}</option>
                    <option value="annual">{t('سنوي', 'Annual')}</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('يوم الدفع', 'Payment Day')}</label>
                  <input type="number" min={1} max={31} value={form.payment_day_of_month ?? 1} onChange={e => setField('payment_day_of_month', parseInt(e.target.value) || 1)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('مدة العقد (أشهر)', 'Duration (months)')}</label>
                  <input type="number" min={1} value={form.default_lease_duration_months ?? 12} onChange={e => setField('default_lease_duration_months', parseInt(e.target.value) || 12)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('فترة الإشعار (أيام)', 'Notice (days)')}</label>
                  <input type="number" min={0} value={form.default_notice_period_days ?? 30} onChange={e => setField('default_notice_period_days', parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('وديعة التأمين %', 'Deposit %')}</label>
                  <input type="number" min={0} step="0.1" value={form.default_security_deposit_percent ?? 5} onChange={e => setField('default_security_deposit_percent', parseFloat(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('رسوم الخدمات', 'Utilities')}</label>
                  <input type="number" min={0} step="0.01" value={form.default_utility_charges ?? 0} onChange={e => setField('default_utility_charges', parseFloat(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('رسوم التأخير %', 'Late Fee %')}</label>
                  <input type="number" min={0} step="0.1" value={form.default_late_fee_percent ?? 2} onChange={e => setField('default_late_fee_percent', parseFloat(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('مدة التجديد (أشهر)', 'Renewal (months)')}</label>
                  <input type="number" min={0} value={form.default_renewal_duration_months ?? 12} onChange={e => setField('default_renewal_duration_months', parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>

              <div className="flex items-center gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" checked={!!form.is_default} onChange={e => setField('is_default', e.target.checked)}
                    className="rounded border-gray-300 text-brand-600 focus:ring-brand-500" />
                  <span className="text-sm text-gray-700">{t('قالب افتراضي', 'Default template')}</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="checkbox" checked={!!form.auto_generate_payments} onChange={e => setField('auto_generate_payments', e.target.checked)}
                    className="rounded border-gray-300 text-brand-600 focus:ring-brand-500" />
                  <span className="text-sm text-gray-700">{t('إنشاء دفعات تلقائياً', 'Auto-generate payments')}</span>
                </label>
              </div>

              <div className="col-span-2">
                <label className="text-xs text-gray-500 mb-1 block">{t('الشروط والأحكام', 'Terms & Conditions')}</label>
                <textarea value={form.terms_conditions || ''} onChange={e => setField('terms_conditions', e.target.value)} rows={4}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>

              <div className="col-span-2">
                <label className="text-xs text-gray-500 mb-1 block">{t('رابط مستند القالب', 'Template Document URL')}</label>
                <input value={form.template_document_url || ''} onChange={e => setField('template_document_url', e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" placeholder="https://..." />
              </div>

              <div className="flex justify-end gap-3 pt-2 border-t border-gray-100">
                <button type="button" onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={saving}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {saving ? '...' : editingId ? t('حفظ', 'Save') : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
