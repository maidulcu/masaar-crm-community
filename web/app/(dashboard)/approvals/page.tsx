'use client'
import { useState, useEffect, useCallback } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { ApprovalRequest, PaginatedResult } from '@/types'

const STATUS_COLORS: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-700',
  approved: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-600',
}

const STATUS_LABELS: Record<string, { en: string; ar: string }> = {
  pending: { en: 'Pending', ar: 'قيد الانتظار' },
  approved: { en: 'Approved', ar: 'تمت الموافقة' },
  rejected: { en: 'Rejected', ar: 'مرفوض' },
}

const ENTITY_LABELS: Record<string, { en: string; ar: string }> = {
  listing: { en: 'Listing', ar: 'إعلان' },
  deal: { en: 'Deal', ar: 'صفقة' },
  offer: { en: 'Offer', ar: 'عرض' },
}

export default function ApprovalsPage() {
  const { user, company } = useAuthStore()
  const { t, lang } = useLang()
  const isDemo = !!company?.is_demo

  const [requests, setRequests] = useState<ApprovalRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [filter, setFilter] = useState('pending')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.approvals.listRequests({ status: filter, page, limit: 50 }) as PaginatedResult<ApprovalRequest>
      setRequests(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch {
      setRequests([])
    } finally { setLoading(false) }
  }, [filter, page])

  useEffect(() => { load() }, [load])

  const handleReview = async (id: string, status: 'approved' | 'rejected') => {
    if (isDemo) return
    const note = status === 'rejected' ? prompt(t('سبب الرفض:', 'Reason for rejection:')) || '' : ''
    try {
      await api.approvals.review(id, status, note)
      load()
    } catch {}
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('طلبات الموافقة', 'Approval Requests')} />

      <div className="flex-1 overflow-auto p-6">
        {/* Filter tabs */}
        <div className="flex gap-1 mb-4 bg-white rounded-lg border border-gray-200 p-1 w-fit">
          {['pending', 'approved', 'rejected', ''].map((f) => (
            <button key={f}
              onClick={() => { setFilter(f); setPage(1) }}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                filter === f ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-100'
              }`}>
              {f === '' ? t('الكل', 'All') : t(STATUS_LABELS[f]?.ar || f, STATUS_LABELS[f]?.en || f)}
            </button>
          ))}
        </div>

        {isDemo && (
          <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800 mb-4">
            <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
            </svg>
            <span><strong>Demo account</strong> — read-only.</span>
          </div>
        )}

        {loading ? (
          <div className="text-center py-12 text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
        ) : requests.length === 0 ? (
          <div className="text-center py-12 text-sm text-gray-500">{t('لا توجد طلبات', 'No approval requests')}</div>
        ) : (
          <div className="space-y-2">
            {requests.map((req) => (
              <div key={req.id} className="bg-white rounded-xl border border-gray-200 p-4 flex items-center gap-4">
                {/* Entity badge */}
                <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full ${
                  req.entity_type === 'listing' ? 'bg-blue-50 text-blue-700' :
                  req.entity_type === 'deal' ? 'bg-purple-50 text-purple-700' :
                  'bg-orange-50 text-orange-700'
                }`}>
                  {lang === 'ar' ? ENTITY_LABELS[req.entity_type]?.ar : ENTITY_LABELS[req.entity_type]?.en}
                </span>

                {/* Status */}
                <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full ${STATUS_COLORS[req.status] || 'bg-gray-100 text-gray-700'}`}>
                  {lang === 'ar' ? STATUS_LABELS[req.status]?.ar : STATUS_LABELS[req.status]?.en}
                </span>

                {/* Details */}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-800 truncate">
                    {t('طلب من', 'Requested by')} <span className="font-semibold">{req.requester_name || req.requested_by.slice(0, 8)}</span>
                  </p>
                  <p className="text-[11px] text-gray-500">
                    {req.entity_type === 'listing' && (
                      <Link href={`/listings/${req.entity_id}`} className="hover:text-brand-600 underline">
                        {t('عرض الإعلان', 'View Listing')}
                      </Link>
                    )}
                    {req.notes && <span className="ml-2">— {req.notes}</span>}
                  </p>
                  {req.reviewer_note && (
                    <p className="text-[11px] text-gray-500 mt-0.5 italic">"{req.reviewer_note}"</p>
                  )}
                </div>

                {/* Actions */}
                <div className="flex gap-1.5 shrink-0">
                  {req.status === 'pending' && !isDemo && (
                    <>
                      <button onClick={() => handleReview(req.id, 'approved')}
                        className="px-3 py-1.5 text-[11px] font-medium bg-green-50 text-green-700 rounded-lg hover:bg-green-100 transition-colors">
                        {t('موافقة', 'Approve')}
                      </button>
                      <button onClick={() => handleReview(req.id, 'rejected')}
                        className="px-3 py-1.5 text-[11px] font-medium bg-red-50 text-red-600 rounded-lg hover:bg-red-100 transition-colors">
                        {t('رفض', 'Reject')}
                      </button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        {total > 50 && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-gray-200 disabled:opacity-40 hover:bg-gray-50">
              {t('السابق', 'Previous')}
            </button>
            <span className="px-3 py-1.5 text-xs text-gray-500">{page} / {Math.ceil(total / 50)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page >= Math.ceil(total / 50)}
              className="px-3 py-1.5 text-xs font-medium rounded-lg border border-gray-200 disabled:opacity-40 hover:bg-gray-50">
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
