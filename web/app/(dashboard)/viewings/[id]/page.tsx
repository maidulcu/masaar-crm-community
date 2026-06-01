'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Viewing } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  scheduled: 'bg-yellow-100 text-yellow-700',
  confirmed: 'bg-blue-100 text-blue-700',
  checked_in: 'bg-indigo-100 text-indigo-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-700',
  no_show: 'bg-gray-100 text-gray-600',
}

const nextStatuses: Record<string, string[]> = {
  scheduled: ['confirmed', 'cancelled'],
  confirmed: ['checked_in', 'cancelled'],
  checked_in: ['completed', 'no_show'],
  completed: [],
  cancelled: [],
  no_show: [],
}

const statusLabels: Record<string, { en: string; ar: string }> = {
  scheduled: { en: 'Scheduled', ar: 'مجدول' },
  confirmed: { en: 'Confirmed', ar: 'مؤكد' },
  checked_in: { en: 'Checked In', ar: 'تم الحضور' },
  completed: { en: 'Completed', ar: 'مكتمل' },
  cancelled: { en: 'Cancelled', ar: 'ملغي' },
  no_show: { en: 'No Show', ar: 'لم يحضر' },
}

export default function ViewingDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const isAdmin = user?.role === 'admin'

  const [viewing, setViewing] = useState<Viewing | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    if (!id) return
    setLoading(true)
    try {
      const v = await api.viewings.get(id) as Viewing
      setViewing(v)
    } catch { setError('Viewing not found') }
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleStatus = async (status: string) => {
    try {
      await api.viewings.updateStatus(id, status)
      load()
    } catch {
      alert(t('فشل تحديث الحالة', 'Failed to update status'))
    }
  }

  const handleDelete = async () => {
    if (!confirm(t('هل أنت متأكد من حذف هذه الزيارة؟', 'Delete this viewing?'))) return
    try {
      await api.viewings.delete(id)
      router.push('/viewings')
    } catch {
      alert(t('فشل الحذف', 'Failed to delete'))
    }
  }

  if (loading) return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الزيارة', 'Viewing')} />
      <div className="flex-1 flex items-center justify-center text-sm text-gray-400">{t('جاري التحميل...', 'Loading...')}</div>
    </div>
  )

  if (error || !viewing) return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الزيارة', 'Viewing')} />
      <div className="flex-1 flex flex-col items-center justify-center gap-4">
        <p className="text-sm text-red-500">{error || t('الزيارة غير موجودة', 'Viewing not found')}</p>
        <Link href="/viewings" className="text-sm text-brand-600 hover:underline">{t('العودة', 'Back')}</Link>
      </div>
    </div>
  )

  const next = nextStatuses[viewing.status] || []

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تفاصيل الزيارة', 'Viewing Details')} back="/viewings" />

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-2xl mx-auto space-y-6">
          <div className="bg-white rounded-xl border border-gray-100 p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-lg font-semibold text-gray-900">
                  {viewing.contact?.full_name || viewing.contact_id}
                </h2>
                <p className="text-xs text-gray-400 mt-0.5">
                  {viewing.contact?.phone_wa && `${viewing.contact.phone_wa} · `}
                  {new Date(viewing.scheduled_at).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', {
                    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
                    hour: '2-digit', minute: '2-digit',
                  })}
                </p>
              </div>
              <span className={clsx('text-sm font-medium px-3 py-1 rounded-full', statusColor[viewing.status])}>
                {statusLabels[viewing.status]?.[lang === 'ar' ? 'ar' : 'en'] || viewing.status}
              </span>
            </div>

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-400 text-xs">{t('العقار', 'Property')}</span>
                <p className="font-medium text-gray-800">{viewing.listing_title || '—'}</p>
              </div>
              <div>
                <span className="text-gray-400 text-xs">{t('الموظف', 'Agent')}</span>
                <p className="font-medium text-gray-800">{viewing.agent_name || '—'}</p>
              </div>
              <div>
                <span className="text-gray-400 text-xs">{t('المدة', 'Duration')}</span>
                <p className="font-medium text-gray-800">{viewing.duration_min} {t('دقيقة', 'min')}</p>
              </div>
              <div>
                <span className="text-gray-400 text-xs">{t('العنوان', 'Address')}</span>
                <p className="font-medium text-gray-800">{viewing.address || '—'}</p>
              </div>
            </div>

            {viewing.notes && (
              <div className="mt-4 pt-4 border-t border-gray-100">
                <span className="text-gray-400 text-xs">{t('ملاحظات', 'Notes')}</span>
                <p className="text-sm text-gray-700 mt-1 whitespace-pre-wrap">{viewing.notes}</p>
              </div>
            )}

            {viewing.checked_in_at && (
              <div className="mt-4 pt-4 border-t border-gray-100 text-xs text-gray-400">
                {t('وقت الحضور', 'Checked in')}: {new Date(viewing.checked_in_at).toLocaleString()}
                {viewing.checked_out_at && ` · ${t('وقت الانتهاء', 'Checked out')}: ${new Date(viewing.checked_out_at).toLocaleString()}`}
              </div>
            )}

            {isAgent && next.length > 0 && (
              <div className="mt-5 pt-4 border-t border-gray-100 flex gap-2">
                {next.map(s => (
                  <button key={s} onClick={() => handleStatus(s)}
                    className={clsx('px-4 py-1.5 text-sm font-medium rounded-lg transition-colors',
                      s === 'cancelled' || s === 'no_show'
                        ? 'border border-red-200 text-red-600 hover:bg-red-50'
                        : 'bg-brand-600 text-white hover:bg-brand-700')}>
                    {statusLabels[s]?.[lang === 'ar' ? 'ar' : 'en'] || s}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="flex justify-between">
            <Link href="/viewings" className="text-sm text-gray-500 hover:text-gray-700 transition-colors">
              ← {t('العودة للقائمة', 'Back to list')}
            </Link>
            {isAdmin && (
              <button onClick={handleDelete}
                className="text-sm text-red-500 hover:text-red-700 transition-colors">
                {t('حذف', 'Delete')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
