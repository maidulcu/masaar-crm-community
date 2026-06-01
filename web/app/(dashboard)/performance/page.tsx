'use client'
import { useState, useEffect, useCallback } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

interface AgentKPIs {
  agent_id: string
  agent_name: string
  rank: number
  deals_won: number
  revenue: number
  listings_added: number
  leads_assigned: number
  leads_converted: number
  avg_response_hours: number
  badges: { key: string; label: string; icon: string }[]
}

const METRICS = [
  { value: 'revenue',         label: 'Revenue' },
  { value: 'deals_won',       label: 'Deals Won' },
  { value: 'listings_added',  label: 'Listings' },
  { value: 'leads_converted', label: 'Leads Converted' },
]

const fmt = (n: number) => n.toLocaleString('en-AE', { maximumFractionDigits: 0 })
const fmtK = (n: number) => n >= 1_000_000 ? `${(n/1_000_000).toFixed(1)}M` : n >= 1_000 ? `${(n/1_000).toFixed(0)}K` : fmt(n)

function periodOptions() {
  const opts = []
  const now = new Date()
  for (let i = 0; i < 12; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    const val = `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}`
    const label = d.toLocaleString('en-AE', { month: 'long', year: 'numeric' })
    opts.push({ val, label })
  }
  return opts
}

