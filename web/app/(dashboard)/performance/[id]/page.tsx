'use client'
import { useState, useEffect, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

interface KPIs {
  agent_id: string; agent_name: string; period: string
  deals_won: number; revenue: number
  listings_added: number; leads_assigned: number; leads_converted: number
  avg_response_hours: number
  badges: { key: string; label: string; icon: string }[]
}
interface Trend {
  metric: string; current: number; prev_month: number; prev_year: number
  change_month_pct: number; change_year_pct: number
}
interface Target {
  id: string; metric: string; target_value: number
  current_value: number; progress_pct: number
}

const METRIC_LABELS: Record<string, string> = {
  deals_won: 'Deals Won', revenue: 'Revenue (AED)',
  listings_added: 'Listings', leads_converted: 'Leads Converted', leads_assigned: 'Leads Assigned',
}

const fmt = (n: number, isRevenue = false) =>
  isRevenue ? `AED ${n.toLocaleString('en-AE', { maximumFractionDigits: 0 })}` : String(Math.round(n))

function periodOptions() {
  const opts = []
  const now = new Date()
  for (let i = 0; i < 12; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    opts.push({
      val: `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}`,
      label: d.toLocaleString('en-AE', { month: 'long', year: 'numeric' }),
    })
  }
  return opts
}

function TrendArrow({ pct }: { pct: number }) {
  if (Math.abs(pct) < 1) return <span className="text-gray-400 text-xs">—</span>
  const up = pct > 0
  return (
    <span className={`text-xs font-semibold ${up ? 'text-green-600' : 'text-red-500'}`}>
      {up ? '↑' : '↓'} {Math.abs(pct).toFixed(1)}%
    </span>
  )
}

export default function AgentPerformancePage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useLang()
  const { user } = useAuthStore()
  const router = useRouter()
  const isAdmin = user?.role === 'admin'
  const isOwnProfile = user?.id === id

  const periods = periodOptions()
  const [period, setPeriod] = useState(periods[0].val)

  const [kpis, setKpis] = useState<KPIs | null>(null)
  const [trends, setTrends] = useState<Trend[]>([])
  const [targets, setTargets] = useState<Target[]>([])
  const [loading, setLoading] = useState(true)

  // Target editor (admin only)
  const [editTarget, setEditTarget] = useState<{ metric: string; value: string } | null>(null)
  const [savingTarget, setSavingTarget] = useState(false)

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const [kpiRes, trendRes, targetRes] = await Promise.all([
        api.performance.agentKPIs(id, period),
        api.performance.agentTrends(id),
        api.performance.agentTargets(id, period),
      ])
      setKpis(kpiRes as KPIs)
      setTrends(((trendRes as any).data ?? []) as Trend[])
      setTargets(((targetRes as any).data ?? []) as Target[])
    } catch {}
    finally { setLoading(false) }
  }, [id, period])

  useEffect(() => { load() }, [load])

  // Only admins can view other agents' profiles; agents see only their own
  useEffect(() => {
    if (!isAdmin && !isOwnProfile) router.push('/performance')
  }, [isAdmin, isOwnProfile, router])

  const handleSaveTarget = async () => {
    if (!editTarget || !id) return
    setSavingTarget(true)
    try {
      await api.performance.upsertTarget({
        agent_id: id,
        metric: editTarget.metric,
        target_value: parseFloat(editTarget.value) || 0,
        period,
      })
      setEditTarget(null)
      load()
    } catch {}
    finally { setSavingTarget(false) }
  }

  if (loading || !kpis) {
    return (
      <div className="flex flex-col flex-1 min-h-0">
        <Header title={t('الأداء', 'Performance')} />
        <div className="flex-1 flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-brand-600 border-t-transparent rounded-full animate-spin" />
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={kpis.agent_name || t('الأداء', 'Performance')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-6">

          {/* Back + period */}
          <div className="flex items-center gap-3 flex-wrap">
            <Link href="/performance" className="text-xs text-gray-500 hover:text-gray-700 flex items-center gap-1">
              ← {t('لوحة المتصدرين', 'Leaderboard')}
            </Link>
            <select value={period} onChange={e => setPeriod(e.target.value)}
              className="ml-auto text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500">
              {periods.map(p => <option key={p.val} value={p.val}>{p.label}</option>)}
            </select>
          </div>

          {/* Agent header */}
          <div className="bg-white rounded-xl border border-gray-200 p-5 flex items-center gap-4">
            <div className="w-14 h-14 rounded-full bg-brand-100 text-brand-700 font-bold text-2xl flex items-center justify-center shrink-0">
              {kpis.agent_name?.[0]?.toUpperCase()}
            </div>
            <div className="flex-1 min-w-0">
              <h2 className="text-lg font-semibold text-gray-900">{kpis.agent_name}</h2>
              <div className="flex flex-wrap gap-1.5 mt-1">
                {kpis.badges.length === 0
                  ? <span className="text-xs text-gray-400">No badges this period</span>
                  : kpis.badges.map(b => (
                    <span key={b.key} className="inline-flex items-center gap-1 px-2 py-0.5 bg-amber-50 border border-amber-200 text-amber-700 text-xs font-medium rounded-full">
                      {b.icon} {b.label}
                    </span>
                  ))
                }
              </div>
            </div>
            {kpis.avg_response_hours > 0 && (
              <div className="text-right shrink-0">
                <p className="text-[10px] text-gray-400 uppercase tracking-wide">Avg Response</p>
                <p className="text-sm font-bold text-gray-800">
                  {kpis.avg_response_hours < 1
                    ? `${Math.round(kpis.avg_response_hours * 60)}min`
                    : `${kpis.avg_response_hours.toFixed(1)}h`}
                </p>
              </div>
            )}
          </div>

          {/* KPI cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[
              { label: 'Revenue', val: `AED ${kpis.revenue.toLocaleString('en-AE', {maximumFractionDigits:0})}`, sub: `${kpis.deals_won} deals won`, color: 'text-green-600' },
              { label: 'Listings Added', val: String(kpis.listings_added), sub: 'this period', color: 'text-brand-600' },
              { label: 'Leads Converted', val: String(kpis.leads_converted), sub: `of ${kpis.leads_assigned} assigned`, color: 'text-purple-600' },
              { label: 'Conversion Rate', val: kpis.leads_assigned > 0 ? `${Math.round((kpis.leads_converted/kpis.leads_assigned)*100)}%` : '—', sub: 'leads won/assigned', color: 'text-amber-600' },
            ].map(card => (
              <div key={card.label} className="bg-white rounded-xl border border-gray-200 p-4">
                <p className="text-[10px] text-gray-400 uppercase tracking-wide mb-1">{card.label}</p>
                <p className={`text-xl font-bold ${card.color}`}>{card.val}</p>
                <p className="text-xs text-gray-400 mt-0.5">{card.sub}</p>
              </div>
            ))}
          </div>

          {/* Trends */}
          {trends.length > 0 && (
            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('المقارنة الزمنية', 'Period Comparison')}</h3>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-[10px] text-gray-400 uppercase tracking-wide">
                      <th className="text-left pb-2 font-semibold">Metric</th>
                      <th className="text-right pb-2 font-semibold">This Month</th>
                      <th className="text-right pb-2 font-semibold">vs Last Month</th>
                      <th className="text-right pb-2 font-semibold">vs Last Year</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {trends.map(tr => (
                      <tr key={tr.metric}>
                        <td className="py-2.5 text-gray-700 font-medium">{METRIC_LABELS[tr.metric] ?? tr.metric}</td>
                        <td className="py-2.5 text-right font-semibold text-gray-800">
                          {tr.metric === 'revenue' ? `AED ${Math.round(tr.current).toLocaleString()}` : Math.round(tr.current)}
                        </td>
                        <td className="py-2.5 text-right"><TrendArrow pct={tr.change_month_pct} /></td>
                        <td className="py-2.5 text-right"><TrendArrow pct={tr.change_year_pct} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Targets */}
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-700">{t('الأهداف', 'Targets')} — {period}</h3>
              {isAdmin && (
                <button onClick={() => setEditTarget({ metric: 'revenue', value: '' })}
                  className="px-3 py-1.5 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700">
                  {t('+ هدف', '+ Set Target')}
                </button>
              )}
            </div>

            {/* Target editor */}
            {editTarget && (
              <div className="mb-4 p-3 bg-gray-50 rounded-lg flex items-center gap-2 flex-wrap">
                <select value={editTarget.metric} onChange={e => setEditTarget(p => p ? { ...p, metric: e.target.value } : null)}
                  className="text-sm border border-gray-200 rounded-lg px-2 py-1.5 bg-white">
                  {Object.entries(METRIC_LABELS).map(([k,v]) => <option key={k} value={k}>{v}</option>)}
                </select>
                <input type="number" min={0} placeholder="Target value" value={editTarget.value}
                  onChange={e => setEditTarget(p => p ? { ...p, value: e.target.value } : null)}
                  className="text-sm border border-gray-200 rounded-lg px-2 py-1.5 w-32" />
                <button onClick={handleSaveTarget} disabled={savingTarget}
                  className="px-3 py-1.5 bg-brand-600 text-white text-xs rounded-lg disabled:opacity-50">
                  {savingTarget ? '...' : 'Save'}
                </button>
                <button onClick={() => setEditTarget(null)} className="text-xs text-gray-400 hover:text-gray-600">Cancel</button>
              </div>
            )}

            {targets.length === 0 ? (
              <p className="text-xs text-gray-400 py-2">{t('لا توجد أهداف محددة لهذه الفترة', 'No targets set for this period')}{isAdmin ? ' — click "+ Set Target" above.' : ''}</p>
            ) : (
              <div className="space-y-3">
                {targets.map(target => (
                  <div key={target.id}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs font-medium text-gray-700">{METRIC_LABELS[target.metric] ?? target.metric}</span>
                      <span className="text-xs text-gray-500">
                        {target.metric === 'revenue'
                          ? `AED ${Math.round(target.current_value).toLocaleString()} / AED ${Math.round(target.target_value).toLocaleString()}`
                          : `${Math.round(target.current_value)} / ${Math.round(target.target_value)}`}
                        <span className={`ml-2 font-semibold ${target.progress_pct >= 100 ? 'text-green-600' : target.progress_pct >= 70 ? 'text-amber-600' : 'text-red-500'}`}>
                          {Math.round(target.progress_pct)}%
                        </span>
                      </span>
                    </div>
                    <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          target.progress_pct >= 100 ? 'bg-green-500' : target.progress_pct >= 70 ? 'bg-amber-500' : 'bg-brand-500'
                        }`}
                        style={{ width: `${Math.min(100, target.progress_pct)}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

        </div>
      </main>
    </div>
  )
}
