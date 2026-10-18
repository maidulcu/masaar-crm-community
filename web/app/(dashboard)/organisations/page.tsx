'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api, type Organisation } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

export default function OrganisationsPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const canEdit = user?.role === 'admin' || user?.role === 'agent'
  const [orgs, setOrgs] = useState<Organisation[]>([])
  const [search, setSearch] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api.organisations.list(search).then(r => setOrgs(r.data)).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [search])
  useEffect(() => { const h = setTimeout(load, 250); return () => clearTimeout(h) }, [load])

  const create = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    try { await api.organisations.create({ name }); setName(''); load() } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الشركات', 'Organisations')} />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <div className="flex flex-wrap gap-3">
            <input aria-label={t('بحث', 'Search')} placeholder={t('بحث بالاسم', 'Search by name')} value={search} onChange={e => setSearch(e.target.value)}
              className="flex-1 min-w-[200px] px-3 py-2 text-sm border border-surface-200 rounded-lg bg-white" />
            {canEdit && (
              <form onSubmit={create} className="flex gap-2">
                <input aria-label={t('اسم الشركة', 'Organisation name')} required placeholder={t('شركة جديدة', 'New organisation')} value={name}
                  onChange={e => setName(e.target.value)} className="px-3 py-2 text-sm border border-surface-200 rounded-lg bg-white" />
                <button className="px-3 py-2 text-sm font-medium bg-primary-600 text-white rounded-lg">{t('إضافة', 'Add')}</button>
              </form>
            )}
          </div>
          {error && <p role="alert" className="text-xs text-red-600">{error}</p>}
          <ul className="bg-white rounded-2xl border border-surface-200/70 divide-y divide-surface-100">
            {orgs.length === 0 && <li className="p-5 text-sm text-surface-400">{t('لا توجد شركات', 'No organisations yet')}</li>}
            {orgs.map(o => (
              <li key={o.id}>
                <Link href={`/organisations/${o.id}`} className="flex justify-between items-center px-5 py-3 hover:bg-surface-50">
                  <span className="text-sm font-medium text-surface-900">{o.name}</span>
                  <span className="text-xs text-surface-500">{o.contact_count} {t('جهة اتصال', 'contacts')}</span>
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </main>
    </div>
  )
}
