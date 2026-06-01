'use client'
import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import Link from 'next/link'

interface Offer {
  id: string
  listing_id: string
  listing_title: string
  contact: { id: string; full_name: string; phone_wa: string; email: string }
  offer_amount: number
  currency: string
  status: string
  terms: string
  notes: string
  valid_until: string | null
  deal_id: string | null
  parent_offer_id: string | null
  created_at: string
}

const STATUS_COLORS: Record<string, string> = {
  submitted:    'bg-blue-100 text-blue-700',
  under_review: 'bg-amber-100 text-amber-700',
  countered:    'bg-purple-100 text-purple-700',
  accepted:     'bg-green-100 text-green-700',
  rejected:     'bg-red-100 text-red-700',
  expired:      'bg-gray-100 text-gray-500',
}

const STATUS_LABELS: Record<string, string> = {
  submitted: 'Submitted', under_review: 'Under Review', countered: 'Countered',
  accepted: 'Accepted', rejected: 'Rejected', expired: 'Expired',
}

const ALL_STATUSES = ['', 'submitted', 'under_review', 'countered', 'accepted', 'rejected', 'expired']

export default function OffersPage() {
  const { t } = useLang()
  const { user, company } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const isDemo = !!company?.is_demo

  const [offers, setOffers] = useState<Offer[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(1)
  const limit = 20

  // Action state
  const [acting, setActing] = useState<string | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const r = await api.offers.list({ status: statusFilter || undefined, page, limit }) as any
      setOffers(r.data ?? [])
      setTotal(r.meta?.total ?? 0)
    } catch { setOffers([]) }
    finally { setLoading(false) }
  }, [statusFilter, page])

  useEffect(() => { load() }, [load])

  const handleStatusChange = async (id: string, status: string) => {
    setActing(id); setError('')
    try {
      await api.offers.updateStatus(id, status)
      await load()
    } catch (err: any) { setError(err.message || 'Failed') }
    finally { setActing(null) }
  }

  const handleAccept = async (id: string) => {
    if (!confirm('Accept this offer and create a Deal?')) return
    setActing(id); setError('')
    try {
      const r = await api.offers.accept(id) as any
      await load()
      if (r.deal_id) {
        window.location.href = `/deals`
      }
    } catch (err: any) { setError(err.message || 'Failed to accept') }
    finally { setActing(null) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this offer?')) return
    setActing(id)
    try { await api.offers.delete(id); await load() }
    catch { setError('Failed to delete') }
    finally { setActing(null) }
  }

  const pages = Math.ceil(total / limit)

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('العروض', 'Offers')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-5xl space-y-4">

          {/* Filters */}
          <div className="flex items-center gap-3 flex-wrap">
            <div className="flex items-center gap-1 bg-white border border-gray-200 rounded-lg p-1">
              {ALL_STATUSES.map(s => (
                <button key={s} onClick={() => { setStatusFilter(s); setPage(1) }}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    statusFilter === s ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-50'
                  }`}>
                  {s ? STATUS_LABELS[s] : t('الكل', 'All')}
                </button>
              ))}
            </div>
            <span className="text-xs text-gray-400 ml-auto">{total} {t('عرض', 'offer(s)')}</span>
          </div>

          {isDemo && (
            <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span><strong>Demo account</strong> — offer actions are read-only. <a href="/signup" className="underline font-semibold">Start a free trial</a> to manage offers.</span>
            </div>
          )}

          {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}

          {/* Table */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            {loading ? (
              <div className="p-12 text-center text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : offers.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-sm text-gray-400">{t('لا توجد عروض', 'No offers yet')}</p>
                <p className="text-xs text-gray-400 mt-1">Create offers from the listing detail page.</p>
              </div>
            ) : (
              <div className="divide-y divide-gray-100">
                {offers.map(offer => (
                  <div key={offer.id} className="p-4 flex items-start gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full ${STATUS_COLORS[offer.status]}`}>
                          {STATUS_LABELS[offer.status]}
                        </span>
                        {offer.parent_offer_id && (
                          <span className="text-[10px] text-gray-400">counter-offer</span>
                        )}
                        {offer.deal_id && (
                          <Link href="/deals" className="text-[10px] text-green-600 hover:underline">→ Deal created</Link>
                        )}
                      </div>

                      <p className="text-sm font-semibold text-gray-800 truncate">
                        <Link href={`/listings/${offer.listing_id}`} className="hover:text-brand-600">
                          {offer.listing_title || 'Listing'}
                        </Link>
                      </p>

                      <div className="flex items-center gap-3 mt-1 text-xs text-gray-500">
                        <span className="font-semibold text-gray-800">
                          {offer.currency} {offer.offer_amount.toLocaleString()}
                        </span>
                        <span>·</span>
                        <Link href={`/contacts/${offer.contact?.id}`} className="hover:text-brand-600">
                          {offer.contact?.full_name}
                        </Link>
                        <span>·</span>
                        <span>{offer.contact?.phone_wa}</span>
                        {offer.valid_until && (
                          <>
                            <span>·</span>
                            <span>Valid until {new Date(offer.valid_until).toLocaleDateString()}</span>
                          </>
                        )}
                      </div>

                      {offer.terms && (
                        <p className="text-xs text-gray-400 mt-1 line-clamp-1">{offer.terms}</p>
                      )}
                    </div>

                    {/* Actions */}
                    {isAgent && !isDemo && offer.status !== 'accepted' && offer.status !== 'rejected' && offer.status !== 'expired' && (
                      <div className="flex items-center gap-1 shrink-0">
                        {offer.status !== 'under_review' && (
                          <button
                            onClick={() => handleStatusChange(offer.id, 'under_review')}
                            disabled={acting === offer.id}
                            className="text-xs px-2 py-1.5 border border-gray-200 rounded-lg text-gray-600 hover:bg-gray-50 transition-colors disabled:opacity-50">
                            Review
                          </button>
                        )}
                        <button
                          onClick={() => handleAccept(offer.id)}
                          disabled={acting === offer.id}
                          className="text-xs px-2 py-1.5 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50">
                          {acting === offer.id ? '...' : 'Accept'}
                        </button>
                        <button
                          onClick={() => handleStatusChange(offer.id, 'rejected')}
                          disabled={acting === offer.id}
                          className="text-xs px-2 py-1.5 text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50">
                          Reject
                        </button>
                      </div>
                    )}

                    {user?.role === 'admin' && !isDemo && (
                      <button
                        onClick={() => handleDelete(offer.id)}
                        disabled={acting === offer.id}
                        className="text-xs text-gray-400 hover:text-red-500 px-1 py-1 rounded shrink-0">
                        ✕
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Pagination */}
          {pages > 1 && (
            <div className="flex justify-center gap-1">
              {Array.from({ length: pages }, (_, i) => i + 1).map(p => (
                <button key={p} onClick={() => setPage(p)}
                  className={`w-8 h-8 rounded-lg text-sm font-medium transition-colors ${
                    p === page ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-100'
                  }`}>
                  {p}
                </button>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
