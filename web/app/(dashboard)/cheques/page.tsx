'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api, type Cheque, type ChequeStatus } from '@/lib/api'
import { CHEQUE_STATUS, chequeStatusBadge } from '@/components/lease/ChequePanel'
import { formatDate } from '@/lib/date'
import { formatMoney } from '@/lib/money'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const day = (d: Date) => d.toISOString().slice(0, 10)

// Cheque register: by default, the cheques in hand that are due to be banked.
export default function ChequesPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const canEdit = user?.role === 'admin' || user?.role === 'agent'
  const [status, setStatus] = useState<ChequeStatus | ''>('received')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState(() => day(new Date()))
  const [cheques, setCheques] = useState<Cheque[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [error, setError] = useState('')

  const load = useCallback(() => {
    setLoading(true); setError('')
    api.cheques.list({ status: status || undefined, from: from || undefined, to: to || undefined })
      .then((c) => { setCheques(c); setSelected(new Set()) })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
      .finally(() => setLoading(false))
  }, [status, from, to])
  useEffect(() => { load() }, [load])

  const preset = (s: ChequeStatus | '', f: string, tt: string) => { setStatus(s); setFrom(f); setTo(tt) }
  const now = new Date()
  const in30 = day(new Date(now.getTime() + 30 * 86400_000))

  const depositSelected = async () => {
    setError('')
    for (const id of Array.from(selected)) {
      try { await api.cheques.setStatus(id, { status: 'deposited', date: day(new Date()) }) } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed'); break
      }
    }
    load()
  }

  const totals = cheques.reduce<Record<string, number>>((acc, c) => { acc[c.currency] = (acc[c.currency] ?? 0) + c.amount; return acc }, {})

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('سجل الشيكات', 'Cheque register')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-5xl space-y-4">
          <div className="flex flex-wrap gap-2 print:hidden">
            <button type="button" onClick={() => preset('received', '', day(now))} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('للإيداع اليوم', 'To deposit today')}</button>
            <button type="button" onClick={() => preset('received', '', in30)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('خلال 30 يوماً', 'Next 30 days')}</button>
            <button type="button" onClick={() => preset('deposited', '', '')} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('بانتظار التحصيل', 'Awaiting clearance')}</button>
            <button type="button" onClick={() => preset('bounced', '', '')} className="px-3 py-1.5 text-xs rounded-lg border border-red-200 text-red-600 bg-white">{t('مرتجعة', 'Bounced')}</button>
            <button type="button" onClick={() => window.print()} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('طباعة', 'Print')}</button>
          </div>
          <div className="flex flex-wrap items-end gap-3 print:hidden">
            <label className="text-xs text-gray-600">{t('الحالة', 'Status')}
              <select className="block mt-1 px-2 py-1.5 border border-gray-300 rounded text-sm" value={status} onChange={(e) => setStatus(e.target.value as ChequeStatus | '')}>
                <option value="">{t('الكل', 'All')}</option>
                {(Object.keys(CHEQUE_STATUS) as ChequeStatus[]).map((s) => <option key={s} value={s}>{t(CHEQUE_STATUS[s].ar, CHEQUE_STATUS[s].en)}</option>)}
              </select>
            </label>
            <label className="text-xs text-gray-600">{t('من', 'From')}<input type="date" value={from} onChange={(e) => setFrom(e.target.value)} className="block mt-1 px-2 py-1.5 border border-gray-300 rounded text-sm" /></label>
            <label className="text-xs text-gray-600">{t('إلى', 'To')}<input type="date" value={to} onChange={(e) => setTo(e.target.value)} className="block mt-1 px-2 py-1.5 border border-gray-300 rounded text-sm" /></label>
            {canEdit && selected.size > 0 && (
              <button type="button" onClick={depositSelected} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">
                {t('تسجيل الإيداع', 'Mark deposited')} ({selected.size})
              </button>
            )}
          </div>
          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
          <div className="bg-white rounded-xl border border-gray-200 overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs text-gray-500">
                <tr>
                  {canEdit && <th scope="col" className="px-3 py-3 print:hidden"><span className="sr-only">{t('تحديد', 'Select')}</span></th>}
                  <th scope="col" className="px-4 py-3 text-start">{t('التاريخ', 'Date')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('رقم الشيك', 'Cheque')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('المستأجر / العقار', 'Tenant / property')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('المبلغ', 'Amount')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('الحالة', 'Status')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {loading && <tr><td colSpan={6} className="px-4 py-6 text-center text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</td></tr>}
                {!loading && cheques.length === 0 && <tr><td colSpan={6} className="px-4 py-6 text-center text-gray-400">{t('لا توجد شيكات', 'No cheques')}</td></tr>}
                {!loading && cheques.map((c) => (
                  <tr key={c.id}>
                    {canEdit && (
                      <td className="px-3 py-2.5 print:hidden">
                        {c.status === 'received' && (
                          <input type="checkbox" aria-label={c.cheque_number} checked={selected.has(c.id)}
                            onChange={(e) => setSelected((s) => { const n = new Set(s); if (e.target.checked) { n.add(c.id) } else { n.delete(c.id) } return n })} />
                        )}
                      </td>
                    )}
                    <td className="px-4 py-2.5 whitespace-nowrap">{formatDate(c.cheque_date)}</td>
                    <td className="px-4 py-2.5"><span className="font-mono">{c.cheque_number}</span>{c.bank_name && <span className="text-xs text-gray-500"> · {c.bank_name}</span>}</td>
                    <td className="px-4 py-2.5">
                      <Link href={`/leases/${c.lease_id}`} className="text-primary-600 hover:underline">{c.tenant_name || '—'}</Link>
                      <span className="text-xs text-gray-500"> · {c.property_name}</span>
                    </td>
                    <td className="px-4 py-2.5 text-end tabular-nums whitespace-nowrap">{formatMoney(c.amount, c.currency)}</td>
                    <td className="px-4 py-2.5">{chequeStatusBadge(c.status, t)}</td>
                  </tr>
                ))}
              </tbody>
              {cheques.length > 0 && (
                <tfoot className="bg-gray-50 text-sm font-medium">
                  <tr>
                    <td colSpan={canEdit ? 4 : 3} className="px-4 py-2.5 text-gray-600">{cheques.length} {t('شيك', 'cheques')}</td>
                    <td className="px-4 py-2.5 text-end tabular-nums">{Object.entries(totals).map(([cur, v]) => formatMoney(v, cur)).join(' · ')}</td>
                    <td />
                  </tr>
                </tfoot>
              )}
            </table>
          </div>
        </div>
      </main>
    </div>
  )
}