export default function LeaderboardPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const periods = periodOptions()
  const [period, setPeriod] = useState(periods[0].val)
  const [metric, setMetric] = useState('revenue')
  const [board, setBoard] = useState<AgentKPIs[]>([])
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const r = await api.performance.leaderboard(period, metric) as any
      setBoard(r.data ?? [])
    } catch { setBoard([]) }
    finally { setLoading(false) }
  }, [period, metric])

  useEffect(() => { load() }, [load])

  const topAgent = board[0]

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('لوحة الأداء', 'Agent Leaderboard')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-5">

          {/* Controls */}
          <div className="flex items-center gap-3 flex-wrap">
            <select value={period} onChange={e => setPeriod(e.target.value)}
              className="text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500">
              {periods.map(p => <option key={p.val} value={p.val}>{p.label}</option>)}
            </select>
            <div className="flex items-center gap-1 bg-white border border-gray-200 rounded-lg p-1">
              {METRICS.map(m => (
                <button key={m.value} onClick={() => setMetric(m.value)}
                  className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                    metric === m.value ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-50'
                  }`}>
                  {m.label}
                </button>
              ))}
            </div>
            {!isAdmin && (
              <Link href={`/performance/${user?.id}`}
                className="ml-auto px-4 py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700">
                {t('أدائي', 'My Performance')} →
              </Link>
            )}
          </div>

          {/* Podium — top 3 */}
          {!loading && board.length >= 3 && (
            <div className="grid grid-cols-3 gap-3">
              {/* 2nd */}
              <div className="bg-white rounded-xl border border-gray-200 p-4 text-center mt-6">
                <div className="w-12 h-12 rounded-full bg-gray-100 text-gray-600 font-bold text-lg flex items-center justify-center mx-auto mb-2">
                  {board[1].agent_name?.[0]?.toUpperCase()}
                </div>
                <div className="text-xs font-bold text-gray-400 mb-0.5">🥈 2nd</div>
                <p className="text-sm font-semibold text-gray-800 truncate">{board[1].agent_name}</p>
                <p className="text-xs text-gray-500 mt-1">{metric === 'revenue' ? `AED ${fmtK(board[1].revenue)}` : metricDisplay(board[1], metric)}</p>
              </div>
              {/* 1st */}
              <div className="bg-gradient-to-b from-amber-50 to-white rounded-xl border-2 border-amber-300 p-4 text-center -mt-2 shadow-sm">
                <div className="text-2xl mb-1">👑</div>
                <div className="w-14 h-14 rounded-full bg-amber-100 text-amber-700 font-bold text-xl flex items-center justify-center mx-auto mb-2">
                  {board[0].agent_name?.[0]?.toUpperCase()}
                </div>
                <div className="text-xs font-bold text-amber-500 mb-0.5">🥇 1st</div>
                <p className="text-sm font-semibold text-gray-800 truncate">{board[0].agent_name}</p>
                <p className="text-xs font-bold text-amber-600 mt-1">{metric === 'revenue' ? `AED ${fmtK(board[0].revenue)}` : metricDisplay(board[0], metric)}</p>
                <div className="flex flex-wrap justify-center gap-1 mt-2">
                  {board[0].badges.slice(0,3).map(b => (
                    <span key={b.key} title={b.label} className="text-sm">{b.icon}</span>
                  ))}
                </div>
              </div>
              {/* 3rd */}
              <div className="bg-white rounded-xl border border-gray-200 p-4 text-center mt-6">
                <div className="w-12 h-12 rounded-full bg-orange-50 text-orange-600 font-bold text-lg flex items-center justify-center mx-auto mb-2">
                  {board[2].agent_name?.[0]?.toUpperCase()}
                </div>
                <div className="text-xs font-bold text-orange-400 mb-0.5">🥉 3rd</div>
                <p className="text-sm font-semibold text-gray-800 truncate">{board[2].agent_name}</p>
                <p className="text-xs text-gray-500 mt-1">{metric === 'revenue' ? `AED ${fmtK(board[2].revenue)}` : metricDisplay(board[2], metric)}</p>
              </div>
            </div>
          )}

          {/* Full rankings table */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            {loading ? (
              <div className="p-12 text-center text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : board.length === 0 ? (
              <div className="p-12 text-center text-sm text-gray-400">{t('لا توجد بيانات', 'No data for this period')}</div>
            ) : (
              <>
                {/* Header */}
                <div className="grid grid-cols-7 gap-2 px-4 py-2 bg-gray-50 border-b border-gray-100 text-[10px] font-semibold text-gray-400 uppercase tracking-wide">
                  <div>#</div>
                  <div className="col-span-2">Agent</div>
                  <div className="text-right">Revenue</div>
                  <div className="text-right">Deals Won</div>
                  <div className="text-right">Listings</div>
                  <div className="text-right">Converted</div>
                </div>
                {board.map((agent, i) => (
                  <Link href={`/performance/${agent.agent_id}`} key={agent.agent_id}
                    className={`grid grid-cols-7 gap-2 px-4 py-3 items-center hover:bg-gray-50 transition-colors border-b border-gray-100 last:border-0 ${
                      i === 0 ? 'bg-amber-50/40' : ''
                    }`}>
                    <div className="font-bold text-sm text-gray-400">{agent.rank}</div>
                    <div className="col-span-2 flex items-center gap-2 min-w-0">
                      <div className={`w-8 h-8 rounded-full font-semibold text-sm flex items-center justify-center shrink-0 ${
                        i === 0 ? 'bg-amber-100 text-amber-700' : 'bg-gray-100 text-gray-600'
                      }`}>
                        {agent.agent_name?.[0]?.toUpperCase()}
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-gray-800 truncate">{agent.agent_name}</p>
                        <div className="flex gap-0.5">
                          {agent.badges.map(b => <span key={b.key} title={b.label} className="text-xs">{b.icon}</span>)}
                        </div>
                      </div>
                    </div>
                    <div className="text-right text-sm font-semibold text-gray-700">AED {fmtK(agent.revenue)}</div>
                    <div className="text-right text-sm text-gray-600">{agent.deals_won}</div>
                    <div className="text-right text-sm text-gray-600">{agent.listings_added}</div>
                    <div className="text-right text-sm text-gray-600">{agent.leads_converted}</div>
                  </Link>
                ))}
              </>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}

function metricDisplay(a: AgentKPIs, metric: string): string {
  switch (metric) {
    case 'deals_won': return `${a.deals_won} deals`
    case 'listings_added': return `${a.listings_added} listings`
    case 'leads_converted': return `${a.leads_converted} leads`
    default: return `AED ${fmtK(a.revenue)}`
  }
}
