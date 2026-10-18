'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

export default function SecuritySettingsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [s, setS] = useState<{ require_2fa: boolean; users_without_2fa: number } | null>(null)
  const [error, setError] = useState('')

  const load = () => api.twoFactor.companySettings().then(setS).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    load()
  }, [user, router])

  if (user?.role !== 'admin' || !s) return null

  const toggle = async (on: boolean) => {
    setError('')
    try { await api.twoFactor.setCompanyRequirement(on); load() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('الأمان', 'Security')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl bg-white rounded-xl border border-gray-200 p-5 space-y-3">
          <label className="flex items-start gap-3">
            <input type="checkbox" className="mt-1" checked={s.require_2fa} onChange={(e) => toggle(e.target.checked)} />
            <span>
              <span className="block text-sm font-medium text-gray-900">{t('إلزام جميع المستخدمين بالتحقق بخطوتين', 'Require two-step verification for every user')}</span>
              <span className="block text-xs text-gray-500 mt-1">
                {t('يُطلب من المستخدمين الذين لم يفعّلوه إعداده عند تسجيل الدخول التالي. لا تنتهي الجلسات الحالية.',
                  'Users who have not turned it on are asked to set it up at their next sign-in. Current sessions are not ended.')}
              </span>
            </span>
          </label>
          <p className="text-xs text-gray-600">
            {s.users_without_2fa === 0
              ? t('جميع المستخدمين النشطين فعّلوا التحقق بخطوتين.', 'Every active user has two-step verification on.')
              : t(`${s.users_without_2fa} من المستخدمين النشطين لم يفعّلوه بعد.`, `${s.users_without_2fa} active user(s) have not turned it on yet.`)}
          </p>
          <p className="text-xs text-gray-500">
            {t('لمستخدم فقد هاتفه: أعد تعيين التحقق بخطوتين من صفحة المستخدمين.', 'For a user who lost their phone: reset two-step verification from the Users page.')}
          </p>
          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
        </div>
      </main>
    </div>
  )
}
