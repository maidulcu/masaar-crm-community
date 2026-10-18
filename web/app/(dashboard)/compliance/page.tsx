'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type AMLCase } from '@/lib/api'
import { formatDate } from '@/lib/date'
import { formatMoney } from '@/lib/money'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { paymentLabel, reasonLabel, screeningLabel } from '@/components/kyc/labels'

const day = (d: Date) => d.toISOString().slice(0, 10)
const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'

// AML review queue (admins and compliance officers). Deals that meet the review criteria
// are listed here; a person decides whether to report them through goAML.
export default function CompliancePage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const allowed = user?.role === 'admin' || user?.role === 'compliance'
  const [status, setStatus] = useState<'flagged' | 'no_report' | 'reported' | 'all'>('flagged')
  const [cases, setCases] = useState<AMLCase[]>([])
  const [review, setReview] = useState<{ id: string; decision: 'no_report' | 'reported'; notes: string; ref: string } | null>(null)
  const [settings, setSettings] = useState<{ cash_threshold: number; flag_virtual_asset: boolean } | null>(null)
  const [range, setRange] = useState(() => ({ from: day(new Date(Date.now() - 365 * 86400_000)), to: day(new Date(Date.now() + 86400_000)) }))
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')

  const load = useCallback(() => {
    api.kyc.amlQueue(status).then(setCases).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [status])
  useEffect(() => {
    if (user && !allowed) { router.push('/dashboard'); return }
    if (!allowed) return
    load()
    api.kyc.settings().then(setSettings).catch(() => {})
  }, [user, allowed, router, load])

  if (!allowed) return null

  const run = async (fn: () => Promise<void>, ok?: string) => {
    setError(''); setMsg('')
    try { await fn(); if (ok) setMsg(ok) } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('الامتثال — مكافحة غسل الأموال', 'Compliance — AML review')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-5xl space-y-4">
          <p className="text-xs text-gray-600">
            {t('تُدرج هنا الصفقات التي تستوفي معايير المراجعة. القرار بالإبلاغ عبر منصة goAML يعود للمسؤول عن الامتثال؛ النظام لا يتخذ قرارات امتثال. تحقق من المعايير الحالية لدى وزارة الاقتصاد ووحدة المعلومات المالية.',
              'Deals that meet the review criteria are listed here. Whether to report through goAML is the compliance officer’s decision; the system makes no compliance decisions. Check the current criteria with the Ministry of Economy and the UAE FIU.')}
          </p>
          <div className="flex flex-wrap gap-2">
            {(['flagged', 'no_report', 'reported', 'all'] as const).map((s) => (
              <button key={s} type="button" onClick={() => setStatus(s)}
                className={`px-3 py-1.5 text-xs rounded-lg border ${status === s ? 'bg-gray-900 text-white border-gray-900' : 'bg-white border-gray-300'}`}>
                {{ flagged: t('بانتظار المراجعة', 'To review'), no_report: t('لا يلزم إبلاغ', 'No report'), reported: t('تم الإبلاغ', 'Reported'), all: t('الكل', 'All') }[s]}
              </button>
            ))}
          </div>
          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
          {msg && <p className="text-xs text-green-700">{msg}</p>}

          <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
            {cases.length === 0 && <p className="p-6 text-center text-sm text-gray-400">{t('لا توجد حالات', 'No cases')}</p>}
            {cases.map((c) => (
              <div key={c.deal_id} className="p-4 space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <Link href={`/deals/${c.deal_id}`} className="font-medium text-primary-600 hover:underline">{c.title}</Link>
                  <span className="text-sm tabular-nums">{formatMoney(c.amount, c.currency)}</span>
                  <span className="text-xs text-gray-500">· {paymentLabel(c.payment_method, t)}{c.cash_amount > 0 && ` (${t('نقداً', 'cash')} ${formatMoney(c.cash_amount, c.currency)})`}</span>
                  {c.reasons.map((r) => <span key={r} className="text-[11px] px-2 py-0.5 rounded-full bg-amber-50 text-amber-700">{reasonLabel(r, t)}</span>)}
                  <span className="flex-1" />
                  {c.flagged_at && <span className="text-xs text-gray-500">{formatDate(c.flagged_at)}</span>}
                </div>
                <div className="text-xs text-gray-600 flex flex-wrap gap-x-3">
                  <Link href={`/contacts/${c.contact_id}`} className="text-primary-600 hover:underline">{c.contact_name}</Link>
                  {c.contact_risk && <span>{t('المخاطر', 'Risk')}: {c.contact_risk}</span>}
                  {c.contact_is_pep && <span className="text-red-600">PEP</span>}
                  <span>{screeningLabel(c.screening, t)}</span>
                  <span>{t('الوكيل', 'Agent')}: {c.owner_name || '—'}</span>
                  {c.status !== 'flagged' && <span>· {c.status === 'reported' ? `${t('أُبلغ', 'Reported')} ${c.goaml_reference}` : t('لا يلزم إبلاغ', 'No report')}{c.notes && ` — ${c.notes}`}</span>}
                </div>
                {review?.id === c.deal_id ? (
                  <form className="flex flex-wrap items-end gap-2 bg-gray-50 rounded-lg p-3" onSubmit={(e) => {
                    e.preventDefault()
                    run(async () => {
                      await api.kyc.review(c.deal_id, { decision: review.decision, notes: review.notes, goaml_reference: review.ref })
                      setReview(null); load()
                    }, t('تم حفظ المراجعة', 'Review saved'))
                  }}>
                    <select className={input} value={review.decision} onChange={(e) => setReview({ ...review, decision: e.target.value as 'no_report' | 'reported' })} aria-label={t('القرار', 'Decision')}>
                      <option value="no_report">{t('لا يلزم إبلاغ', 'No report needed')}</option>
                      <option value="reported">{t('تم الإبلاغ عبر goAML', 'Reported through goAML')}</option>
                    </select>
                    {review.decision === 'reported' && (
                      <input className={`${input} w-40`} placeholder={t('مرجع goAML', 'goAML reference')} value={review.ref} onChange={(e) => setReview({ ...review, ref: e.target.value })} required />
                    )}
                    <input className={`${input} flex-1 min-w-[12rem]`} placeholder={t('ملاحظات المراجعة', 'Review notes')} value={review.notes}
                      onChange={(e) => setReview({ ...review, notes: e.target.value })} required={review.decision === 'no_report'} />
                    <button type="submit" className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">{t('حفظ', 'Save')}</button>
                    <button type="button" onClick={() => setReview(null)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
                  </form>
                ) : (
                  <button type="button" onClick={() => setReview({ id: c.deal_id, decision: 'no_report', notes: c.notes, ref: c.goaml_reference })}
                    className="text-xs text-primary-600 hover:underline">{c.status === 'flagged' ? t('مراجعة', 'Review') : t('تعديل المراجعة', 'Change review')}</button>
                )}
              </div>
            ))}
          </div>

          <section className="bg-white rounded-xl border border-gray-200 p-4 space-y-2">
            <h2 className="text-sm font-semibold text-gray-900">{t('تصدير لإعداد بلاغات goAML', 'Export for goAML reports')}</h2>
            <p className="text-xs text-gray-500">
              {t('ملف CSV بالبيانات اللازمة عن المعاملة والعميل ووثيقة هويته والعقار والوكيل. يُستخدم لإعداد البلاغ في منصة goAML وليس إرسالاً مباشراً.',
                'A CSV with what a report needs about the transaction, the client and their ID document, the property and the agent. It is input for preparing the report in the goAML portal, not a submission.')}
            </p>
            <div className="flex flex-wrap items-end gap-2">
              <label className="text-xs text-gray-600">{t('من', 'From')}<input type="date" className={`${input} block`} value={range.from} onChange={(e) => setRange({ ...range, from: e.target.value })} /></label>
              <label className="text-xs text-gray-600">{t('إلى', 'To')}<input type="date" className={`${input} block`} value={range.to} onChange={(e) => setRange({ ...range, to: e.target.value })} /></label>
              <button type="button" onClick={() => run(() => api.kyc.exportCSV(range.from, range.to))} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('تنزيل CSV', 'Download CSV')}</button>
            </div>
          </section>

          {settings && (
            <section className="bg-white rounded-xl border border-gray-200 p-4 space-y-2">
              <h2 className="text-sm font-semibold text-gray-900">{t('معايير المراجعة', 'Review criteria')}</h2>
              <div className="flex flex-wrap items-end gap-3">
                <label className="text-xs text-gray-600">{t('حد النقد', 'Cash threshold')}
                  <input type="number" min={0} disabled={user?.role !== 'admin'} className={`${input} block w-36`} value={settings.cash_threshold}
                    onChange={(e) => setSettings({ ...settings, cash_threshold: Number(e.target.value) })} />
                </label>
                <label className="flex items-center gap-2 text-xs text-gray-700 pb-2">
                  <input type="checkbox" disabled={user?.role !== 'admin'} checked={settings.flag_virtual_asset} onChange={(e) => setSettings({ ...settings, flag_virtual_asset: e.target.checked })} />
                  {t('مراجعة الدفع بالأصول الافتراضية', 'Review payments in virtual assets')}
                </label>
                {user?.role === 'admin' && (
                  <button type="button" onClick={() => run(async () => { setSettings(await api.kyc.updateSettings(settings)) }, t('تم الحفظ', 'Saved'))}
                    className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">{t('حفظ', 'Save')}</button>
                )}
              </div>
            </section>
          )}
        </div>
      </main>
    </div>
  )
}
