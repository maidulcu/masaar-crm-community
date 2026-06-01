'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useDebounce } from '@/hooks/useDebounce'
import type { WhatsAppThread } from '@/types'
import clsx from 'clsx'

const statusTabs = [
  { key: '', label: { en: 'All', ar: 'الكل' } },
  { key: 'open', label: { en: 'Open', ar: 'مفتوح' } },
  { key: 'pending', label: { en: 'Pending', ar: 'معلق' } },
  { key: 'closed', label: { en: 'Closed', ar: 'مغلق' } },
]

const statusColor: Record<string, string> = {
  open:    'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-100',
  pending: 'bg-gold-50 text-gold-700 ring-1 ring-gold-100',
  closed:  'bg-surface-100 text-surface-600 ring-1 ring-surface-200',
}

const PAGE_SIZE = 40

export default function InboxPage() {
  const [threads, setThreads] = useState<WhatsAppThread[]>([])
  const [status, setStatus] = useState('')
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const { lang, t } = useLang()

  const debouncedStatus = useDebounce(status, 300)

  useEffect(() => {
    setPage(1)
  }, [debouncedStatus])

  useEffect(() => {
    setLoading(true)
    api.threads.list({ status: debouncedStatus, page, limit: PAGE_SIZE }).then((res: any) => {
      const items = Array.isArray(res) ? res : []
      if (page === 1) {
        setThreads(items)
      } else {
        setThreads(prev => [...prev, ...items])
      }
      setHasMore(items.length === PAGE_SIZE)
    }).catch(() => {
      if (page === 1) setThreads([])
    }).finally(() => setLoading(false))
  }, [debouncedStatus, page])

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('صندوق الرسائل', 'Inbox')} />

      {/* Status tabs */}
      <div className="flex gap-1 px-6 pt-3 pb-0 border-b border-surface-200/70 bg-white">
        {statusTabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setStatus(tab.key)}
            className={clsx(
              'px-3.5 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors',
              status === tab.key
                ? 'border-primary-600 text-primary-700'
                : 'border-transparent text-surface-500 hover:text-surface-900'
            )}
          >
            {lang === 'ar' ? tab.label.ar : tab.label.en}
          </button>
        ))}
      </div>

      {/* Thread list */}
      <div className="flex-1 overflow-y-auto bg-white">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-surface-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : threads.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-72 gap-3 text-center">
            <div className="w-16 h-16 rounded-2xl bg-surface-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-surface-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </div>
            <p className="text-sm font-medium text-surface-600">{t('لا توجد محادثات', 'No threads')}</p>
            <p className="text-xs text-surface-400">{t('ستظهر المحادثات عند وصول رسائل واتساب', 'WhatsApp messages will appear here')}</p>
          </div>
        ) : (
          <>
            <ul className="divide-y divide-surface-100">
              {threads.map((thread) => (
                <li key={thread.id}>
                  <Link
                    href={`/inbox/${thread.id}`}
                    className="flex items-center gap-4 px-6 py-4 hover:bg-surface-50/70 transition-colors"
                  >
                    {/* Avatar */}
                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-primary-100 to-primary-200 text-primary-700 font-semibold text-sm flex items-center justify-center shrink-0 ring-1 ring-primary-200/50">
                      {thread.contact?.full_name?.[0]?.toUpperCase() ?? '?'}
                    </div>

                    {/* Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-2">
                        <p className="font-semibold text-sm text-surface-900 truncate">
                          {thread.contact?.full_name ?? thread.contact_id}
                        </p>
                        <span className={clsx('text-[10px] font-semibold px-2 py-0.5 rounded-full shrink-0 capitalize', statusColor[thread.thread_status])}>
                          {thread.thread_status}
                        </span>
                      </div>
                      <p className="text-xs text-surface-500 mt-0.5 truncate">
                        <span className="font-mono">{thread.contact?.phone_wa}</span>
                        <span className="text-surface-300"> · </span>
                        {thread.message_count} {t('رسالة', 'messages')}
                      </p>
                      {thread.ai_summary && (
                        <p className="text-xs text-surface-500 mt-1 truncate italic">{thread.ai_summary}</p>
                      )}
                    </div>

                    {/* Time */}
                    {thread.last_message_at && (
                      <span className="text-[11px] text-surface-400 shrink-0 font-medium tabular-nums">
                        {new Date(thread.last_message_at).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', {
                          month: 'short', day: 'numeric'
                        })}
                      </span>
                    )}
                  </Link>
                </li>
              ))}
            </ul>
            {hasMore && (
              <div className="py-4 flex justify-center">
                <button
                  onClick={() => setPage(p => p + 1)}
                  disabled={loading}
                  className="px-6 py-2 text-sm font-medium bg-surface-100 text-surface-600 rounded-lg hover:bg-surface-200 disabled:opacity-50 transition-colors"
                >
                  {t('تحميل المزيد', 'Load More')}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
