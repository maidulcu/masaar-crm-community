'use client'
import { useCallback, useEffect, useState } from 'react'
import { api, type LeaseNotice, type NoticeDraft, type NoticeKind } from '@/lib/api'
import { formatDate } from '@/lib/date'
import { formatMoney } from '@/lib/money'
import { useLang } from '@/context/LangContext'

const KINDS: Record<NoticeKind, [string, string]> = {
  rent_change: ['تعديل الإيجار', 'Rent change'],
  renewal_terms: ['شروط التجديد', 'Renewal terms'],
  non_renewal: ['عدم التجديد', 'Non-renewal'],
  other: ['أخرى', 'Other'],
}
const VIA: Record<string, [string, string]> = {
  email: ['البريد الإلكتروني', 'Email'], whatsapp: ['واتساب', 'WhatsApp'], hand: ['تسليم باليد', 'By hand'],
  registered_mail: ['بريد مسجل', 'Registered mail'], notary: ['كاتب العدل', 'Notary public'], other: ['أخرى', 'Other'],
}
const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'

export default function LeaseNoticesPanel({ leaseId, canEdit }: { leaseId: string; canEdit: boolean }) {
  const { t } = useLang()
  const [notices, setNotices] = useState<LeaseNotice[]>([])
  const [kind, setKind] = useState<NoticeKind>('rent_change')
  const [newRent, setNewRent] = useState('')
  const [draft, setDraft] = useState<NoticeDraft | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => { api.leaseOps.notices(leaseId).then(setNotices).catch(() => {}) }, [leaseId])
  useEffect(() => { load() }, [load])

  const wrap = async (fn: () => Promise<void>) => {
    setBusy(true); setError('')
    try { await fn() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') } finally { setBusy(false) }
  }
  const makeDraft = () => wrap(async () => {
    setDraft(await api.leaseOps.draftNotice(leaseId, { kind, new_annual_rent: newRent ? Number(newRent) : undefined }))
  })
  const save = () => wrap(async () => {
    if (!draft) return
    const n = await api.leaseOps.createNotice(leaseId, { kind, new_annual_rent: newRent ? Number(newRent) : undefined, body_en: draft.body_en, body_ar: draft.body_ar })
    setDraft(null); setNewRent(''); load()
    window.open(`/lease-notices/${n.id}/print`, '_blank', 'noopener')
  })
  const [serving, setServing] = useState<{ id: string; via: string; date: string } | null>(null)
  const markSent = () => wrap(async () => {
    if (!serving) return
    await api.leaseOps.markSent(serving.id, { via: serving.via, sent_at: serving.date })
    setServing(null); load()
  })

  return (
    <div className="bg-white rounded-xl border border-gray-100 p-5 space-y-4">
      <div>
        <h2 className="font-semibold text-gray-900 text-sm">{t('الإشعارات للمستأجر', 'Notices to the tenant')}</h2>
        <p className="text-[11px] text-gray-500 mt-1">
          {t('المسودات نقطة بداية وليست استشارة قانونية. تختلف مدد الإشعار وشروط زيادة الإيجار أو إنهاء العقد حسب الإمارة؛ تحقق من القواعد السارية قبل الإرسال.',
            'Drafts are a starting point, not legal advice. Notice periods and the conditions for raising rent or ending a tenancy depend on the emirate; check the current rules before serving a notice.')}
        </p>
      </div>
      {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}

      {canEdit && (
        <div className="flex flex-wrap items-end gap-2">
          <label className="text-xs text-gray-600">{t('النوع', 'Kind')}
            <select className={`${input} block mt-1`} value={kind} onChange={(e) => { setKind(e.target.value as NoticeKind); setDraft(null) }}>
              {(Object.keys(KINDS) as NoticeKind[]).map((k) => <option key={k} value={k}>{t(...KINDS[k])}</option>)}
            </select>
          </label>
          {(kind === 'rent_change' || kind === 'renewal_terms') && (
            <label className="text-xs text-gray-600">{t('الإيجار السنوي الجديد', 'New annual rent')}
              <input type="number" min={0} className={`${input} block mt-1 w-32`} value={newRent} onChange={(e) => setNewRent(e.target.value)} />
            </label>
          )}
          <button type="button" disabled={busy} onClick={makeDraft} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 disabled:opacity-50">{t('إنشاء مسودة', 'Draft')}</button>
        </div>
      )}

      {draft && (
        <div className="space-y-2">
          <p className="text-xs text-gray-600">
            {t('ينتهي العقد', 'Lease ends')} {formatDate(draft.lease_end)} · {t('آخر موعد للإشعار', 'Notice due by')}{' '}
            <strong>{formatDate(draft.notice_deadline)}</strong> ({draft.notice_days} {t('يوماً', 'days')}) ·{' '}
            {t('الإيجار الحالي', 'Current rent')} {formatMoney(draft.current_annual_rent, draft.currency)}
          </p>
          <div className="grid md:grid-cols-2 gap-3">
            <textarea dir="ltr" rows={12} className="w-full p-2 border border-gray-300 rounded-lg text-sm font-mono" value={draft.body_en}
              onChange={(e) => setDraft((d) => d && { ...d, body_en: e.target.value })} aria-label="English" />
            <textarea dir="rtl" rows={12} className="w-full p-2 border border-gray-300 rounded-lg text-sm" value={draft.body_ar}
              onChange={(e) => setDraft((d) => d && { ...d, body_ar: e.target.value })} aria-label="العربية" />
          </div>
          <div className="flex gap-2">
            <button type="button" disabled={busy} onClick={save} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">{t('حفظ وطباعة', 'Save and print')}</button>
            <button type="button" onClick={() => setDraft(null)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
          </div>
        </div>
      )}

      {notices.length > 0 && (
        <ul className="divide-y divide-gray-100 text-sm">
          {notices.map((n) => (
            <li key={n.id} className="py-2 flex flex-wrap items-center gap-2">
              <span className="font-medium text-gray-900">{t(...KINDS[n.kind])}</span>
              {n.new_rent != null && <span className="text-xs text-gray-500">→ {formatMoney(n.new_rent)}</span>}
              <span className="text-xs text-gray-500">{formatDate(n.created_at)}</span>
              <span className="flex-1" />
              {n.sent_at
                ? <span className="text-xs text-green-700">{t('أُرسل', 'Served')} {formatDate(n.sent_at)} · {VIA[n.sent_via] ? t(...VIA[n.sent_via]) : n.sent_via}</span>
                : canEdit && (serving?.id === n.id ? (
                  <span className="flex items-center gap-1.5">
                    <select className={`${input} text-xs py-1`} value={serving.via} onChange={(e) => setServing({ ...serving, via: e.target.value })}>
                      {Object.keys(VIA).map((v) => <option key={v} value={v}>{t(...VIA[v])}</option>)}
                    </select>
                    <input type="date" className={`${input} text-xs py-1`} value={serving.date} max={new Date().toISOString().slice(0, 10)}
                      onChange={(e) => setServing({ ...serving, date: e.target.value })} />
                    <button type="button" disabled={busy} onClick={markSent} className="text-xs px-2 py-1 rounded bg-primary-600 text-white disabled:opacity-50">{t('حفظ', 'Save')}</button>
                  </span>
                ) : (
                  <button type="button" onClick={() => setServing({ id: n.id, via: 'hand', date: new Date().toISOString().slice(0, 10) })}
                    className="text-xs text-primary-600 hover:underline">{t('تسجيل الإرسال', 'Record as served')}</button>
                ))}
              <a href={`/lease-notices/${n.id}/print`} target="_blank" rel="noopener" className="text-xs text-gray-600 hover:underline">{t('طباعة', 'Print')}</a>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
