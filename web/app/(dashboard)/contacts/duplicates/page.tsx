'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api, type DuplicateGroup } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

const REASONS = { email: { en: 'Same email', ar: 'نفس البريد' }, name: { en: 'Same name', ar: 'نفس الاسم' }, phone: { en: 'Shared phone number', ar: 'رقم هاتف مشترك' } }

export default function DuplicatesPage() {
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const [groups, setGroups] = useState<DuplicateGroup[]>([])
  const [keep, setKeep] = useState<Record<number, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    api.duplicates.list().then(setGroups).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed')).finally(() => setLoading(false))
  }, [])
  useEffect(() => { load() }, [load])

  const merge = async (i: number) => {
    const g = groups[i]
    const keepId = keep[i] ?? g.contacts[0].id
    const others = g.contacts.filter(c => c.id !== keepId).map(c => c.id)
    if (!confirm(t('دمج جهات الاتصال؟ لا يمكن التراجع.', `Merge ${others.length} contact(s) into the selected one? This cannot be undone.`))) return
    setError('')
    try { await api.duplicates.merge(keepId, others); load() } catch (e: unknown) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('جهات الاتصال المكررة', 'Duplicate Contacts')} />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <p className="text-sm text-surface-600">
            {t('عند الدمج تنتقل العملاء المحتملون والعروض والمعاينات والمحادثات والموافقات والمهام إلى جهة الاتصال المختارة، وتُحذف البقية.',
              'Merging moves leads, offers, viewings, conversations, consent and tasks to the contact you keep, fills its empty fields, and deletes the others.')}
          </p>
          {error && <p role="alert" className="text-xs text-red-600">{error}</p>}
          {loading ? <p className="text-sm text-surface-400">{t('جاري التحميل...', 'Loading...')}</p> : groups.length === 0 ? (
            <p className="text-sm text-surface-400">{t('لا توجد تكرارات محتملة.', 'No likely duplicates found.')}</p>
          ) : groups.map((g, i) => (
            <fieldset key={g.reason + g.key} className="bg-white rounded-2xl border border-surface-200/70 p-5">
              <legend className="text-xs font-medium text-surface-500 px-1">{lang === 'ar' ? REASONS[g.reason].ar : REASONS[g.reason].en}: <span dir="ltr">{g.key}</span></legend>
              <ul className="space-y-2">
                {g.contacts.map(c => (
                  <li key={c.id} className="flex items-center gap-3 text-sm">
                    {isAdmin && (
                      <input type="radio" name={`keep-${i}`} aria-label={t('الاحتفاظ بهذه', 'Keep this one')}
                        checked={(keep[i] ?? g.contacts[0].id) === c.id} onChange={() => setKeep(k => ({ ...k, [i]: c.id }))} />
                    )}
                    <Link href={`/contacts/${c.id}`} className="text-primary-600 hover:underline">{c.full_name}</Link>
                    <span dir="ltr" className="text-surface-500">{c.phone_wa}</span>
                    <span className="text-surface-400">{c.email}</span>
                  </li>
                ))}
              </ul>
              {isAdmin && (
                <button onClick={() => merge(i)} className="mt-3 px-3 py-1.5 text-xs font-medium bg-primary-600 text-white rounded-lg">
                  {t('دمج في المحددة', 'Merge into selected')}
                </button>
              )}
            </fieldset>
          ))}
        </div>
      </main>
    </div>
  )
}
