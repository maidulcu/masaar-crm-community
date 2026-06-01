'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { LeaseRenewalWorkflow, RenewalCommunicationTemplate } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  in_progress: 'bg-blue-100 text-blue-700',
  offer_sent: 'bg-purple-100 text-purple-700',
  accepted: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
  expired: 'bg-gray-100 text-gray-600',
}

const responseColor: Record<string, string> = {
  pending: 'bg-gray-100 text-gray-600',
  accepted: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
  counter_offer: 'bg-blue-100 text-blue-700',
}

export default function RenewalDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()

  const [renewal, setRenewal] = useState<LeaseRenewalWorkflow | null>(null)
  const [templates, setTemplates] = useState<RenewalCommunicationTemplate[]>([])
  const [loading, setLoading] = useState(true)

  const [proposeOpen, setProposeOpen] = useState(false)
  const [offerOpen, setOfferOpen] = useState(false)
  const [counterOpen, setCounterOpen] = useState(false)

  const [proposedRent, setProposedRent] = useState('')
  const [selectedTemplate, setSelectedTemplate] = useState('')
  const [commType, setCommType] = useState('email')
  const [counterAmount, setCounterAmount] = useState('')

  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'agent' || isAdmin
  const canAct = isAgent && !['accepted', 'rejected', 'expired'].includes(renewal?.renewal_status || '')

  const load = async () => {
    if (!id) return
    setLoading(true)
    try {
      const [r, tpl] = await Promise.all([
        api.leaseRenewals.get(id) as Promise<LeaseRenewalWorkflow>,
        api.leaseRenewals.templates.list() as Promise<{ data: RenewalCommunicationTemplate[] }>,
      ])
      setRenewal(r)
      setTemplates(tpl.data ?? [])
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const doAction = async (label: string, fn: () => Promise<unknown>) => {
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      await fn()
      setSuccess(label)
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('فشلت العملية', 'Action failed'))
    } finally { setActionLoading(false) }
  }

  const handlePropose = (e: React.FormEvent) => {
    e.preventDefault()
    const amount = parseFloat(proposedRent)
    if (isNaN(amount) || amount <= 0) return
    doAction(t('تم اقتراح الإيجار', 'Proposed'), () =>
      api.leaseRenewals.propose(id, { proposed_rent_amount: amount }))
    setProposeOpen(false)
  }

  const handleSendOffer = (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedTemplate) return
    doAction(t('تم إرسال العرض', 'Offer sent'), () =>
      api.leaseRenewals.sendOffer(id, { template_id: selectedTemplate, communication_type: commType }))
    setOfferOpen(false)
  }

  const handleCounterOffer = (e: React.FormEvent) => {
    e.preventDefault()
    const amount = parseFloat(counterAmount)
    if (isNaN(amount) || amount <= 0) return
    doAction(t('تم إرسال العرض المقابل', 'Counter offer sent'), () =>
      api.leaseRenewals.counterOffer(id, { counter_offer_amount: amount }))
    setCounterOpen(false)
  }

  const fmtDate = (d: string | null) =>
    d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'
  const fmtAmount = (v: number | null | undefined) => v != null ? `AED ${v.toLocaleString()}` : '—'

  const btnCls = 'w-full text-sm font-medium rounded-lg py-2.5 disabled:opacity-50 transition-colors'
  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500'
  const labelCls = 'block text-xs font-medium text-gray-600 mb-1'

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  }
  if (!renewal) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Renewal not found')}</div>
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تجديد العقد', 'Renewal Details')} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/renewals" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">
          ← {t('العودة', 'Back to Renewals')}
        </Link>

        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}

        {/* Status & Tenant info */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[renewal.renewal_status])}>
              {renewal.renewal_status.replace('_', ' ')}
            </span>
            <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', responseColor[renewal.tenant_response])}>
              {t('رد المستأجر:', 'Tenant:')} {renewal.tenant_response.replace('_', ' ')}
            </span>
          </div>
        </div>

        {/* Key details */}
        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المستأجر', 'Tenant')}</p>
            <p className="font-semibold text-gray-900">{renewal.lease?.tenant?.full_name_en ?? '—'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('العقار', 'Property')}</p>
            <p className="font-medium text-gray-700">{renewal.lease?.property?.name ?? '—'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('تاريخ التجديد', 'Renewal Date')}</p>
            <p className="font-medium text-gray-700">{fmtDate(renewal.renewal_date)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الأيام قبل انتهاء العقد', 'Days Before Expiry')}</p>
            <p className="font-medium text-gray-700">{renewal.days_before_expiry}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الإيجار الحالي', 'Current Rent')}</p>
            <p className="font-semibold text-gray-900">{fmtAmount(renewal.lease?.monthly_rent)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الإيجار المقترح', 'Proposed Rent')}</p>
            <p className="font-semibold text-gray-900">{fmtAmount(renewal.proposed_rent_amount)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('عرض المستأجر المقابل', 'Counter Offer')}</p>
            <p className="font-medium text-gray-700">{fmtAmount(renewal.tenant_counter_offer)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('تاريخ العرض المقابل', 'Counter Offer Date')}</p>
            <p className="font-medium text-gray-700">{fmtDate(renewal.counter_offer_date)}</p>
          </div>
        </div>

        {/* Action buttons */}
        {canAct && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('الإجراءات', 'Actions')}</h3>
            <div className="flex flex-wrap gap-3">
              {renewal.renewal_status === 'pending' && (
                <button onClick={() => setProposeOpen(true)}
                  className={`${btnCls} bg-brand-600 text-white hover:bg-brand-700 px-6`}>
                  {t('اقتراح إيجار جديد', 'Propose New Rent')}
                </button>
              )}
              {['in_progress', 'offer_sent'].includes(renewal.renewal_status) && (
                <button onClick={() => setOfferOpen(true)}
                  className={`${btnCls} bg-purple-600 text-white hover:bg-purple-700 px-6`}>
                  {t('إرسال عرض', 'Send Offer')}
                </button>
              )}
              {renewal.tenant_response === 'counter_offer' && (
                <button onClick={() => setCounterOpen(true)}
                  className={`${btnCls} bg-blue-600 text-white hover:bg-blue-700 px-6`}>
                  {t('رد على العرض المقابل', 'Respond to Counter Offer')}
                </button>
              )}
              {renewal.renewal_status !== 'accepted' && ['in_progress', 'offer_sent'].includes(renewal.renewal_status) && (
                <button onClick={() => doAction(t('تم القبول', 'Accepted'), () => api.leaseRenewals.accept(id))}
                  disabled={actionLoading}
                  className={`${btnCls} bg-green-600 text-white hover:bg-green-700 px-6`}>
                  {t('قبول', 'Accept')}
                </button>
              )}
              {!['accepted', 'rejected'].includes(renewal.renewal_status) && (
                <button onClick={() => { if (confirm(t('تأكيد الرفض؟', 'Confirm rejection?'))) doAction(t('تم الرفض', 'Rejected'), () => api.leaseRenewals.reject(id)) }}
                  disabled={actionLoading}
                  className={`${btnCls} bg-red-500 text-white hover:bg-red-600 px-6`}>
                  {t('رفض', 'Reject')}
                </button>
              )}
            </div>
          </div>
        )}

        {/* Propose Modal */}
        {proposeOpen && (
          <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setProposeOpen(false)}>
            <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('اقتراح إيجار جديد', 'Propose New Rent')}</h3>
              <form onSubmit={handlePropose} className="space-y-4">
                <div>
                  <label className={labelCls}>{t('الإيجار المقترح (AED)', 'Proposed Rent (AED)')}</label>
                  <input type="number" min={0} step={100} required className={inputCls}
                    value={proposedRent} onChange={e => setProposedRent(e.target.value)}
                    placeholder={t('مثلاً: 85000', 'e.g. 85000')} />
                </div>
                <div className="flex gap-2">
                  <button type="button" onClick={() => setProposeOpen(false)}
                    className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
                    {t('إلغاء', 'Cancel')}
                  </button>
                  <button type="submit" disabled={actionLoading}
                    className="flex-1 py-2.5 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700">
                    {actionLoading ? '...' : t('تأكيد', 'Confirm')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Send Offer Modal */}
        {offerOpen && (
          <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setOfferOpen(false)}>
            <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('إرسال عرض', 'Send Offer')}</h3>
              <form onSubmit={handleSendOffer} className="space-y-4">
                <div>
                  <label className={labelCls}>{t('القناة', 'Channel')}</label>
                  <select className={inputCls} value={commType} onChange={e => setCommType(e.target.value)}>
                    <option value="email">Email</option>
                    <option value="whatsapp">WhatsApp</option>
                  </select>
                </div>
                <div>
                  <label className={labelCls}>{t('القالب', 'Template')}</label>
                  {templates.length === 0 ? (
                    <p className="text-xs text-gray-400">{t('لا توجد قوالب متاحة', 'No templates available')}</p>
                  ) : (
                    <select className={inputCls} value={selectedTemplate} onChange={e => setSelectedTemplate(e.target.value)}>
                      <option value="">{t('اختر قالباً...', 'Select template...')}</option>
                      {templates.map(tpl => (
                        <option key={tpl.id} value={tpl.id}>{tpl.template_name}</option>
                      ))}
                    </select>
                  )}
                </div>
                <div className="flex gap-2">
                  <button type="button" onClick={() => setOfferOpen(false)}
                    className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
                    {t('إلغاء', 'Cancel')}
                  </button>
                  <button type="submit" disabled={actionLoading || !selectedTemplate || templates.length === 0}
                    className="flex-1 py-2.5 bg-purple-600 text-white rounded-lg text-sm font-medium hover:bg-purple-700 disabled:opacity-50">
                    {actionLoading ? '...' : t('إرسال', 'Send')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Counter Offer Modal */}
        {counterOpen && (
          <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setCounterOpen(false)}>
            <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('الرد على العرض المقابل', 'Respond to Counter Offer')}</h3>
              {renewal.tenant_counter_offer && (
                <p className="text-xs text-gray-500 mb-3">
                  {t('عرض المستأجر:', 'Tenant counter offer:')} {fmtAmount(renewal.tenant_counter_offer)}
                </p>
              )}
              <form onSubmit={handleCounterOffer} className="space-y-4">
                <div>
                  <label className={labelCls}>{t('مبلغ العرض المقابل (AED)', 'Counter Offer Amount (AED)')}</label>
                  <input type="number" min={0} step={100} required className={inputCls}
                    value={counterAmount} onChange={e => setCounterAmount(e.target.value)}
                    placeholder={t('مثلاً: 90000', 'e.g. 90000')} />
                </div>
                <div className="flex gap-2">
                  <button type="button" onClick={() => setCounterOpen(false)}
                    className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
                    {t('إلغاء', 'Cancel')}
                  </button>
                  <button type="submit" disabled={actionLoading}
                    className="flex-1 py-2.5 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">
                    {actionLoading ? '...' : t('تأكيد', 'Confirm')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Lease details */}
        {renewal.lease && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-700">{t('تفاصيل العقد', 'Lease Details')}</h3>
              <Link href={`/leases/${renewal.lease_id}`} className="text-xs text-brand-600 hover:underline font-medium">
                {t('عرض العقد', 'View Lease →')}
              </Link>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <p className="text-xs text-gray-400">{t('الإيجار الشهري', 'Monthly Rent')}</p>
                <p className="font-medium text-gray-700">{fmtAmount(renewal.lease.monthly_rent)}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">{t('تاريخ البداية', 'Start Date')}</p>
                <p className="font-medium text-gray-700">{fmtDate(renewal.lease.start_date)}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">{t('تاريخ النهاية', 'End Date')}</p>
                <p className="font-medium text-gray-700">{fmtDate(renewal.lease.end_date)}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">{t('حالة العقد', 'Lease Status')}</p>
                <p className="font-medium text-gray-700 capitalize">{renewal.lease.status}</p>
              </div>
            </div>
          </div>
        )}

        {/* Proposed terms */}
        {renewal.proposed_terms && Object.keys(renewal.proposed_terms).length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('الشروط المقترحة', 'Proposed Terms')}</h3>
            <pre className="text-xs text-gray-600 whitespace-pre-wrap">{JSON.stringify(renewal.proposed_terms, null, 2)}</pre>
          </div>
        )}
      </div>
    </div>
  )
}
