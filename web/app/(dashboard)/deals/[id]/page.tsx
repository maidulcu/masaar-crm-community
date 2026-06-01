'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { Deal, VATInvoice } from '@/types'
import clsx from 'clsx'
import DocumentAttachmentSection from '@/components/DocumentAttachmentSection'
import EmailComposeModal from '@/components/communication/EmailComposeModal'

const invoiceStatusColor: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-600',
  sent:  'bg-blue-100 text-blue-700',
  paid:  'bg-green-100 text-green-700',
}

const invoiceStatusLabel: Record<string, { en: string; ar: string }> = {
  draft: { en: 'Draft', ar: 'مسودة' },
  sent:  { en: 'Sent',  ar: 'مرسلة' },
  paid:  { en: 'Paid',  ar: 'مدفوعة' },
}

export default function DealDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [deal, setDeal] = useState<Deal | null>(null)
  const [invoices, setInvoices] = useState<VATInvoice[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [subtotal, setSubtotal] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [showEditForm, setShowEditForm] = useState(false)
  const [editAmount, setEditAmount] = useState('')
  const [editProbability, setEditProbability] = useState('')
  const [editCloseDate, setEditCloseDate] = useState('')
  const [savingEdit, setSavingEdit] = useState(false)
  const [error, setError] = useState('')
  const [emailOpen, setEmailOpen] = useState(false)
  const [contactEmail, setContactEmail] = useState('')
  const [contactName, setContactName] = useState('')
  const { lang, t } = useLang()

  const load = () => {
    if (!id) return
    Promise.all([
      api.deals.get(id) as Promise<Deal>,
      api.deals.invoices(id) as Promise<VATInvoice[]>,
    ]).then(([d, invs]) => {
      setDeal(d)
      setInvoices(Array.isArray(invs) ? invs : [])
      setEditAmount(d.amount?.toString() || '')
      setEditProbability(d.probability?.toString() || '')
      setEditCloseDate(d.close_date || '')
      if (d.lead_id) {
        api.leads.get(d.lead_id).then((lead: any) => {
          const c = lead?.contact
          if (c?.email) setContactEmail(c.email)
          if (c?.full_name) setContactName(c.full_name)
        }).catch(() => {})
      }
    }).finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [id])

  const handleCreateInvoice = async () => {
    const amount = parseFloat(subtotal)
    if (!amount || amount <= 0) return
    setCreating(true)
    try {
      await api.invoices.create({ deal_id: id, subtotal: amount })
      setSubtotal('')
      setShowForm(false)
      load()
    } finally {
      setCreating(false)
    }
  }

  const handleSaveEdit = async () => {
    if (!deal) return
    setSavingEdit(true)
    setError('')
    try {
      await api.deals.update(id, {
        title: deal.title,
        amount: editAmount !== '' ? parseFloat(editAmount) : 0,
        probability: editProbability !== '' ? parseInt(editProbability) : 0,
        close_date: editCloseDate || null,
      })
      setShowEditForm(false)
      load()
    } catch (e) {
      setError((e as Error).message || t('فشل تحديث الصفقة', 'Failed to update deal'))
    } finally {
      setSavingEdit(false)
    }
  }

  const handleSend = async (invId: string) => {
    try {
      await api.invoices.send(invId)
      load()
    } catch (e) {
      setError((e as Error).message || t('فشل إرسال الفاتورة', 'Failed to send invoice'))
    }
  }

  const handleMarkPaid = async (invId: string) => {
    try {
      await api.invoices.updateStatus(invId, 'paid')
      load()
    } catch (e) {
      setError((e as Error).message || t('فشل تحديث الفاتورة', 'Failed to update invoice'))
    }
  }

  const formatAmount = (amount: number) =>
    new Intl.NumberFormat(lang === 'ar' ? 'ar-AE' : 'en-AE', {
      style: 'currency', currency: deal?.currency || 'AED',
    }).format(amount)

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        {t('جاري التحميل...', 'Loading...')}
      </div>
    )
  }

  if (!deal) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        {t('الصفقة غير موجودة', 'Deal not found')}
      </div>
    )
  }

  const vatAmount = invoices.reduce((s, i) => s + i.vat_amount, 0)
  const total = invoices.reduce((s, i) => s + i.total, 0)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={deal.title} />

      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-700 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="text-red-400 hover:text-red-600">✕</button>
          </div>
        )}

        {/* Deal summary with edit button */}
        <div>
          <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-4 mb-3">
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('القيمة', 'Amount')}</p>
              <p className="font-semibold text-gray-900">{deal.amount ? formatAmount(deal.amount) : '—'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('المرحلة', 'Stage')}</p>
              <p className="font-medium text-gray-700 capitalize">{deal.stage}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('الاحتمالية', 'Probability')}</p>
              <p className="font-medium text-gray-700">{deal.probability}%</p>
            </div>
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('تاريخ الإغلاق', 'Close Date')}</p>
              <p className="font-medium text-gray-700">
                {deal.close_date
                  ? new Date(deal.close_date).toLocaleDateString(
                      lang === 'ar' ? 'ar-AE' : 'en-AE',
                      { year: 'numeric', month: 'short', day: 'numeric' }
                    )
                  : '—'}
              </p>
            </div>
          </div>
          <button
            onClick={() => setShowEditForm(!showEditForm)}
            className="text-xs font-medium text-blue-600 hover:text-blue-700"
          >
            {t('تعديل', 'Edit')}
          </button>
        </div>

        {/* Edit form */}
        {showEditForm && (
          <div className="bg-blue-50 border border-blue-100 rounded-xl p-5 space-y-4">
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <div>
                <label className="text-xs text-gray-600 mb-2 block">{t('القيمة', 'Amount')}</label>
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={editAmount}
                  onChange={(e) => setEditAmount(e.target.value)}
                  placeholder="0"
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-300"
                />
              </div>
              <div>
                <label className="text-xs text-gray-600 mb-2 block">{t('الاحتمالية %', 'Probability %')}</label>
                <input
                  type="number"
                  min="0"
                  max="100"
                  value={editProbability}
                  onChange={(e) => setEditProbability(e.target.value)}
                  placeholder="0"
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-300"
                />
              </div>
              <div>
                <label className="text-xs text-gray-600 mb-2 block">{t('تاريخ الإغلاق', 'Close Date')}</label>
                <input
                  type="date"
                  value={editCloseDate}
                  onChange={(e) => setEditCloseDate(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-300"
                />
              </div>
            </div>
            <div className="flex items-center gap-3 justify-end">
              <button
                onClick={() => setShowEditForm(false)}
                className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700"
              >
                {t('إلغاء', 'Cancel')}
              </button>
              <button
                onClick={handleSaveEdit}
                disabled={savingEdit}
                className="px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {savingEdit ? t('جاري الحفظ...', 'Saving...') : t('حفظ', 'Save')}
              </button>
            </div>
          </div>
        )}

        {/* Contact Email */}
        {contactEmail && (
          <div className="bg-white rounded-xl border border-gray-100 p-5 flex items-center justify-between">
            <div>
              <p className="text-xs text-gray-400 mb-1">{t('جهة الاتصال', 'Contact')}</p>
              <p className="font-medium text-gray-700">{contactName || contactEmail}</p>
              <p className="text-xs text-gray-400">{contactEmail}</p>
            </div>
            <button onClick={() => setEmailOpen(true)}
              className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
              ✉ {t('إرسال بريد', 'Send Email')}
            </button>
          </div>
        )}

        {/* Invoices section */}
        <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
            <h2 className="font-semibold text-gray-900 text-sm">{t('الفواتير', 'VAT Invoices')}</h2>
            <button
              onClick={() => setShowForm(!showForm)}
              className="text-xs font-medium px-3 py-1.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors"
            >
              {t('+ فاتورة جديدة', '+ New Invoice')}
            </button>
          </div>

          {/* Create invoice form */}
          {showForm && (
            <div className="px-5 py-4 bg-gray-50 border-b border-gray-100 flex items-end gap-3">
              <div className="flex-1">
                <label className="text-xs text-gray-500 mb-1 block">{t('المبلغ (بدون ضريبة)', 'Subtotal (excl. VAT)')}</label>
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={subtotal}
                  onChange={(e) => setSubtotal(e.target.value)}
                  placeholder="e.g. 10000"
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-300"
                />
              </div>
              <div className="text-xs text-gray-400 pb-2">
                <p>{t('ضريبة القيمة المضافة 5%', 'VAT 5%')}</p>
                {subtotal && <p className="font-medium text-gray-600">{t('الإجمالي:', 'Total:')} {formatAmount(parseFloat(subtotal) * 1.05)}</p>}
              </div>
              <button
                onClick={handleCreateInvoice}
                disabled={creating || !subtotal}
                className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                {creating ? t('جاري الإنشاء...', 'Creating...') : t('إنشاء', 'Create')}
              </button>
              <button
                onClick={() => setShowForm(false)}
                className="px-4 py-2 text-sm text-gray-500 hover:text-gray-700"
              >
                {t('إلغاء', 'Cancel')}
              </button>
            </div>
          )}

          {/* Invoice list */}
          {invoices.length === 0 ? (
            <div className="flex items-center justify-center h-24 text-sm text-gray-400">
              {t('لا توجد فواتير بعد', 'No invoices yet')}
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-400 uppercase tracking-wide border-b border-gray-50">
                  <th className="text-start px-5 py-3 font-medium">{t('رقم الفاتورة', 'Invoice No.')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('المبلغ', 'Subtotal')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('الضريبة', 'VAT')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('الإجمالي', 'Total')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-5 py-3.5 font-mono text-xs text-gray-700">{inv.invoice_no}</td>
                    <td className="px-5 py-3.5 text-gray-700">{formatAmount(inv.subtotal)}</td>
                    <td className="px-5 py-3.5 text-gray-500">{formatAmount(inv.vat_amount)}</td>
                    <td className="px-5 py-3.5 font-semibold text-gray-900">{formatAmount(inv.total)}</td>
                    <td className="px-5 py-3.5">
                      <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', invoiceStatusColor[inv.status])}>
                        {lang === 'ar' ? invoiceStatusLabel[inv.status]?.ar : invoiceStatusLabel[inv.status]?.en}
                      </span>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2 justify-end">
                        <a
                          href={api.invoices.getPDF(inv.id)}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                          </svg>
                          {t('تحميل', 'PDF')}
                        </a>
                        {inv.status === 'draft' && (
                          <button
                            onClick={() => handleSend(inv.id)}
                            className="text-xs text-blue-600 hover:underline font-medium"
                          >
                            {t('إرسال', 'Send')}
                          </button>
                        )}
                        {inv.status === 'sent' && (
                          <button
                            onClick={() => handleMarkPaid(inv.id)}
                            className="text-xs text-green-600 hover:underline font-medium"
                          >
                            {t('تم الدفع', 'Mark Paid')}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
              {invoices.length > 1 && (
                <tfoot className="border-t border-gray-100 bg-gray-50 text-sm">
                  <tr>
                    <td colSpan={3} className="px-5 py-3 text-gray-500 font-medium text-end">{t('الإجمالي الكلي', 'Grand Total')}</td>
                    <td className="px-5 py-3 font-bold text-gray-900">{formatAmount(total)}</td>
                    <td colSpan={2} />
                  </tr>
                </tfoot>
              )}
            </table>
          )}
        </div>

        {/* Documents section */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <DocumentAttachmentSection entityType="deal" entityId={id} canEdit />
        </div>

        {/* VAT summary card */}
        {invoices.length > 0 && (
          <div className="bg-indigo-50 border border-indigo-100 rounded-xl p-5 text-sm">
            <p className="font-semibold text-indigo-800 mb-3 text-xs uppercase tracking-wide">
              {t('ملخص ضريبة القيمة المضافة', 'VAT Summary')}
            </p>
            <div className="grid grid-cols-3 gap-4">
              <div>
                <p className="text-xs text-indigo-400">{t('المبلغ قبل الضريبة', 'Subtotal')}</p>
                <p className="font-semibold text-indigo-900">{formatAmount(total - vatAmount)}</p>
              </div>
              <div>
                <p className="text-xs text-indigo-400">{t('ضريبة القيمة المضافة (5%)', 'VAT (5%)')}</p>
                <p className="font-semibold text-indigo-900">{formatAmount(vatAmount)}</p>
              </div>
              <div>
                <p className="text-xs text-indigo-400">{t('الإجمالي شامل الضريبة', 'Total incl. VAT')}</p>
                <p className="font-bold text-indigo-900 text-base">{formatAmount(total)}</p>
              </div>
            </div>
          </div>
        )}

        <EmailComposeModal
          open={emailOpen}
          onClose={() => setEmailOpen(false)}
          toEmail={contactEmail}
          toName={contactName}
          relatedTo="deal"
          relatedId={id}
        />
      </div>
    </div>
  )
}
