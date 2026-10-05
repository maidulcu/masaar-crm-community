'use client'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { NotificationBell } from './NotificationBell'
import { api } from '@/lib/api'

export function Header({ title, back }: { title: string; back?: string }) {
  const { user, logout } = useAuthStore()
  const { lang, setLang, t } = useLang()
  const router = useRouter()

  const handleLogout = async () => {
    // Revokes the refresh token server-side and clears the HttpOnly cookie.
    await api.auth.logout().catch(() => {})
    logout()
    router.push('/login')
  }

  return (
    <header className="h-16 flex items-center justify-between px-6 bg-white/80 backdrop-blur-md border-b border-surface-200/70 shrink-0 sticky top-0 z-30">
      <div className="flex items-center gap-3 min-w-0">
        {back && (
          <Link
            href={back}
            aria-label={t('رجوع', 'Back')}
            className="text-surface-500 hover:text-surface-900 transition-colors rtl:rotate-180 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 rounded-md p-0.5"
          >
            <span aria-hidden="true">←</span>
          </Link>
        )}
        <h1 className="font-semibold text-surface-900 text-[15px] tracking-tight truncate">{title}</h1>
      </div>

      <div className="flex items-center gap-2">
        {/* RTL/LTR toggle */}
        <button
          onClick={() => setLang(lang === 'ar' ? 'en' : 'ar')}
          aria-label={lang === 'ar' ? 'تغيير اللغة إلى الإنجليزية' : 'Switch language to English'}
          title={lang === 'ar' ? 'English' : 'عربي'}
          className="text-xs px-2.5 py-1.5 rounded-lg border border-surface-200 text-surface-600 hover:bg-surface-100 hover:text-surface-900 transition-colors font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
        >
          {lang === 'ar' ? 'EN' : 'عربي'}
        </button>

        <NotificationBell />

        {/* Divider */}
        <span className="w-px h-6 bg-surface-200 mx-1" aria-hidden="true" />

        {/* Avatar + logout */}
        <div className="flex items-center gap-2.5">
          <div
            role="img"
            title={user?.name || user?.email || t('المستخدم', 'User')}
            aria-label={user?.name || user?.email || t('المستخدم', 'User')}
            className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-500 to-primary-700 text-white text-[13px] font-semibold flex items-center justify-center shadow-card"
          >
            {user?.name?.[0]?.toUpperCase() ?? 'U'}
          </div>
          <button
            onClick={handleLogout}
            aria-label={t('تسجيل الخروج', 'Sign out')}
            className="text-xs text-surface-500 hover:text-red-500 transition-colors font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 rounded px-1.5 py-1"
          >
            {t('خروج', 'Sign out')}
          </button>
        </div>
      </div>
    </header>
  )
}
