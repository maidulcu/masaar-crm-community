'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Payment, PaymentConfirmation } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  received: 'bg-green-100 text-green-700',
  overdue: 'bg-red-100 text-red-700',
  failed: 'bg-red-100 text-red-700',
  refunded: 'bg-gray-100 text-gray-600',
}

const confirmStatusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  sent: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  bounced: 'bg-gray-100 text-gray-600',
}

export default function PaymentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [payment, setPayment] = useState<Payment | null>(null)
  const [loading, setLoading] = useState(true)
  const [confirmation, setConfirmation] = useState<PaymentConfirmation | null>(null)
  const [sendingConf, setSendingConf] = useState(false)
  const [confMsg, setConfMsg] = useState('')

  const load = async () => {
    if (!id) return; setLoading(true)
    try { setPayment(await api.payments.get(id) as Payment) }
    catch {} finally { setLoading(false) }
  }

  const loadConfirmation = async () => {
    try {
      setConfirmation(await api.paymentConfirmations.getByPayment(id) as PaymentConfirmation)
    } catch {
      setConfirmation(null)
    }
  }

  useEffect(() => { load() }, [id])
  useEffect(() => { if (id) loadConfirmation() }, [id])

  const handleSendConfirmation = async () => {
    setSendingConf(true); setConfMsg('')
    try {
      await api.paymentConfirmations.send(id)
      setConfMsg(t('تم إرسال التأكيد', 'Confirmation sent'))
      loadConfirmation()
      setTimeout(() => setConfMsg(''), 5000)
    } catch {
      alert(t('فشل الإرسال', 'Failed to send confirmation'))
    } finally { setSendingConf(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'
  const fmtAmount = (v: number) => `${payment?.currency || 'AED'} ${v.toLocaleString()}`

  if (loading) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!payment) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Payment not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تفاصيل الدفعة', 'Payment Details')} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/payments" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">← {t('العودة', 'Back to Payments')}</Link>

        {confMsg && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{confMsg}</p>}

        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[payment.status])}>{payment.status}</span>
          {payment.late_fee_applied && <span className="text-xs text-red-500 font-medium">{t('متأخر', 'Late fee applied')}</span>}
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div><p className="text-xs text-gray-400 mb-1">{t('المبلغ', 'Amount')}</p><p className="font-semibold text-gray-900 text-lg">{fmtAmount(payment.amount)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الاستحقاق', 'Due Date')}</p><p className="font-medium text-gray-700">{fmtDate(payment.due_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الدفع', 'Paid Date')}</p><p className="font-medium text-gray-700">{fmtDate(payment.paid_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('طريقة الدفع', 'Method')}</p><p className="font-medium text-gray-700 capitalize">{payment.payment_method.replace('_', ' ')}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('مرجع الدفع', 'Reference')}</p><p className="font-medium text-gray-700 font-mono text-xs">{payment.payment_reference || '—'}</p></div>
          {payment.late_fee_applied && <div><p className="text-xs text-gray-400 mb-1">{t('رسوم التأخير', 'Late Fee')}</p><p className="font-medium text-red-600">{fmtAmount(payment.late_fee_amount)}</p></div>}
        </div>

        {payment.lease && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-gray-700">{t('العقد', 'Lease')}</h3>
              <Link href={`/leases/${payment.lease_id}`} className="text-xs text-brand-600 hover:underline font-medium">{t('عرض العقد', 'View Lease →')}</Link>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div><p className="text-xs text-gray-400">{t('المستأجر', 'Tenant')}</p><p className="font-medium text-gray-700">{payment.lease.tenant?.full_name_en || '—'}</p></div>
              <div><p className="text-xs text-gray-400">{t('العقار', 'Property')}</p><p className="font-medium text-gray-700">{payment.lease.property?.name || '—'}</p></div>
              <div><p className="text-xs text-gray-400">{t('الإيجار الشهري', 'Monthly Rent')}</p><p className="font-medium text-gray-700">{fmtAmount(payment.lease.monthly_rent)}</p></div>
              <div><p className="text-xs text-gray-400">{t('تاريخ النهاية', 'End Date')}</p><p className="font-medium text-gray-700">{fmtDate(payment.lease.end_date)}</p></div>
            </div>
          </div>
        )}

        {payment.notes && <div className="bg-white rounded-xl border border-gray-100 p-5"><h3 className="text-sm font-semibold text-gray-700 mb-2">{t('ملاحظات', 'Notes')}</h3><p className="text-sm text-gray-700 whitespace-pre-wrap">{payment.notes}</p></div>}
        {payment.receipt_url && <div className="bg-white rounded-xl border border-gray-100 p-5"><h3 className="text-sm font-semibold text-gray-700 mb-2">{t('الإيصال', 'Receipt')}</h3><a href={payment.receipt_url} target="_blank" rel="noopener noreferrer" className="text-sm text-brand-600 hover:underline font-medium">{t('عرض الإيصال', 'View Receipt →')}</a></div>}

        {payment.reconciled_at && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('التسوية', 'Reconciliation')}</h3>
            <p className="text-xs text-gray-500">{t('تمت التسوية في:', 'Reconciled on:')} {fmtDate(payment.reconciled_at)}</p>
          </div>
        )}

        {/* Payment Confirmation */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('تأكيد الدفع', 'Payment Confirmation')}</h3>
          {confirmation ? (
            <div className="space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', confirmStatusColor[confirmation.delivery_status])}>
                  {confirmation.delivery_status}
                </span>
                <span className="text-xs text-gray-400">
                  {t('رقم التأكيد:', 'Confirmation #:')} {confirmation.confirmation_number}
                </span>
              </div>
              <p className="text-xs text-gray-500">
                {t('بريد المستأجر:', 'Tenant email:')} {confirmation.tenant_email}
              </p>
              {confirmation.sent_at && (
                <p className="text-xs text-gray-500">
                  {t('أُرسل في:', 'Sent on:')} {fmtDate(confirmation.sent_at)}
                </p>
              )}
              {confirmation.delivery_status === 'sent' && confirmation.pdf_url && (
                <a href={confirmation.pdf_url} target="_blank" rel="noopener noreferrer"
                  className="text-xs text-brand-600 hover:underline font-medium inline-block mt-1">
                  {t('عرض PDF', 'View PDF →')}
                </a>
              )}
              {isAdmin && (
                <button onClick={handleSendConfirmation} disabled={sendingConf}
                  className="mt-2 px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {sendingConf ? '...' : t('إعادة إرسال', 'Resend')}
                </button>
              )}
            </div>
          ) : (
            <div>
              <p className="text-xs text-gray-400 mb-3">{t('لم يتم إرسال تأكيد بعد', 'No confirmation sent yet')}</p>
              {isAdmin && (
                <button onClick={handleSendConfirmation} disabled={sendingConf}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {sendingConf ? '...' : t('إرسال تأكيد الدفع', 'Send Confirmation')}
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
