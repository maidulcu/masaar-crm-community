'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { LeaseRenewalWorkflow, PaginatedResult } from '@/types'

const STATUS_FILTERS = ['', 'pending', 'in_progress', 'offer_sent', 'accepted', 'rejected', 'expired']

const statusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  in_progress: 'bg-blue-100 text-blue-700',
  offer_sent: 'bg-purple-100 text-purple-700',
  accepted: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
  expired: 'bg-gray-100 text-gray-600',
}

export default function RenewalsPage() {
  const { t, lang } = useLang()
  const [renewals, setRenewals] = useState<LeaseRenewalWorkflow[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState('')
  const limit = 20

  useEffect(() => { load() }, [page, statusFilter])

  const load = async () => {
    setLoading(true)
    try {
      const r = await api.leaseRenewals.list({ page, limit, status: statusFilter || undefined }) as PaginatedResult<LeaseRenewalWorkflow>
      setRenewals(r.data ?? [])
      setTotal(r.total ?? 0)
    } catch { setRenewals([]) } finally { setLoading(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'
  const fmtAmount = (v: number | null | undefined) => v != null ? `AED ${v.toLocaleString()}` : '—'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تجديد العقود', 'Lease Renewals')} />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-800">{t('تجديد العقود', 'Renewals')} ({total})</h2>
          <div className="flex gap-1">
            {STATUS_FILTERS.map(s => (
              <button key={s} onClick={() => { setStatusFilter(s); setPage(1) }}
                className={`text-xs px-3 py-1.5 rounded-lg font-medium transition-colors ${statusFilter === s ? 'bg-brand-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}>
                {s ? t(s.replace('_', ' '), s.replace('_', ' ')) : t('الكل', 'All')}
              </button>
            ))}
          </div>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : renewals.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
            <p className="text-gray-400 text-sm">{t('لا توجد طلبات تجديد', 'No renewal requests yet')}</p>
          </div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  {[t('المستأجر', 'Tenant'), t('العقار', 'Property'), t('تاريخ التجديد', 'Renewal Date'), t('الإيجار الحالي', 'Current Rent'), t('الإيجار المقترح', 'Proposed Rent'), t('الحالة', 'Status')].map(h => (
                    <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {renewals.map(r => (
                  <tr key={r.id} className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer">
                    <td className="px-4 py-3">
                      <Link href={`/renewals/${r.id}`} className="font-medium text-gray-900 hover:text-brand-600">
                        {r.lease?.tenant?.full_name_en || '-'}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gray-600">{r.lease?.property?.name || '-'}</td>
                    <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(r.renewal_date)}</td>
                    <td className="px-4 py-3 text-gray-700">{fmtAmount(r.lease?.monthly_rent)}</td>
                    <td className="px-4 py-3 text-gray-700">{fmtAmount(r.proposed_rent_amount)}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${statusColor[r.renewal_status] || ''}`}>
                        {r.renewal_status.replace('_', ' ')}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {total > limit && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('السابق', 'Prev')}
            </button>
            <span className="flex items-center px-4 text-sm text-gray-500">{page} / {Math.ceil(total / limit)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
