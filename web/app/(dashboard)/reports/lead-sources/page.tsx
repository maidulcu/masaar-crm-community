'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type LeadSourceRow } from '@/lib/api'
import { leadSourceLabel } from '@/lib/leadSource'
import { companyCurrency, formatMoney } from '@/lib/money'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

function isoDay(d: Date) {
  return d.toISOString().slice(0, 10)
}

export default function LeadSourcesPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [from, setFrom] = useState(() => isoDay(new Date(Date.now() - 90 * 86400_000)))
  const [to, setTo] = useState(() => isoDay(new Date()))
  const [rows, setRows] = useState<LeadSourceRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    if (!from || !to) return
    setLoading(true); setError('')
    // The API's end date is exclusive; the form's is inclusive.
    const end = isoDay(new Date(new Date(to).getTime() + 86400_000))
    api.reports.leadSources(from, end)
      .then((r) => setRows(r.rows))
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false))
  }, [user, router, from, to])

  if (user?.role !== 'admin') return null

  const total = rows.reduce((n, r) => n + r.leads, 0)
  const currency = companyCurrency()
  const pct = (n: number, d: number) => (d ? `${Math.round((n / d) * 100)}%` : '—')

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('مصادر العملاء', 'Lead Sources')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-4">
          <div className="flex flex-wrap items-end gap-3">
            <label className="text-xs text-gray-600">{t('من', 'From')}
              <input type="date" value={from} max={to} onChange={(e) => setFrom(e.target.value)} className="block mt-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            </label>
            <label className="text-xs text-gray-600">{t('إلى', 'To')}
              <input type="date" value={to} min={from} onChange={(e) => setTo(e.target.value)} className="block mt-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            </label>
            <p className="text-xs text-gray-500 pb-2">
              {t('العملاء الذين أُنشئوا في هذه الفترة، ونتائجهم حتى اليوم.', 'Leads created in this period, and how they have converted so far.')}
            </p>
          </div>
          {error && <p role="alert" className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
          <div className="bg-white rounded-xl border border-gray-200 overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs text-gray-500">
                <tr>
                  <th scope="col" className="px-4 py-3 text-start">{t('المصدر', 'Source')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('العملاء', 'Leads')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('الحصة', 'Share')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('تم الفوز', 'Won')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('معدل التحويل', 'Conversion')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('الصفقات', 'Deals')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('إيرادات الصفقات المكسوبة', 'Won deal value')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {loading && <tr><td colSpan={7} className="px-4 py-6 text-center text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</td></tr>}
                {!loading && rows.length === 0 && <tr><td colSpan={7} className="px-4 py-6 text-center text-gray-400">{t('لا يوجد عملاء في هذه الفترة', 'No leads in this period')}</td></tr>}
                {!loading && rows.map((r) => (
                  <tr key={r.source}>
                    <td className="px-4 py-3 text-gray-900">{leadSourceLabel(r.source, t)}</td>
                    <td className="px-4 py-3 text-end tabular-nums">{r.leads}</td>
                    <td className="px-4 py-3 text-end tabular-nums text-gray-500">{pct(r.leads, total)}</td>
                    <td className="px-4 py-3 text-end tabular-nums">{r.won_leads}</td>
                    <td className="px-4 py-3 text-end tabular-nums">{pct(r.won_leads, r.leads)}</td>
                    <td className="px-4 py-3 text-end tabular-nums">{r.deals}</td>
                    <td className="px-4 py-3 text-end tabular-nums">{formatMoney(r.won_revenue, currency)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  )
}
