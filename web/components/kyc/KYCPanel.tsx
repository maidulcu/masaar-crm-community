'use client'
import { useCallback, useEffect, useState } from 'react'
import { api, type KYCProfile } from '@/lib/api'
import { formatDate } from '@/lib/date'
import { useLang } from '@/context/LangContext'
import { DOC_TYPE_KEYS, SCREENING_KEYS, docTypeLabel, missingLabel, screeningLabel } from './labels'

const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'

// A contact's KYC: identity documents (agents add them), and the risk rating, PEP flag and
// screening result (admins and compliance officers).
export default function KYCPanel({ contactId, canAdd, canReview }: { contactId: string; canAdd: boolean; canReview: boolean }) {
  const { t } = useLang()
  const [p, setP] = useState<KYCProfile | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [adding, setAdding] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<Partial<KYCProfile>>({})

  const load = useCallback(() => {
    api.kyc.profile(contactId).then((v) => { setP(v); setDraft(v) }).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [contactId])
  useEffect(() => { load() }, [load])

  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true); setError('')
    try { await fn(); load(); return true } catch (e) { setError(e instanceof Error ? e.message : 'Failed'); return false } finally { setBusy(false) }
  }

  if (!p) return error ? <p className="text-xs text-red-600">{error}</p> : null
  const today = new Date().toISOString().slice(0, 10)

  return (
    <div className="bg-white rounded-2xl border border-surface-200/70 p-5 space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-sm font-semibold text-surface-900">{t('اعرف عميلك (KYC)', 'Know your customer (KYC)')}</h3>
        {p.missing.length === 0
          ? <span className="text-[11px] px-2 py-0.5 rounded-full bg-green-50 text-green-700">{t('مكتمل', 'Complete')}</span>
          : p.missing.map((m) => <span key={m} className="text-[11px] px-2 py-0.5 rounded-full bg-amber-50 text-amber-700">{missingLabel(m, t)}</span>)}
        {p.is_pep && <span className="text-[11px] px-2 py-0.5 rounded-full bg-red-50 text-red-700">PEP</span>}
        {p.risk && <span className="text-[11px] px-2 py-0.5 rounded-full bg-gray-100 text-gray-700">{t('المخاطر', 'Risk')}: {p.risk}</span>}
        <span className="text-[11px] px-2 py-0.5 rounded-full bg-gray-100 text-gray-700">{screeningLabel(p.screening, t)}</span>
      </div>
      {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}

      <ul className="divide-y divide-gray-100 text-sm">
        {p.documents.map((d) => (
          <li key={d.id} className="py-2 flex flex-wrap items-center gap-2">
            <span className="font-medium text-gray-900">{docTypeLabel(d.doc_type, t)}</span>
            <span dir="ltr" className="font-mono text-xs text-gray-600">{d.number}</span>
            {d.issuing_country && <span className="text-xs text-gray-500">{d.issuing_country}</span>}
            {d.expiry_date && (
              <span className={`text-xs ${d.expiry_date.slice(0, 10) < today ? 'text-red-600' : 'text-gray-500'}`}>
                {t('ينتهي', 'expires')} {formatDate(d.expiry_date)}
              </span>
            )}
            <span className="flex-1" />
            {d.has_file && (
              <button type="button" onClick={() => run(() => api.kyc.downloadDocument(d.id, d.file_name || `${d.doc_type}-${d.number}`))} className="text-xs text-primary-600 hover:underline">
                {t('النسخة', 'Copy')}
              </button>
            )}
            {canReview && (
              <button type="button" onClick={() => { if (confirm(t('حذف هذه الوثيقة؟', 'Delete this document?'))) run(() => api.kyc.deleteDocument(d.id)) }}
                className="text-xs text-red-600 hover:underline">{t('حذف', 'Delete')}</button>
            )}
          </li>
        ))}
        {p.documents.length === 0 && <li className="py-2 text-xs text-gray-400">{t('لا توجد وثائق', 'No documents')}</li>}
      </ul>

      {canAdd && !adding && <button type="button" onClick={() => setAdding(true)} className="text-xs text-primary-600 hover:underline">{t('إضافة وثيقة هوية', 'Add identity document')}</button>}
      {canAdd && adding && (
        <form className="flex flex-wrap items-end gap-2 bg-gray-50 border border-gray-200 rounded-lg p-3"
          onSubmit={async (e) => { e.preventDefault(); const ok = await run(() => api.kyc.addDocument(contactId, new FormData(e.currentTarget))); if (ok) setAdding(false) }}>
          <label className="text-xs text-gray-600">{t('النوع', 'Type')}
            <select name="doc_type" className={`${input} block`}>{DOC_TYPE_KEYS.map((k) => <option key={k} value={k}>{docTypeLabel(k, t)}</option>)}</select>
          </label>
          <label className="text-xs text-gray-600">{t('الرقم', 'Number')}<input name="number" required maxLength={64} dir="ltr" className={`${input} block w-44`} /></label>
          <label className="text-xs text-gray-600">{t('بلد الإصدار', 'Country')}<input name="issuing_country" maxLength={2} placeholder="AE" dir="ltr" className={`${input} block w-16 uppercase`} /></label>
          <label className="text-xs text-gray-600">{t('تاريخ الانتهاء', 'Expiry')}<input name="expiry_date" type="date" required className={`${input} block`} /></label>
          <label className="text-xs text-gray-600">{t('نسخة (JPEG/PNG/PDF)', 'Copy (JPEG/PNG/PDF)')}<input name="file" type="file" accept="image/jpeg,image/png,application/pdf" className="block text-xs mt-1" /></label>
          <button type="submit" disabled={busy} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">{t('حفظ', 'Save')}</button>
          <button type="button" onClick={() => setAdding(false)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
        </form>
      )}

      {canReview && !editing && <button type="button" onClick={() => setEditing(true)} className="block text-xs text-primary-600 hover:underline">{t('تقييم المخاطر والفحص', 'Risk rating and screening')}</button>}
      {canReview && editing && (
        <form className="grid sm:grid-cols-2 gap-3 bg-gray-50 border border-gray-200 rounded-lg p-3"
          onSubmit={async (e) => { e.preventDefault(); const ok = await run(() => api.kyc.update(contactId, draft)); if (ok) setEditing(false) }}>
          <label className="text-xs text-gray-600">{t('تاريخ الميلاد', 'Date of birth')}
            <input type="date" className={`${input} block w-full`} value={draft.date_of_birth?.slice(0, 10) ?? ''} onChange={(e) => setDraft({ ...draft, date_of_birth: e.target.value || null })} />
          </label>
          <label className="text-xs text-gray-600">{t('تصنيف المخاطر', 'Risk rating')}
            <select className={`${input} block w-full`} value={draft.risk ?? ''} onChange={(e) => setDraft({ ...draft, risk: e.target.value as KYCProfile['risk'] })}>
              <option value="">—</option><option value="low">{t('منخفض', 'Low')}</option><option value="medium">{t('متوسط', 'Medium')}</option><option value="high">{t('مرتفع', 'High')}</option>
            </select>
          </label>
          <label className="text-xs text-gray-600 sm:col-span-2">{t('السبب', 'Reason')}
            <input className={`${input} block w-full`} value={draft.risk_reason ?? ''} onChange={(e) => setDraft({ ...draft, risk_reason: e.target.value })} />
          </label>
          <label className="flex items-center gap-2 text-xs text-gray-700">
            <input type="checkbox" checked={!!draft.is_pep} onChange={(e) => setDraft({ ...draft, is_pep: e.target.checked })} />
            {t('شخص مكشوف سياسياً (PEP)', 'Politically exposed person (PEP)')}
          </label>
          <label className="text-xs text-gray-600">{t('نتيجة الفحص', 'Screening result')}
            <select className={`${input} block w-full`} value={draft.screening ?? 'not_screened'} onChange={(e) => setDraft({ ...draft, screening: e.target.value as KYCProfile['screening'] })}>
              {SCREENING_KEYS.map((k) => <option key={k} value={k}>{screeningLabel(k, t)}</option>)}
            </select>
          </label>
          <label className="text-xs text-gray-600 sm:col-span-2">{t('مصدر الفحص (القائمة، التاريخ، المرجع)', 'Screening source (list, date, reference)')}
            <input className={`${input} block w-full`} value={draft.screening_source ?? ''} onChange={(e) => setDraft({ ...draft, screening_source: e.target.value })} />
          </label>
          <label className="text-xs text-gray-600 sm:col-span-2">{t('ملاحظات', 'Notes')}
            <textarea rows={3} className={`${input} block w-full`} value={draft.notes ?? ''} onChange={(e) => setDraft({ ...draft, notes: e.target.value })} />
          </label>
          <div className="flex flex-wrap gap-2 sm:col-span-2">
            <button type="submit" disabled={busy} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">{t('حفظ', 'Save')}</button>
            <button type="button" disabled={busy} onClick={() => run(() => api.kyc.screen(contactId))} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('فحص عبر المزوّد', 'Screen with provider')}</button>
            <button type="button" onClick={() => { setEditing(false); setDraft(p) }} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
          </div>
          {p.reviewed_at && <p className="text-[11px] text-gray-500 sm:col-span-2">{t('آخر مراجعة', 'Last reviewed')} {formatDate(p.reviewed_at)}</p>}
        </form>
      )}
    </div>
  )
}
