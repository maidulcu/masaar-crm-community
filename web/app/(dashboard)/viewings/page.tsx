'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { Viewing, ViewingStatus } from '@/types'

const statusColor: Record<string, string> = {
  scheduled: 'bg-yellow-100 text-yellow-700',
  confirmed: 'bg-blue-100 text-blue-700',
  checked_in: 'bg-indigo-100 text-indigo-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-700',
  no_show: 'bg-gray-100 text-gray-600',
}

const emptyForm = {
  contact_id: '',
  scheduled_at: '',
  duration_min: 30,
  listing_id: '',
  agent_id: '',
  lead_id: '',
  address: '',
  notes: '',
}

export default function ViewingsPage() {
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [viewings, setViewings] = useState<Viewing[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 30
  const [statusFilter, setStatusFilter] = useState('')

  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState(emptyForm)

  useEffect(() => { loadViewings() }, [page, statusFilter])

  const loadViewings = async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = { limit, page }
      if (statusFilter) params.status = statusFilter
      const res: any = await api.viewings.list(params as any)
      setViewings(res?.data ?? [])
      setTotal(res?.meta?.total ?? 0)
    } catch { setViewings([]) }
    finally { setLoading(false) }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.contact_id || !form.scheduled_at) return
    setCreating(true)
    try {
      await api.viewings.create({
        ...form,
        scheduled_at: new Date(form.scheduled_at).toISOString(),
      })
      setShowCreate(false)
      setForm(emptyForm)
      setPage(1)
      loadViewings()
    } catch {
      alert(t('فشل إنشاء الزيارة', 'Failed to create viewing'))
    } finally { setCreating(false) }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الزيارات', 'Viewings')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <p className="text-sm text-gray-500">{total} {t('زيارة', 'viewings')}</p>
                <div className="flex gap-1.5">
                  {['', 'scheduled', 'confirmed', 'completed', 'cancelled'].map(s => (
                    <button key={s} onClick={() => setStatusFilter(s)}
                      className={clsx('text-xs px-2.5 py-1 rounded-full border transition-colors',
                        statusFilter === s
                          ? 'bg-brand-600 text-white border-brand-600'
                          : 'border-gray-200 text-gray-500 hover:bg-gray-50')}>
                      {s ? t(s, s.replace('_', ' ')) : t('الكل', 'All')}
                    </button>
                  ))}
                </div>
              </div>
              {isAgent && (
                <button onClick={() => setShowCreate(true)}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                  {t('+ زيارة جديدة', '+ New Viewing')}
                </button>
              )}
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('العميل', 'Client')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('العقار', 'Property')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('التاريخ', 'Date')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الموظف', 'Agent')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {viewings.map(v => (
                    <tr key={v.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-5 py-3.5 font-medium text-gray-900">
                        {v.contact?.full_name || v.contact_id.slice(0, 8)}
                      </td>
                      <td className="px-5 py-3.5 text-gray-600 text-xs">{v.listing_title || '—'}</td>
                      <td className="px-5 py-3.5 text-gray-600 text-xs">
                        {new Date(v.scheduled_at).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', {
                          day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit',
                        })}
                      </td>
                      <td className="px-5 py-3.5 text-xs text-gray-500">{v.agent_name || '—'}</td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', statusColor[v.status])}>
                          {v.status.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-end">
                        <Link href={`/viewings/${v.id}`}
                          className="text-xs text-brand-600 hover:underline font-medium">
                          {t('عرض', 'View')}
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-6">
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('السابق', 'Prev')}
                </button>
                {Array.from({ length: Math.min(totalPages, 10) }, (_, i) => i + 1).map(p => (
                  <button key={p} onClick={() => setPage(p)}
                    className={clsx('px-3 py-1.5 text-sm rounded-lg', p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50')}>
                    {p}
                  </button>
                ))}
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {showCreate && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('زيارة جديدة', 'New Viewing')}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف العميل', 'Contact ID')} *</label>
                  <input value={form.contact_id} onChange={e => setForm(f => ({ ...f, contact_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف العقار', 'Listing ID')}</label>
                  <input value={form.listing_id} onChange={e => setForm(f => ({ ...f, listing_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('تاريخ الزيارة', 'Date & Time')} *</label>
                <input type="datetime-local" value={form.scheduled_at} onChange={e => setForm(f => ({ ...f, scheduled_at: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('المدة (دقيقة)', 'Duration (min)')}</label>
                  <input type="number" min={15} step={15} value={form.duration_min} onChange={e => setForm(f => ({ ...f, duration_min: parseInt(e.target.value) || 30 }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف الموظف', 'Agent ID')}</label>
                  <input value={form.agent_id} onChange={e => setForm(f => ({ ...f, agent_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('العنوان', 'Address')}</label>
                <input value={form.address} onChange={e => setForm(f => ({ ...f, address: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('ملاحظات', 'Notes')}</label>
                <textarea value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} rows={2}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowCreate(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={creating}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {creating ? '...' : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
