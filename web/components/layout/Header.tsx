'use client'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { NotificationBell } from './NotificationBell'
import { api } from '@/lib/api'
import { getRefreshToken } from '@/lib/auth'

export function Header({ title }: { title: string }) {
  const { user, logout } = useAuthStore()
  const { lang, setLang, t } = useLang()
  const router = useRouter()

  const handleLogout = async () => {
    const rt = getRefreshToken()
    if (rt) await api.auth.logout(rt).catch(() => {})
    logout()
    router.push('/login')
  }

  return (
    <header className="h-16 flex items-center justify-between px-6 bg-white/80 backdrop-blur-md border-b border-surface-200/70 shrink-0 sticky top-0 z-30">
      <h1 className="font-semibold text-surface-900 text-[15px] tracking-tight">{title}</h1>

      <div className="flex items-center gap-2">
        {/* RTL/LTR toggle */}
        <button
          onClick={() => setLang(lang === 'ar' ? 'en' : 'ar')}
          className="text-xs px-2.5 py-1.5 rounded-lg border border-surface-200 text-surface-600 hover:bg-surface-100 hover:text-surface-900 transition-colors font-medium"
        >
          {lang === 'ar' ? 'EN' : 'عربي'}
        </button>

        <NotificationBell />

        {/* Divider */}
        <span className="w-px h-6 bg-surface-200 mx-1" aria-hidden="true" />

        {/* Avatar + logout */}
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-500 to-primary-700 text-white text-[13px] font-semibold flex items-center justify-center shadow-card">
            {user?.name?.[0]?.toUpperCase() ?? 'U'}
          </div>
          <button
            onClick={handleLogout}
            className="text-xs text-surface-500 hover:text-red-500 transition-colors font-medium"
          >
            {t('خروج', 'Sign out')}
          </button>
        </div>
      </div>
    </header>
  )
}
