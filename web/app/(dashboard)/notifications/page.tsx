'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { useNotifications } from '@/hooks/useNotifications'
import { api } from '@/lib/api'
import type { Notification } from '@/types'
import clsx from 'clsx'

const typeLabels: Record<string, { en: string; ar: string }> = {
  lead_assigned: { en: 'Lead Assigned', ar: 'عميل محتمل مُسند' },
  lead_stage_changed: { en: 'Stage Changed', ar: 'تغير مرحلة البيع' },
  new_message: { en: 'New Message', ar: 'رسالة جديدة' },
  payment_reminder_sent: { en: 'Payment Reminder', ar: 'تذكير بالدفع' },
  lead_created: { en: 'Lead Created', ar: 'تم إنشاء عميل محتمل' },
}

const typeColors: Record<string, string> = {
  lead_assigned: 'bg-blue-100 text-blue-700',
  lead_stage_changed: 'bg-purple-100 text-purple-700',
  new_message: 'bg-green-100 text-green-700',
  payment_reminder_sent: 'bg-yellow-100 text-yellow-700',
  lead_created: 'bg-indigo-100 text-indigo-700',
}

export default function NotificationsPage() {
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const { notifications, unread, markRead, markAllRead, loadMore } = useNotifications(user?.id ?? null)
  const [items, setItems] = useState<Notification[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [typeFilter, setTypeFilter] = useState('')
  const limit = 20

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res: any = await api.notifications.list({ page })
      setItems(prev => {
        const existing = new Set(prev.map(n => n.id))
        const merged = page === 1 ? (res.data ?? []) : [...prev, ...(res.data ?? []).filter((n: Notification) => !existing.has(n.id))]
        return merged
      })
      setTotal(res.total ?? 0)
    } catch {
      setItems([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (notifications.length > 0) {
      setItems(prev => {
        const existing = new Set(prev.map(n => n.id))
        const newItems = notifications.filter(n => !existing.has(n.id))
        return newItems.length > 0 ? [...newItems, ...prev] : prev
      })
    }
  }, [notifications])

  const filtered = typeFilter ? items.filter(n => n.type === typeFilter) : items

  const fmtDate = (d: string) => {
    const date = new Date(d)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    if (diffMins < 1) return t('الآن', 'Just now')
    if (diffMins < 60) return t(`منذ ${diffMins} دقيقة`, `${diffMins}m ago`)
    const diffHours = Math.floor(diffMins / 60)
    if (diffHours < 24) return t(`منذ ${diffHours} ساعة`, `${diffHours}h ago`)
    return date.toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' })
  }

  const types = Object.keys(typeLabels)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الإشعارات', 'Notifications')} />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('الإشعارات', 'Notifications')}
            {unread > 0 && (
              <span className="ml-2 text-sm font-normal text-gray-500">
                ({unread} {t('غير مقروء', 'unread')})
              </span>
            )}
          </h2>
          <div className="flex gap-2">
            <select
              value={typeFilter}
              onChange={e => setTypeFilter(e.target.value)}
              className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <option value="">{t('الكل', 'All')}</option>
              {types.map(tp => (
                <option key={tp} value={tp}>
                  {lang === 'ar' ? (typeLabels[tp]?.ar ?? tp) : (typeLabels[tp]?.en ?? tp)}
                </option>
              ))}
            </select>
            {unread > 0 && (
              <button
                onClick={markAllRead}
                className="px-3 py-1.5 text-sm text-brand-600 hover:text-brand-700 font-medium border border-gray-200 rounded-lg hover:bg-gray-50"
              >
                {t('تحديد الكل كمقروء', 'Mark all read')}
              </button>
            )}
          </div>
        </div>

        {loading && page === 1 ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : filtered.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
            <p className="text-gray-400 text-sm">{t('لا توجد إشعارات', 'No notifications')}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {filtered.map(n => (
              <div
                key={n.id}
                onClick={() => !n.read && markRead(n.id)}
                className={clsx(
                  'flex items-start gap-3 p-4 rounded-xl border transition-colors cursor-pointer',
                  n.read ? 'bg-white border-gray-200' : 'bg-primary-50/60 border-primary-200'
                )}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={clsx('px-2 py-0.5 rounded text-[10px] font-medium', typeColors[n.type] || 'bg-gray-100 text-gray-600')}>
                      {lang === 'ar' ? (typeLabels[n.type]?.ar ?? n.type) : (typeLabels[n.type]?.en ?? n.type)}
                    </span>
                    {!n.read && <span className="w-2 h-2 rounded-full bg-brand-500" />}
                  </div>
                  <p className={clsx('text-sm', n.read ? 'text-gray-700' : 'text-gray-900 font-medium')}>
                    {n.title}
                  </p>
                  {n.body && (
                    <p className="text-xs text-gray-500 mt-0.5">{n.body}</p>
                  )}
                </div>
                <span className="text-[10px] text-gray-400 whitespace-nowrap shrink-0">
                  {fmtDate(n.created_at)}
                </span>
              </div>
            ))}
          </div>
        )}

        {total > limit && page * limit < total && (
          <div className="text-center mt-6">
            <button
              onClick={() => setPage(p => p + 1)}
              disabled={loading}
              className="px-6 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50 disabled:opacity-40"
            >
              {loading ? t('جاري التحميل...', 'Loading...') : t('تحميل المزيد', 'Load more')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
