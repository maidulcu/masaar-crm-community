'use client'

import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { Lease, Payment, PaginatedResult } from '@/types'
import DocumentAttachmentSection from '@/components/DocumentAttachmentSection'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  active:     'bg-green-100 text-green-700',
  expired:    'bg-gray-100 text-gray-600',
  terminated: 'bg-red-100 text-red-700',
  pending:    'bg-blue-100 text-blue-700',
}

const paymentStatusColor: Record<string, string> = {
  pending:  'bg-yellow-100 text-yellow-700',
  received: 'bg-green-100 text-green-700',
  overdue:  'bg-red-100 text-red-700',
  failed:   'bg-red-100 text-red-700',
  refunded: 'bg-gray-100 text-gray-600',
}

export default function LeaseDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()

  const [lease, setLease] = useState<Lease | null>(null)
  const [payments, setPayments] = useState<Payment[]>([])
  const [loading, setLoading] = useState(true)

  const load = async () => {
    if (!id) return
    setLoading(true)
    try {
      const [leaseData, paymentsData] = await Promise.all([
        api.leases.get(id) as Promise<Lease>,
        api.payments.list({ limit: 100 }) as Promise<PaginatedResult<Payment>>,
      ])
      setLease(leaseData)
      // filter payments to only this lease
      const leasePayments = (paymentsData.data ?? []).filter((p) => p.lease_id === id)
      setPayments(leasePayments)
    } catch (err) {
      console.error('Failed to load lease:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [id])

  const fmtDate = (d: string | null) =>
    d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', {
      year: 'numeric', month: 'short', day: 'numeric',
    }) : '—'

  const fmtAmount = (amount: number, currency = 'AED') =>
    new Intl.NumberFormat(lang === 'ar' ? 'ar-AE' : 'en-AE', {
      style: 'currency', currency,
    }).format(amount)

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        {t('جاري التحميل...', 'Loading...')}
      </div>
    )
  }

  if (!lease) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        {t('العقد غير موجود', 'Lease not found')}
      </div>
    )
  }

  const totalPaid = payments.filter((p) => p.status === 'received').reduce((s, p) => s + p.amount, 0)
  const totalPending = payments.filter((p) => p.status === 'pending' || p.status === 'overdue').reduce((s, p) => s + p.amount, 0)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={lease.tenant?.full_name_en ?? t('تفاصيل العقد', 'Lease Details')} />

      <div className="flex-1 overflow-y-auto p-6 space-y-6">

        {/* Back */}
        <Link href="/leases" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">
          ← {t('العودة للعقود', 'Back to Leases')}
        </Link>

        {/* Status badge */}
        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[lease.status])}>
            {lease.status}
          </span>
          {lease.ejari_number && (
            <span className="text-xs text-gray-500 font-mono">Ejari: {lease.ejari_number}</span>
          )}
        </div>

        {/* Key details grid */}
        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المستأجر', 'Tenant')}</p>
            <p className="font-semibold text-gray-900">{lease.tenant?.full_name_en ?? '—'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('العقار', 'Property')}</p>
            <p className="font-medium text-gray-700">{lease.property?.name ?? '—'}</p>
            {lease.property?.unit_number && (
              <p className="text-xs text-gray-500">Unit {lease.property.unit_number}</p>
            )}
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الإيجار الشهري', 'Monthly Rent')}</p>
            <p className="font-semibold text-gray-900">{fmtAmount(lease.monthly_rent, lease.currency)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('مبلغ التأمين', 'Security Deposit')}</p>
            <p className="font-medium text-gray-700">{fmtAmount(lease.security_deposit, lease.currency)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('تاريخ البداية', 'Start Date')}</p>
            <p className="font-medium text-gray-700">{fmtDate(lease.start_date)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('تاريخ النهاية', 'End Date')}</p>
            <p className="font-medium text-gray-700">{fmtDate(lease.end_date)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('دورية الدفع', 'Payment Frequency')}</p>
            <p className="font-medium text-gray-700 capitalize">{lease.payment_frequency.replace('_', ' ')}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('يوم الدفع', 'Payment Day')}</p>
            <p className="font-medium text-gray-700">Day {lease.payment_day_of_month}</p>
          </div>
        </div>

        {/* Payment summary */}
        {payments.length > 0 && (
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-green-50 border border-green-100 rounded-xl p-4">
              <p className="text-xs text-green-600 font-medium mb-1">{t('مدفوع', 'Total Paid')}</p>
              <p className="font-bold text-green-900">{fmtAmount(totalPaid, lease.currency)}</p>
            </div>
            <div className="bg-yellow-50 border border-yellow-100 rounded-xl p-4">
              <p className="text-xs text-yellow-600 font-medium mb-1">{t('معلق', 'Pending')}</p>
              <p className="font-bold text-yellow-900">{fmtAmount(totalPending, lease.currency)}</p>
            </div>
            <div className="bg-blue-50 border border-blue-100 rounded-xl p-4">
              <p className="text-xs text-blue-600 font-medium mb-1">{t('إجمالي الدفعات', 'Total Payments')}</p>
              <p className="font-bold text-blue-900">{payments.length}</p>
            </div>
          </div>
        )}

        {/* Payments table */}
        {payments.length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100">
              <h2 className="font-semibold text-gray-900 text-sm">{t('الدفعات', 'Payments')}</h2>
            </div>
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-100">
                <tr className="text-xs text-gray-400 uppercase tracking-wide">
                  <th className="text-start px-5 py-3 font-medium">{t('المبلغ', 'Amount')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('تاريخ الاستحقاق', 'Due Date')}</th>
                  <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {payments.map((p) => (
                  <tr key={p.id} className="hover:bg-gray-50">
                    <td className="px-5 py-3.5 font-medium text-gray-900">{fmtAmount(p.amount, lease.currency)}</td>
                    <td className="px-5 py-3.5 text-gray-500 text-xs">{fmtDate(p.due_date)}</td>
                    <td className="px-5 py-3.5">
                      <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full capitalize', paymentStatusColor[p.status])}>
                        {p.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Documents */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <DocumentAttachmentSection entityType="lease" entityId={id} canEdit />
        </div>

        {/* Notes */}
        {lease.notes && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide mb-3">{t('ملاحظات', 'Notes')}</h3>
            <p className="text-sm text-gray-700 whitespace-pre-wrap">{lease.notes}</p>
          </div>
        )}
      </div>
    </div>
  )
}
