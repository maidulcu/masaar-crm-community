'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
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

export default function InvoiceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const router = useRouter()

  const [invoice, setInvoice] = useState<VATInvoice | null>(null)
  const [loading, setLoading] = useState(true)
  const [updating, setUpdating] = useState(false)

  useEffect(() => { load() }, [id])

  const load = async () => {
    setLoading(true)
    try {
      const inv = await api.invoices.get(id) as VATInvoice
      setInvoice(inv)
    } catch {
      setInvoice(null)
    } finally {
      setLoading(false)
    }
  }

  const handleSend = async () => {
    if (!invoice) return
    setUpdating(true)
    try {
      await api.invoices.send(invoice.id)
      await load()
    } finally {
      setUpdating(false)
    }
  }

  const handleMarkPaid = async () => {
    if (!invoice) return
    setUpdating(true)
    try {
      await api.invoices.updateStatus(invoice.id, 'paid')
      await load()
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

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={invoice.invoice_no} />

      <div className="flex-1 overflow-y-auto p-6 max-w-2xl">
        {/* Breadcrumb */}
        <button
          onClick={() => router.push('/invoices')}
          className="text-xs text-gray-400 hover:text-brand-600 mb-6 flex items-center gap-1 transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          {t('جميع الفواتير', 'All Invoices')}
        </button>

        {/* Invoice card */}
        <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
          {/* Header */}
          <div className="px-6 py-5 border-b border-gray-100 flex items-center justify-between">
            <div>
              <p className="font-mono text-lg font-semibold text-gray-900">{invoice.invoice_no}</p>
              <p className="text-xs text-gray-400 mt-0.5">{t('تاريخ الإصدار', 'Issued')} {fmtDate(invoice.issued_at)}</p>
            </div>
            <span className={clsx('text-sm font-medium px-3 py-1 rounded-full', STATUS_COLORS[invoice.status])}>
              {lang === 'ar' ? STATUS_LABELS[invoice.status].ar : STATUS_LABELS[invoice.status].en}
            </span>
          </div>

          {/* Line items */}
          <div className="px-6 py-5 space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">{t('المبلغ قبل الضريبة', 'Subtotal')}</span>
              <span className="tabular-nums text-gray-900">{fmtCurrency(invoice.subtotal)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-500">{t('ضريبة القيمة المضافة', 'VAT')} ({(invoice.vat_rate * 100).toFixed(0)}%)</span>
              <span className="tabular-nums text-gray-900">{fmtCurrency(invoice.vat_amount)}</span>
            </div>
            <div className="flex justify-between text-sm font-semibold border-t border-gray-100 pt-3">
              <span className="text-gray-900">{t('الإجمالي', 'Total')}</span>
              <span className="tabular-nums text-gray-900">{fmtCurrency(invoice.total)}</span>
            </div>
          </div>

          {/* Actions */}
          <div className="px-6 py-4 bg-gray-50 border-t border-gray-100 flex items-center gap-3">
            <a
              href={api.invoices.getPDF(invoice.id)}
              target="_blank"
              rel="noopener noreferrer"
              className="px-4 py-2 text-sm font-medium border border-gray-200 rounded-lg bg-white hover:bg-gray-50 transition-colors text-gray-700 flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              {t('تحميل PDF', 'Download PDF')}
            </a>

            {isAdmin && invoice.status === 'draft' && (
              <button
                onClick={handleSend}
                disabled={updating}
                className="px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {updating ? '...' : t('إرسال', 'Mark as Sent')}
              </button>
            )}

            {isAdmin && invoice.status === 'sent' && (
              <button
                onClick={handleMarkPaid}
                disabled={updating}
                className="px-4 py-2 text-sm font-medium bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 transition-colors"
              >
                {updating ? '...' : t('تأكيد الدفع', 'Mark as Paid')}
              </button>
            )}

            {invoice.status === 'paid' && (
              <span className="text-sm text-green-600 font-medium flex items-center gap-1.5">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                {t('تم الدفع', 'Payment received')}
              </span>
            )}
          </div>
        </div>

        {/* Deal link */}
        <div className="mt-4">
          <button
            onClick={() => router.push(`/deals/${invoice.deal_id}`)}
            className="text-sm text-brand-600 hover:underline flex items-center gap-1"
          >
            {t('عرض الصفقة المرتبطة', 'View linked deal')}
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}
