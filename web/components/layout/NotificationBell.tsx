'use client'
import { useState } from 'react'
import Link from 'next/link'
import { useNotifications } from '@/hooks/useNotifications'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import clsx from 'clsx'

export function NotificationBell() {
  const user = useAuthStore((s) => s.user)
  const { notifications, unread, markRead, markAllRead } = useNotifications(user?.id ?? null)
  const [open, setOpen] = useState(false)
  const { t, isRtl } = useLang()

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="relative p-2 rounded-lg text-surface-500 hover:bg-surface-100 hover:text-surface-900 transition-colors"
        aria-label="Notifications"
      >
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75}
            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
        </svg>
        {unread > 0 && (
          <span className="absolute top-1 end-1 min-w-[16px] h-4 px-1 bg-red-500 ring-2 ring-white text-white text-[10px] font-bold rounded-full flex items-center justify-center">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className={clsx(
            "absolute top-12 z-20 w-80 bg-white rounded-2xl shadow-pop border border-surface-200/70 overflow-hidden animate-scale-in origin-top",
            isRtl ? "start-0" : "end-0"
          )}>
            <div className="flex items-center justify-between px-4 py-3 border-b border-surface-200/70">
              <span className="font-semibold text-sm text-surface-900">
                {t('الإشعارات', 'Notifications')}
              </span>
              {unread > 0 && (
                <button
                  onClick={markAllRead}
                  className="text-xs text-primary-600 hover:text-primary-700 font-medium"
                >
                  {t('تحديد الكل كمقروء', 'Mark all read')}
                </button>
              )}
            </div>
            <div className="max-h-80 overflow-y-auto divide-y divide-surface-100">
              {notifications.length === 0 ? (
                <p className="px-4 py-8 text-sm text-surface-400 text-center">
                  {t('لا توجد إشعارات', 'No notifications')}
                </p>
              ) : (
                notifications.slice(0, 15).map((n) => (
                  <button
                    key={n.id}
                    onClick={() => markRead(n.id)}
                    className={clsx(
                      "w-full text-start px-4 py-3 text-sm hover:bg-surface-50 transition-colors",
                      !n.read && "bg-primary-50/60"
                    )}
                  >
                    <p className="font-medium text-surface-900">{n.title}</p>
                    {n.body && <p className="text-surface-500 text-xs mt-0.5">{n.body}</p>}
                  </button>
                ))
              )}
            </div>
            <Link
              href="/notifications"
              onClick={() => setOpen(false)}
              className="block px-4 py-2.5 text-sm text-center text-primary-600 hover:bg-surface-50 font-medium border-t border-surface-200/70"
            >
              {t('عرض الكل', 'View all')}
            </Link>
          </div>
        </>
      )}
    </div>
  )
}
