'use client'
import { useEffect, useState, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { VATInvoice } from '@/types'
import { formatMoney } from '@/lib/money'
import { formatDate } from '@/lib/date'
import { INVOICE_STATUS_COLORS, INVOICE_STATUS_LABELS, VAT_CATEGORY_LABELS, documentTitle } from '@/lib/invoice'

export default function InvoiceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const router = useRouter()

  const [invoice, setInvoice] = useState<VATInvoice | null>(null)
  const [creditNotes, setCreditNotes] = useState<VATInvoice[]>([])
  const [loading, setLoading] = useState(true)
  const [updating, setUpdating] = useState(false)
  const [error, setError] = useState('')
  const [panel, setPanel] = useState<'' | 'void' | 'credit'>('')
  const [reason, setReason] = useState('')
  const [creditAmount, setCreditAmount] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const inv = await api.invoices.get(id) as VATInvoice
      setInvoice(inv)
      if (inv.kind === 'invoice') {
        setCreditNotes(await api.invoices.creditNotes(id) as VATInvoice[])
      }
    } catch {
      setInvoice(null)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { load() }, [load])

  const run = async (fn: () => Promise<unknown>) => {
    setUpdating(true); setError('')
    try {
      await fn()
      setPanel(''); setReason(''); setCreditAmount('')
      await load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally {
      setUpdating(false)
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col flex-1 overflow-hidden">
        <Header title={t('الفاتورة', 'Invoice')} />
        <div className="flex items-center justify-center flex-1 text-sm text-gray-400">
          {t('جاري التحميل...', 'Loading...')}
        </div>
      </div>
    )
  }

  if (!invoice) {
    return (
      <div className="flex flex-col flex-1 overflow-hidden">
        <Header title={t('الفاتورة', 'Invoice')} />
        <div className="flex items-center justify-center flex-1">
          <div className="text-center">
            <p className="text-sm text-gray-400">{t('الفاتورة غير موجودة', 'Invoice not found')}</p>
            <button onClick={() => router.push('/invoices')} className="mt-3 text-sm text-brand-600 hover:underline">
              {t('← العودة للفواتير', '← Back to Invoices')}
            </button>
          </div>
        </div>
      </div>
    )
  }

  const money = (n: number) => formatMoney(n, invoice.currency)
  const title = documentTitle(invoice.kind, invoice.is_tax_invoice)
  const isInvoice = invoice.kind === 'invoice'
  const canVoid = isAdmin && (invoice.status === 'draft' || invoice.status === 'sent') &&
    !creditNotes.some(cn => cn.status !== 'void')
  const canCredit = isAdmin && isInvoice && invoice.status !== 'void' && invoice.total - invoice.credited_total > 0
  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={invoice.invoice_no} />

      <div className="flex-1 overflow-y-auto p-6 max-w-3xl">
        <button
          onClick={() => router.push('/invoices')}
          className="text-xs text-gray-400 hover:text-brand-600 mb-6 flex items-center gap-1 transition-colors"
        >
          <svg className="w-3.5 h-3.5 rtl:rotate-180" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          {t('جميع الفواتير', 'All Invoices')}
        </button>

        <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
          <div className="px-6 py-5 border-b border-gray-100 flex items-start justify-between gap-4">
            <div>
              <p className="text-xs uppercase tracking-wide text-gray-400">{lang === 'ar' ? title.ar : title.en}</p>
              <p className="font-mono text-lg font-semibold text-gray-900">{invoice.invoice_no}</p>
              <p className="text-xs text-gray-400 mt-0.5">
                {t('تاريخ الإصدار', 'Issued')} {formatDate(invoice.issued_at)}
                {invoice.supply_date && <> · {t('تاريخ التوريد', 'Supply date')} {formatDate(invoice.supply_date)}</>}
              </p>
              {invoice.original_invoice_id && (
                <Link href={`/invoices/${invoice.original_invoice_id}`} className="text-xs text-brand-600 hover:underline">
                  {t('الفاتورة الأصلية', 'Original invoice')}
                </Link>
              )}
            </div>
            <span className={clsx('text-sm font-medium px-3 py-1 rounded-full', INVOICE_STATUS_COLORS[invoice.status])}>
              {lang === 'ar' ? INVOICE_STATUS_LABELS[invoice.status].ar : INVOICE_STATUS_LABELS[invoice.status].en}
            </span>
          </div>

          {!invoice.is_tax_invoice && (
            <p className="px-6 py-2 text-xs bg-amber-50 text-amber-800 border-b border-amber-100">
              {t('أُصدر هذا المستند دون رقم تسجيل ضريبي للشركة، لذلك لا يُعد فاتورة ضريبية ولا يتضمن ضريبة.',
                'Issued without a company TRN, so this is not a tax invoice and carries no VAT.')}
            </p>
          )}

          <div className="grid grid-cols-2 gap-6 px-6 py-5 border-b border-gray-100 text-sm">
            {[
              { label: t('المورد', 'Supplier'), p: invoice.supplier },
              { label: t('العميل', 'Customer'), p: invoice.customer },
            ].map(({ label, p }) => (
              <div key={label}>
                <p className="text-xs text-gray-400 mb-1">{label}</p>
                <p className="font-medium text-gray-900">{p.name || '—'}</p>
                {p.address && <p className="text-gray-600 whitespace-pre-line">{p.address}</p>}
                {p.trn && <p className="text-gray-600" dir="ltr">TRN {p.trn}</p>}
              </div>
            ))}
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-xs text-gray-400 bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-2 text-start font-medium">{t('الوصف', 'Description')}</th>
                  <th scope="col" className="px-3 py-2 text-end font-medium">{t('الكمية', 'Qty')}</th>
                  <th scope="col" className="px-3 py-2 text-end font-medium">{t('سعر الوحدة', 'Unit price')}</th>
                  <th scope="col" className="px-3 py-2 text-start font-medium">{t('الضريبة', 'VAT')}</th>
                  <th scope="col" className="px-6 py-2 text-end font-medium">{t('المبلغ', 'Amount')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {(invoice.lines ?? []).map(l => (
                  <tr key={l.id ?? l.position}>
                    <td className="px-6 py-2.5 text-gray-900">{l.description}</td>
                    <td className="px-3 py-2.5 text-end tabular-nums">{l.quantity}</td>
                    <td className="px-3 py-2.5 text-end tabular-nums">{money(l.unit_price)}</td>
                    <td className="px-3 py-2.5 text-xs text-gray-500">
                      {lang === 'ar' ? VAT_CATEGORY_LABELS[l.vat_category].ar : VAT_CATEGORY_LABELS[l.vat_category].en}
                      {' '}({Math.round((l.vat_rate ?? 0) * 10000) / 100}%)
                    </td>
                    <td className="px-6 py-2.5 text-end tabular-nums">{money(l.net_amount ?? 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="px-6 py-5 space-y-2 border-t border-gray-100">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">{t('المبلغ قبل الضريبة', 'Subtotal')}</span>
              <span className="tabular-nums text-gray-900">{money(invoice.subtotal)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">{t('ضريبة القيمة المضافة', 'VAT')}</span>
              <span className="tabular-nums text-gray-900">{money(invoice.vat_amount)}</span>
            </div>
            <div className="flex justify-between text-sm font-semibold border-t border-gray-100 pt-2">
              <span className="text-gray-900">{t('الإجمالي', 'Total')}</span>
              <span className="tabular-nums text-gray-900">{money(invoice.total)}</span>
            </div>
            {isInvoice && invoice.credited_total > 0 && (
              <>
                <div className="flex justify-between text-sm text-amber-700">
                  <span>{t('إشعارات دائنة', 'Credited')}</span>
                  <span className="tabular-nums">−{money(invoice.credited_total)}</span>
                </div>
                <div className="flex justify-between text-sm font-semibold">
                  <span>{t('المستحق', 'Outstanding')}</span>
                  <span className="tabular-nums">{money(invoice.outstanding)}</span>
                </div>
              </>
            )}
            {invoice.reason && <p className="text-xs text-gray-500">{t('السبب:', 'Reason:')} {invoice.reason}</p>}
            {invoice.notes && <p className="text-xs text-gray-500 whitespace-pre-line">{invoice.notes}</p>}
            {invoice.status === 'void' && invoice.void_reason && (
              <p className="text-xs text-red-600">{t('سبب الإلغاء:', 'Void reason:')} {invoice.void_reason}</p>
            )}
          </div>

          <div className="px-6 py-4 bg-gray-50 border-t border-gray-100 flex flex-wrap items-center gap-3">
            <a
              href={`/invoices/${invoice.id}/print`}
              target="_blank"
              rel="noopener noreferrer"
              className="px-4 py-2 text-sm font-medium border border-gray-200 rounded-lg bg-white hover:bg-gray-50 transition-colors text-gray-700"
            >
              {t('طباعة / حفظ PDF', 'Print / Save as PDF')}
            </a>
            {isAdmin && invoice.status === 'draft' && (
              <button onClick={() => run(() => api.invoices.send(invoice.id))} disabled={updating}
                className="px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
                {t('تعليم كمُرسلة', 'Mark as Sent')}
              </button>
            )}
            {isAdmin && isInvoice && invoice.status === 'sent' && (
              <button onClick={() => run(() => api.invoices.updateStatus(invoice.id, 'paid'))} disabled={updating}
                className="px-4 py-2 text-sm font-medium bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50">
                {t('تأكيد الدفع', 'Mark as Paid')}
              </button>
            )}
            {canCredit && (
              <button onClick={() => setPanel(panel === 'credit' ? '' : 'credit')}
                className="px-4 py-2 text-sm font-medium border border-amber-300 text-amber-800 rounded-lg hover:bg-amber-50">
                {t('إصدار إشعار دائن', 'Issue credit note')}
              </button>
            )}
            {canVoid && (
              <button onClick={() => setPanel(panel === 'void' ? '' : 'void')}
                className="px-4 py-2 text-sm font-medium border border-red-200 text-red-700 rounded-lg hover:bg-red-50">
                {t('إلغاء المستند', 'Void')}
              </button>
            )}
          </div>

          {panel && (
            <div className="px-6 py-4 border-t border-gray-100 space-y-3">
              {panel === 'credit' && (
                <div>
                  <label htmlFor="cn-amount" className="block text-xs font-medium text-gray-600 mb-1">
                    {t('مبلغ الإشعار (شامل الضريبة) — اتركه فارغاً لإلغاء الفاتورة بالكامل', 'Credit amount incl. VAT — leave empty to credit the whole invoice')}
                  </label>
                  <input id="cn-amount" type="number" min={0} step="0.001" className={inputCls} value={creditAmount}
                    onChange={e => setCreditAmount(e.target.value)} dir="ltr" />
                </div>
              )}
              <div>
                <label htmlFor="doc-reason" className="block text-xs font-medium text-gray-600 mb-1">{t('السبب', 'Reason')} *</label>
                <input id="doc-reason" className={inputCls} value={reason} onChange={e => setReason(e.target.value)} />
              </div>
              <button
                disabled={updating || !reason.trim()}
                onClick={() => run(async () => {
                  if (panel === 'void') return api.invoices.updateStatus(invoice.id, 'void', reason)
                  const amt = parseFloat(creditAmount)
                  if (!creditAmount || !Number.isFinite(amt)) return api.invoices.createCreditNote(invoice.id, { reason })
                  // A partial credit is one line in the invoice's main VAT category, net of VAT.
                  const main = invoice.lines?.[0]
                  const rate = main?.vat_rate ?? 0
                  return api.invoices.createCreditNote(invoice.id, {
                    reason,
                    lines: [{
                      description: `${t('إشعار دائن عن', 'Credit for')} ${invoice.invoice_no}`,
                      quantity: 1,
                      unit_price: Math.round((amt / (1 + rate)) * 1000) / 1000,
                      vat_category: main?.vat_category ?? 'standard',
                    }],
                  })
                })}
                className="px-4 py-2 text-sm font-medium bg-gray-900 text-white rounded-lg disabled:opacity-50"
              >
                {panel === 'void' ? t('تأكيد الإلغاء', 'Confirm void') : t('إصدار', 'Issue')}
              </button>
            </div>
          )}
          {error && <p role="alert" className="px-6 pb-4 text-xs text-red-600">{error}</p>}
        </div>

        {creditNotes.length > 0 && (
          <div className="mt-6 bg-white rounded-2xl border border-gray-100 overflow-hidden">
            <h2 className="px-6 py-3 text-sm font-semibold text-gray-700 border-b border-gray-100">{t('الإشعارات الدائنة', 'Credit notes')}</h2>
            <ul className="divide-y divide-gray-50">
              {creditNotes.map(cn => (
                <li key={cn.id} className="px-6 py-3 flex items-center justify-between text-sm">
                  <Link href={`/invoices/${cn.id}`} className="font-mono text-brand-600 hover:underline">{cn.invoice_no}</Link>
                  <span className="text-gray-500 truncate mx-3">{cn.reason}</span>
                  <span className={clsx('tabular-nums', cn.status === 'void' && 'line-through text-gray-400')}>{money(cn.total)}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="mt-4">
          <button
            onClick={() => router.push(`/deals/${invoice.deal_id}`)}
            className="text-sm text-brand-600 hover:underline flex items-center gap-1"
          >
            {t('عرض الصفقة المرتبطة', 'View linked deal')}
          </button>
        </div>
      </div>
    </div>
  )
}
