'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { api, type ContactProfile, type Organisation } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface Props {
  contactId: string
  profile?: ContactProfile
  canEdit: boolean
  onSaved: () => void
}

const EMPTY: ContactProfile = {
  secondary_phones: [], nationality: '', source: '', preferred_contact_time: '',
  address: '', notes: '', job_title: '', organisation_id: null,
}

/** The contact's optional details: other numbers, nationality, organisation, notes. */
export default function ProfilePanel({ contactId, profile, canEdit, onSaved }: Props) {
  const { t } = useLang()
  const p = profile ?? EMPTY
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<ContactProfile>(p)
  const [phones, setPhones] = useState('')
  const [orgs, setOrgs] = useState<Organisation[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    if (editing) api.organisations.list().then(r => setOrgs(r.data)).catch(() => {})
  }, [editing])

  const start = () => { setDraft(p); setPhones(p.secondary_phones.join(', ')); setEditing(true) }
  const save = async () => {
    setError('')
    try {
      await api.contacts.update(contactId, {
        profile: { ...draft, secondary_phones: phones.split(',').map(s => s.trim()).filter(Boolean) },
      })
      setEditing(false)
      onSaved()
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  const input = 'w-full px-2.5 py-1.5 text-sm border border-surface-200 rounded-lg bg-white'
  const set = (k: keyof ContactProfile) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) =>
    setDraft(d => ({ ...d, [k]: e.target.value }))
  const row = (label: string, value: React.ReactNode) => (
    <div><dt className="text-xs text-surface-500">{label}</dt><dd className="text-sm text-surface-900 break-words">{value || '—'}</dd></div>
  )

  return (
    <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-surface-900">{t('الملف الشخصي', 'Profile')}</h3>
        {canEdit && !editing && <button onClick={start} className="text-xs text-primary-600 hover:underline">{t('تعديل', 'Edit')}</button>}
      </div>
      {!editing ? (
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {row(t('الشركة', 'Organisation'), p.organisation_id
            ? <Link href={`/organisations/${p.organisation_id}`} className="text-primary-600 hover:underline">{p.organisation_name}</Link> : '')}
          {row(t('المسمى الوظيفي', 'Job title'), p.job_title)}
          {row(t('أرقام أخرى', 'Other numbers'), <span dir="ltr">{p.secondary_phones.join(', ')}</span>)}
          {row(t('الجنسية', 'Nationality'), p.nationality)}
          {row(t('المصدر', 'Source'), p.source)}
          {row(t('وقت التواصل المفضل', 'Preferred contact time'), p.preferred_contact_time)}
          {row(t('العنوان', 'Address'), p.address)}
          {row(t('ملاحظات', 'Notes'), <span className="whitespace-pre-line">{p.notes}</span>)}
        </dl>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <label className="text-xs text-surface-500">{t('الشركة', 'Organisation')}
            <select className={input} value={draft.organisation_id ?? ''} onChange={e => setDraft(d => ({ ...d, organisation_id: e.target.value || null }))}>
              <option value="">—</option>
              {orgs.map(o => <option key={o.id} value={o.id}>{o.name}</option>)}
            </select>
          </label>
          <label className="text-xs text-surface-500">{t('المسمى الوظيفي', 'Job title')}<input className={input} value={draft.job_title} onChange={set('job_title')} /></label>
          <label className="text-xs text-surface-500">{t('أرقام أخرى (مفصولة بفواصل)', 'Other numbers (comma separated)')}<input dir="ltr" className={input} value={phones} onChange={e => setPhones(e.target.value)} placeholder="+971501234567" /></label>
          <label className="text-xs text-surface-500">{t('الجنسية (رمز الدولة)', 'Nationality (country code)')}<input dir="ltr" maxLength={2} className={input} value={draft.nationality} onChange={set('nationality')} placeholder="AE" /></label>
          <label className="text-xs text-surface-500">{t('المصدر', 'Source')}<input className={input} value={draft.source} onChange={set('source')} placeholder={t('إحالة، معرض…', 'Referral, event…')} /></label>
          <label className="text-xs text-surface-500">{t('وقت التواصل المفضل', 'Preferred contact time')}<input className={input} value={draft.preferred_contact_time} onChange={set('preferred_contact_time')} /></label>
          <label className="text-xs text-surface-500 sm:col-span-2">{t('العنوان', 'Address')}<input className={input} value={draft.address} onChange={set('address')} /></label>
          <label className="text-xs text-surface-500 sm:col-span-2">{t('ملاحظات', 'Notes')}<textarea rows={3} className={input} value={draft.notes} onChange={set('notes')} /></label>
          <div className="flex gap-2 sm:col-span-2">
            <button onClick={save} className="px-3 py-1.5 text-xs font-medium bg-primary-600 text-white rounded-lg">{t('حفظ', 'Save')}</button>
            <button onClick={() => setEditing(false)} className="px-3 py-1.5 text-xs text-surface-500">{t('إلغاء', 'Cancel')}</button>
          </div>
        </div>
      )}
      {error && <p role="alert" className="text-xs text-red-600 mt-2">{error}</p>}
    </div>
  )
}
