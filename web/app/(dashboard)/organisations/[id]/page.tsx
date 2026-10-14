'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams, useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type Organisation } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

const FIELDS: { key: keyof Organisation; en: string; ar: string; ltr?: boolean }[] = [
  { key: 'name', en: 'Name', ar: 'الاسم' },
  { key: 'trade_license_no', en: 'Trade licence no.', ar: 'رقم الرخصة التجارية', ltr: true },
  { key: 'trn', en: 'TRN', ar: 'الرقم الضريبي', ltr: true },
  { key: 'phone', en: 'Phone', ar: 'الهاتف', ltr: true },
  { key: 'email', en: 'Email', ar: 'البريد الإلكتروني', ltr: true },
  { key: 'website', en: 'Website', ar: 'الموقع الإلكتروني', ltr: true },
  { key: 'address', en: 'Address', ar: 'العنوان' },
  { key: 'notes', en: 'Notes', ar: 'ملاحظات' },
]

export default function OrganisationPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const canEdit = user?.role === 'admin' || user?.role === 'agent'
  const [org, setOrg] = useState<Organisation | null>(null)
  const [contacts, setContacts] = useState<{ id: string; full_name: string; phone_wa: string; email: string }[]>([])
  const [draft, setDraft] = useState<Partial<Organisation>>({})
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api.organisations.get(id).then(r => { setOrg(r.organisation); setContacts(r.contacts) })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [id])
  useEffect(() => { load() }, [load])

  const save = async () => {
    setError('')
    try { await api.organisations.update(id, draft); setEditing(false); load() } catch (e: unknown) { setError(e instanceof Error ? e.message : 'Failed') }
  }
  const remove = async () => {
    if (!confirm(t('حذف هذه الشركة؟ ستبقى جهات الاتصال.', 'Delete this organisation? Its contacts are kept.'))) return
    try { await api.organisations.delete(id); router.push('/organisations') } catch (e: unknown) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  if (!org) return <div className="flex-1 flex items-center justify-center text-sm text-surface-400">{error || t('جاري التحميل...', 'Loading...')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={org.name} />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
            <div className="flex justify-between mb-3">
              <h2 className="text-sm font-semibold text-surface-900">{t('التفاصيل', 'Details')}</h2>
              {canEdit && !editing && (
                <div className="flex gap-3">
                  <button onClick={() => { setDraft(org); setEditing(true) }} className="text-xs text-primary-600 hover:underline">{t('تعديل', 'Edit')}</button>
                  {user?.role === 'admin' && <button onClick={remove} className="text-xs text-red-600 hover:underline">{t('حذف', 'Delete')}</button>}
                </div>
              )}
            </div>
            <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {FIELDS.map(f => (
                <div key={f.key} className={f.key === 'notes' || f.key === 'address' ? 'sm:col-span-2' : ''}>
                  <dt className="text-xs text-surface-500"><label htmlFor={`org-${f.key}`}>{lang === 'ar' ? f.ar : f.en}</label></dt>
                  <dd className="text-sm text-surface-900">
                    {editing ? (
                      <input id={`org-${f.key}`} dir={f.ltr ? 'ltr' : undefined} className="w-full px-2.5 py-1.5 text-sm border border-surface-200 rounded-lg"
                        value={String(draft[f.key] ?? '')} onChange={e => setDraft(d => ({ ...d, [f.key]: e.target.value }))} />
                    ) : (String(org[f.key] ?? '') || '—')}
                  </dd>
                </div>
              ))}
            </dl>
            {editing && (
              <div className="flex gap-2 mt-3">
                <button onClick={save} className="px-3 py-1.5 text-xs font-medium bg-primary-600 text-white rounded-lg">{t('حفظ', 'Save')}</button>
                <button onClick={() => setEditing(false)} className="px-3 py-1.5 text-xs text-surface-500">{t('إلغاء', 'Cancel')}</button>
              </div>
            )}
            {error && <p role="alert" className="text-xs text-red-600 mt-2">{error}</p>}
          </div>
          <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
            <h2 className="text-sm font-semibold text-surface-900 mb-3">{t('جهات الاتصال', 'Contacts')}</h2>
            {contacts.length === 0 ? <p className="text-sm text-surface-400">{t('اربط جهات الاتصال من ملفها الشخصي.', 'Link contacts from their profile.')}</p> : (
              <ul className="divide-y divide-surface-100">
                {contacts.map(c => (
                  <li key={c.id} className="py-2 flex justify-between text-sm">
                    <Link href={`/contacts/${c.id}`} className="text-primary-600 hover:underline">{c.full_name}</Link>
                    <span dir="ltr" className="text-surface-500">{c.phone_wa}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
