'use client'
import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

interface Commission {
  id: string
  agent_id: string
  agent_name: string
  commission_period_start: string
  commission_period_end: string
  deals_count: number
  deals_revenue: number
  leases_count: number
  leases_revenue: number
  total_commission: number | null
  status: string
  payment_reference: string
  approval_date: string | null
  payment_date: string | null
  notes: string
  created_at: string
}

const STATUS_COLORS: Record<string, string> = {
  pending:  'bg-amber-100 text-amber-700',
  approved: 'bg-blue-100 text-blue-700',
  paid:     'bg-green-100 text-green-700',
  disputed: 'bg-red-100 text-red-700',
}

const STATUS_LABELS: Record<string, string> = {
  pending: 'Pending', approved: 'Approved', paid: 'Paid', disputed: 'Disputed',
}

const fmt = (n: number) => n.toLocaleString('en-AE', { minimumFractionDigits: 0, maximumFractionDigits: 0 })

export default function CommissionsPage() {
  const { t } = useLang()
  const { user, company } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isDemo = !!company?.is_demo

  const [commissions, setCommissions] = useState<Commission[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(1)
  const limit = 20

  // Create form
  const [showCreate, setShowCreate] = useState(false)
  const [createForm, setCreateForm] = useState({
    agent_id: '', period_from: '', period_to: '', notes: '',
  })
  const [calculated, setCalculated] = useState<any>(null)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')

  // Action state
  const [acting, setActing] = useState<string | null>(null)
  const [editID, setEditID] = useState<string | null>(null)
  const [editAmount, setEditAmount] = useState('')
  const [editNotes, setEditNotes] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const r = await api.commissions.list({ status: statusFilter || undefined, page, limit }) as any
      setCommissions(r.data ?? [])
      setTotal(r.meta?.total ?? 0)
    } catch { setCommissions([]) }
    finally { setLoading(false) }
  }, [statusFilter, page])

  useEffect(() => { load() }, [load])

  const handleCalculate = async () => {
    if (!createForm.agent_id || !createForm.period_from || !createForm.period_to) return
    try {
      const r = await api.commissions.calculate(createForm.agent_id, createForm.period_from, createForm.period_to) as any
      setCalculated(r)
    } catch (err: any) { setCreateError(err.message || 'Failed to calculate') }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isDemo) return
    setCreating(true); setCreateError('')
    try {
      await api.commissions.create({
        agent_id: createForm.agent_id,
        commission_period_start: createForm.period_from,
        commission_period_end: createForm.period_to,
        deals_count: calculated?.deals_count ?? 0,
        deals_revenue: calculated?.deals_revenue ?? 0,
        leases_count: calculated?.leases_count ?? 0,
        leases_revenue: calculated?.leases_revenue ?? 0,
        notes: createForm.notes,
      })
      setShowCreate(false)
      setCreateForm({ agent_id: '', period_from: '', period_to: '', notes: '' })
      setCalculated(null)
      load()
    } catch (err: any) { setCreateError(err.message || 'Failed') }
    finally { setCreating(false) }
  }

  const handleStatusChange = async (id: string, status: string) => {
    if (isDemo) return
    const ref = status === 'paid' ? prompt('Payment reference (optional):') ?? '' : ''
    setActing(id)
    try { await api.commissions.updateStatus(id, status, ref || undefined); load() }
    catch {}
    finally { setActing(null) }
  }

  const handleSaveAmount = async (id: string) => {
    if (isDemo) return
    setActing(id)
    try {
      await api.commissions.updateAmount(id, parseFloat(editAmount) || 0, editNotes)
      setEditID(null); load()
    } catch {}
    finally { setActing(null) }
  }

  const pages = Math.ceil(total / limit)
  const ALL_STATUSES = ['', 'pending', 'approved', 'paid', 'disputed']

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('العمولات', 'Commissions')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-5xl space-y-4">

          {/* Demo notice */}
          {isDemo && (
            <div className="flex items-center gap-2 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span><strong>Demo account</strong> — read-only. <a href="/signup" className="underline font-semibold">Start a free trial</a> to manage commissions.</span>
            </div>
          )}

          {/* Filters + Create */}
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
            <span className="text-xs text-gray-400 ml-auto">{total} {t('سجل', 'record(s)')}</span>
            {isAdmin && !isDemo && (
              <button onClick={() => setShowCreate(!showCreate)}
                className="px-4 py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 transition-colors">
                {showCreate ? t('إلغاء', 'Cancel') : t('+ إضافة عمولة', '+ New Commission')}
              </button>
            )}
          </div>

          {/* Create form */}
          {showCreate && (
            <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
              <h3 className="text-sm font-semibold text-gray-700">{t('إضافة عمولة', 'New Commission Record')}</h3>
              <form onSubmit={handleCreate} className="space-y-3">
                <div className="grid grid-cols-3 gap-3">
                  <div className="col-span-3 md:col-span-1">
                    <label className="block text-xs font-medium text-gray-600 mb-1">Agent ID</label>
                    <input value={createForm.agent_id} onChange={e => setCreateForm(p => ({ ...p, agent_id: e.target.value }))}
                      placeholder="uuid" required
                      className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">{t('من', 'From')}</label>
                    <input type="date" value={createForm.period_from} onChange={e => setCreateForm(p => ({ ...p, period_from: e.target.value }))} required
                      className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">{t('إلى', 'To')}</label>
                    <input type="date" value={createForm.period_to} onChange={e => setCreateForm(p => ({ ...p, period_to: e.target.value }))} required
                      className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                  </div>
                </div>

                <button type="button" onClick={handleCalculate}
                  className="px-4 py-2 border border-brand-300 text-brand-600 text-xs font-semibold rounded-lg hover:bg-brand-50 transition-colors">
                  {t('احسب من الصفقات والإيجارات', 'Calculate from Deals & Leases')}
                </button>

                {calculated && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {[
                      { label: 'Deals', count: calculated.deals_count, rev: calculated.deals_revenue },
                      { label: 'Leases', count: calculated.leases_count, rev: calculated.leases_revenue },
                    ].map(item => (
                      <div key={item.label} className="bg-gray-50 rounded-lg p-3 col-span-1">
                        <p className="text-[10px] text-gray-400 uppercase tracking-wide">{item.label}</p>
                        <p className="text-sm font-bold text-gray-800">{item.count} deals</p>
                        <p className="text-xs text-gray-600">AED {fmt(item.rev)}</p>
                      </div>
                    ))}
                    <div className="bg-brand-50 rounded-lg p-3 col-span-2">
                      <p className="text-[10px] text-brand-600 uppercase tracking-wide">Total Revenue</p>
                      <p className="text-lg font-bold text-brand-700">AED {fmt(calculated.total_revenue)}</p>
                    </div>
                  </div>
                )}

                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('ملاحظات', 'Notes')}</label>
                  <input value={createForm.notes} onChange={e => setCreateForm(p => ({ ...p, notes: e.target.value }))}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                {createError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{createError}</p>}
                <button type="submit" disabled={creating}
                  className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50">
                  {creating ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ العمولة', 'Save Commission')}
                </button>
              </form>
            </div>
          )}

          {/* Commission list */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            {loading ? (
              <div className="p-12 text-center text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : commissions.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-sm text-gray-400">{t('لا توجد سجلات عمولات', 'No commission records yet')}</p>
                {isAdmin && !isDemo && <p className="text-xs text-gray-400 mt-1">Create a commission record for an agent above.</p>}
              </div>
            ) : (
              <div className="divide-y divide-gray-100">
                {commissions.map(c => (
                  <div key={c.id} className="p-4">
                    <div className="flex items-start gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full ${STATUS_COLORS[c.status]}`}>
                            {STATUS_LABELS[c.status]}
                          </span>
                          <span className="text-xs text-gray-500 font-medium">{c.agent_name || c.agent_id.slice(0, 8)}</span>
                        </div>
                        <div className="flex items-center gap-3 text-xs text-gray-500">
                          <span>{new Date(c.commission_period_start).toLocaleDateString()} → {new Date(c.commission_period_end).toLocaleDateString()}</span>
                          <span>·</span>
                          <span>{c.deals_count} deals (AED {fmt(c.deals_revenue)})</span>
                          <span>·</span>
                          <span>{c.leases_count} leases (AED {fmt(c.leases_revenue)})</span>
                        </div>
                        {c.payment_date && (
                          <p className="text-xs text-green-600 mt-0.5">
                            Paid {new Date(c.payment_date).toLocaleDateString()}{c.payment_reference ? ` · ${c.payment_reference}` : ''}
                          </p>
                        )}
                        {c.notes && <p className="text-xs text-gray-400 mt-0.5 line-clamp-1">{c.notes}</p>}
                      </div>

                      {/* Amount */}
                      <div className="shrink-0 text-right">
                        {editID === c.id ? (
                          <div className="flex items-center gap-1">
                            <input type="number" value={editAmount} onChange={e => setEditAmount(e.target.value)}
                              className="w-28 text-sm border border-gray-200 rounded-lg px-2 py-1 text-right" />
                            <button onClick={() => handleSaveAmount(c.id)} disabled={acting === c.id}
                              className="px-2 py-1 bg-brand-600 text-white text-xs rounded-md">✓</button>
                            <button onClick={() => setEditID(null)} className="px-2 py-1 text-gray-400 text-xs">✕</button>
                          </div>
                        ) : (
                          <button onClick={() => { if (!isDemo && isAdmin) { setEditID(c.id); setEditAmount(String(c.total_commission ?? 0)); setEditNotes(c.notes) } }}
                            className={`text-sm font-bold text-gray-800 ${isAdmin && !isDemo ? 'hover:text-brand-600 cursor-pointer' : 'cursor-default'}`}>
                            {c.total_commission != null ? `AED ${fmt(c.total_commission)}` : '—'}
                          </button>
                        )}
                      </div>

                      {/* Status actions */}
                      {isAdmin && !isDemo && c.status !== 'paid' && (
                        <div className="flex items-center gap-1 shrink-0">
                          {c.status === 'pending' && (
                            <button onClick={() => handleStatusChange(c.id, 'approved')} disabled={acting === c.id}
                              className="text-xs px-2 py-1 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50">
                              Approve
                            </button>
                          )}
                          {c.status === 'approved' && (
                            <button onClick={() => handleStatusChange(c.id, 'paid')} disabled={acting === c.id}
                              className="text-xs px-2 py-1 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50">
                              Mark Paid
                            </button>
                          )}
                          <button onClick={() => handleStatusChange(c.id, 'disputed')} disabled={acting === c.id}
                            className="text-xs px-2 py-1 text-red-500 hover:bg-red-50 rounded-md">
                            Dispute
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {pages > 1 && (
            <div className="flex justify-center gap-1">
              {Array.from({ length: pages }, (_, i) => i + 1).map(p => (
                <button key={p} onClick={() => setPage(p)}
                  className={`w-8 h-8 rounded-lg text-sm font-medium transition-colors ${p === page ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>
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
