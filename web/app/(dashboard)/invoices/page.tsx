'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { VATInvoice, InvoiceStatus } from '@/types'

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  draft: 'bg-gray-100 text-gray-600',
  sent:  'bg-blue-100 text-blue-700',
  paid:  'bg-green-100 text-green-700',
}

const STATUS_LABELS: Record<InvoiceStatus, { en: string; ar: string }> = {
  draft: { en: 'Draft', ar: 'مسودة' },
  sent:  { en: 'Sent',  ar: 'مُرسلة' },
  paid:  { en: 'Paid',  ar: 'مدفوعة' },
}

function fmtCurrency(n: number) {
  return new Intl.NumberFormat('en-AE', { style: 'currency', currency: 'AED', minimumFractionDigits: 2 }).format(n)
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export default function InvoicesPage() {
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const router = useRouter()

  const [invoices, setInvoices] = useState<VATInvoice[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 50

  const [statusFilter, setStatusFilter] = useState<InvoiceStatus | ''>('')

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res: any = await api.invoices.list(page, limit)
      setInvoices(res?.data ?? [])
      setTotal(res?.total ?? 0)
    } catch {
      setInvoices([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))
  const filtered = statusFilter ? invoices.filter(i => i.status === statusFilter) : invoices

  const totals = filtered.reduce(
    (acc, inv) => ({
      subtotal: acc.subtotal + inv.subtotal,
      vat: acc.vat + inv.vat_amount,
      total: acc.total + inv.total,
    }),
    { subtotal: 0, vat: 0, total: 0 }
  )

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الفواتير', 'Invoices')} />

      <div className="flex-1 overflow-y-auto p-6">
        {/* Summary cards */}
        {!loading && invoices.length > 0 && (
          <div className="grid grid-cols-3 gap-4 mb-6">
            {[
              { label: t('الإجمالي قبل الضريبة', 'Subtotal'), value: fmtCurrency(totals.subtotal) },
              { label: t('ضريبة القيمة المضافة', 'VAT (5%)'), value: fmtCurrency(totals.vat) },
              { label: t('الإجمالي النهائي', 'Total'), value: fmtCurrency(totals.total), highlight: true },
            ].map(card => (
              <div key={card.label} className={clsx(
                'rounded-xl border p-4',
                card.highlight ? 'bg-brand-50 border-brand-100' : 'bg-white border-gray-100'
              )}>
                <p className="text-xs text-gray-400 mb-1">{card.label}</p>
                <p className={clsx('text-lg font-semibold tabular-nums', card.highlight ? 'text-brand-700' : 'text-gray-900')}>
                  {card.value}
                </p>
              </div>
            ))}
          </div>
        )}

        {/* Status filters */}
        <div className="flex items-center gap-3 mb-4">
          <button
            onClick={() => setStatusFilter('')}
            className={clsx(
              'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
              statusFilter === '' ? 'bg-brand-600 text-white' : 'bg-surface-100 text-surface-600 hover:bg-surface-200'
            )}
          >
            {t('الكل', 'All')} ({total})
          </button>
          {(['draft', 'sent', 'paid'] as InvoiceStatus[]).map(s => (
            <button key={s} onClick={() => setStatusFilter(s)}
              className={clsx(
                'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                statusFilter === s ? 'bg-brand-600 text-white' : 'bg-surface-100 text-surface-600 hover:bg-surface-200'
              )}
            >
              {lang === 'ar' ? STATUS_LABELS[s].ar : STATUS_LABELS[s].en}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-3">
            <div className="w-14 h-14 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-7 h-7 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 14l6-6m-5.5.5h.01m4.99 5h.01M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16l3.5-2 3.5 2 3.5-2 3.5 2z" />
              </svg>
            </div>
            <p className="text-sm text-gray-400">{t('لا توجد فواتير', 'No invoices found')}</p>
          </div>
        ) : (
          <>
            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('رقم الفاتورة', 'Invoice No.')}</th>
                    <th className="text-start px-5 py-3 font-medium hidden md:table-cell">{t('تاريخ الإصدار', 'Issued')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('المبلغ', 'Subtotal')}</th>
                    <th className="text-start px-5 py-3 font-medium hidden md:table-cell">{t('الضريبة', 'VAT')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الإجمالي', 'Total')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {filtered.map(inv => (
                    <tr
                      key={inv.id}
                      className="hover:bg-gray-50 transition-colors cursor-pointer"
                      onClick={() => router.push(`/invoices/${inv.id}`)}
                    >
                      <td className="px-5 py-3.5 font-mono text-sm font-medium text-gray-900">
                        {inv.invoice_no}
                      </td>
                      <td className="px-5 py-3.5 text-gray-500 hidden md:table-cell">
                        {fmtDate(inv.issued_at)}
                      </td>
                      <td className="px-5 py-3.5 tabular-nums text-gray-700">
                        {fmtCurrency(inv.subtotal)}
                      </td>
                      <td className="px-5 py-3.5 tabular-nums text-gray-500 hidden md:table-cell">
                        {fmtCurrency(inv.vat_amount)}
                      </td>
                      <td className="px-5 py-3.5 tabular-nums font-semibold text-gray-900">
                        {fmtCurrency(inv.total)}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', STATUS_COLORS[inv.status])}>
                          {lang === 'ar' ? STATUS_LABELS[inv.status].ar : STATUS_LABELS[inv.status].en}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-end" onClick={e => e.stopPropagation()}>
                        <a
                          href={api.invoices.getPDF(inv.id)}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-brand-600 hover:text-brand-700 font-medium"
                        >
                          PDF
                        </a>
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
                {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                  const p = page <= 3 ? i + 1 : page - 2 + i
                  return p <= totalPages ? (
                    <button key={p} onClick={() => setPage(p)}
                      className={clsx('px-3 py-1.5 text-sm rounded-lg', p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50')}>
                      {p}
                    </button>
                  ) : null
                })}
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
