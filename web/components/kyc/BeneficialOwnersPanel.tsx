'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { api, type BeneficialOwner } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { missingLabel } from './labels'

// A corporate client's beneficial owners: people linked as contacts, with their KYC status.
export default function BeneficialOwnersPanel({ orgId, contacts, canEdit }: {
  orgId: string; contacts: { id: string; full_name: string }[]; canEdit: boolean
}) {
  const { t } = useLang()
  const [owners, setOwners] = useState<BeneficialOwner[]>([])
  const [form, setForm] = useState({ contact_id: '', pct: '', role: 'owner' })
  const [error, setError] = useState('')
  const load = useCallback(() => { api.kyc.owners(orgId).then(setOwners).catch(() => {}) }, [orgId])
  useEffect(() => { load() }, [load])
  const run = async (fn: () => Promise<void>) => {
    setError('')
    try { await fn(); load() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }
  const roleLabel = (r: string) => ({ owner: t('مالك', 'Owner'), controller: t('مسيطر', 'Controller'), signatory: t('مفوّض بالتوقيع', 'Signatory') }[r] ?? r)

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-3">
      <h3 className="text-sm font-semibold text-gray-900">{t('المستفيدون الحقيقيون', 'Beneficial owners')}</h3>
      <ul className="divide-y divide-gray-100 text-sm">
        {owners.map((o) => (
          <li key={o.contact_id} className="py-2 flex flex-wrap items-center gap-2">
            <Link href={`/contacts/${o.contact_id}`} className="text-primary-600 hover:underline">{o.full_name}</Link>
            <span className="text-xs text-gray-500">{roleLabel(o.role)}{o.ownership_pct != null && ` · ${o.ownership_pct}%`}</span>
            {o.kyc_missing.length === 0
              ? <span className="text-[11px] px-2 py-0.5 rounded-full bg-green-50 text-green-700">KYC ✓</span>
              : <span className="text-[11px] px-2 py-0.5 rounded-full bg-amber-50 text-amber-700">{missingLabel(o.kyc_missing[0], t)}</span>}
            <span className="flex-1" />
            {canEdit && <button type="button" onClick={() => run(() => api.kyc.removeOwner(orgId, o.contact_id))} className="text-xs text-red-600 hover:underline">{t('إزالة', 'Remove')}</button>}
          </li>
        ))}
        {owners.length === 0 && <li className="py-2 text-xs text-gray-400">{t('لم يُسجَّل أي مستفيد', 'None recorded')}</li>}
      </ul>
      {canEdit && contacts.length > 0 && (
        <form className="flex flex-wrap items-end gap-2" onSubmit={(e) => {
          e.preventDefault()
          if (!form.contact_id) return
          run(() => api.kyc.setOwner(orgId, { contact_id: form.contact_id, ownership_pct: form.pct ? Number(form.pct) : null, role: form.role }))
        }}>
          <select value={form.contact_id} onChange={(e) => setForm({ ...form, contact_id: e.target.value })} className="px-2 py-1.5 border border-gray-300 rounded-lg text-sm" aria-label={t('جهة الاتصال', 'Contact')}>
            <option value="">{t('— اختر جهة اتصال —', '— choose a contact —')}</option>
            {contacts.map((c) => <option key={c.id} value={c.id}>{c.full_name}</option>)}
          </select>
          <input type="number" min={0.01} max={100} step="0.01" value={form.pct} onChange={(e) => setForm({ ...form, pct: e.target.value })} placeholder="%" className="w-20 px-2 py-1.5 border border-gray-300 rounded-lg text-sm" aria-label={t('نسبة الملكية', 'Ownership %')} />
          <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} className="px-2 py-1.5 border border-gray-300 rounded-lg text-sm" aria-label={t('الصفة', 'Role')}>
            {['owner', 'controller', 'signatory'].map((r) => <option key={r} value={r}>{roleLabel(r)}</option>)}
          </select>
          <button type="submit" className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">{t('إضافة', 'Add')}</button>
        </form>
      )}
      <p className="text-[11px] text-gray-500">{t('أضف المستفيدين كجهات اتصال في المؤسسة أولاً، ثم سجّل وثائق هويتهم.', 'Add the owners as contacts of the organisation first, then record their identity documents.')}</p>
      {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
    </div>
  )
}
