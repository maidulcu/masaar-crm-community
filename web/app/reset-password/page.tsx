'use client'
import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

function ResetPasswordContent() {
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle')
  const [error, setError] = useState('')
  const router = useRouter()
  const searchParams = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const { lang, setLang, t } = useLang()

  useEffect(() => {
    if (!token) {
      setStatus('error')
      setError(t('رابط إعادة التعيين غير صالح أو منتهي الصلاحية', 'Reset link is invalid or has expired'))
    }
  }, [token])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (password !== confirm) {
      setError(t('كلمتا المرور غير متطابقتين', 'Passwords do not match'))
      return
    }
    if (password.length < 8) {
      setError(t('يجب أن تكون كلمة المرور 8 أحرف على الأقل', 'Password must be at least 8 characters'))
      return
    }
    setError('')
    setStatus('loading')
    try {
      await api.auth.resetPassword(token, password)
      setStatus('success')
      setTimeout(() => router.push('/login'), 2000)
    } catch (err: any) {
      setStatus('error')
      setError(err.message || t('رابط منتهي الصلاحية أو مستخدم مسبقاً', 'Link expired or already used'))
    }
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center px-4 bg-surface-50 overflow-hidden">
      <div aria-hidden="true" className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute -top-32 -start-32 w-[420px] h-[420px] rounded-full bg-primary-200/40 blur-3xl" />
        <div className="absolute -bottom-32 -end-32 w-[420px] h-[420px] rounded-full bg-gold-200/40 blur-3xl" />
      </div>

      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2.5 mb-3">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-2xl bg-primary-600 text-white text-lg font-bold shadow-card">
              M
            </span>
            <span className="text-2xl font-bold tracking-tight text-surface-900">
              {lang === 'ar' ? 'مسار' : 'Masaar'}
            </span>
          </div>
        </div>

        {status === 'success' ? (
          <div className="bg-white rounded-2xl shadow-card border border-surface-200/70 p-8 space-y-4 text-center">
            <div className="mx-auto w-12 h-12 rounded-2xl bg-green-50 text-green-600 flex items-center justify-center mb-1">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-surface-900">
              {t('تم تعيين كلمة المرور', 'Password updated!')}
            </h3>
            <p className="text-sm text-surface-500">
              {t('جاري تحويلك لصفحة تسجيل الدخول...', 'Redirecting you to sign in...')}
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4">
            <div>
              <h2 className="text-base font-semibold text-surface-900 mb-1">
                {t('تعيين كلمة مرور جديدة', 'Set a new password')}
              </h2>
              <p className="text-xs text-surface-500">
                {t('يجب أن تكون 8 أحرف على الأقل', 'Must be at least 8 characters')}
              </p>
            </div>

            <div>
              <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                {t('كلمة المرور الجديدة', 'New password')}
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={8}
                autoFocus
                autoComplete="new-password"
                disabled={status === 'error' && !token}
                className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow disabled:bg-surface-50 disabled:cursor-not-allowed"
                placeholder="••••••••"
              />
            </div>

            <div>
              <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                {t('تأكيد كلمة المرور', 'Confirm password')}
              </label>
              <input
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                required
                autoComplete="new-password"
                disabled={status === 'error' && !token}
                className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow disabled:bg-surface-50 disabled:cursor-not-allowed"
                placeholder="••••••••"
              />
            </div>

            {error && (
              <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg">{error}</p>
            )}

            <button
              type="submit"
              disabled={status === 'loading' || (status === 'error' && !token)}
              className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {status === 'loading' ? t('جاري الحفظ...', 'Saving...') : t('تعيين كلمة المرور', 'Set password')}
            </button>

            <div className="text-center">
              <a href="/login" className="text-xs font-medium text-surface-500 hover:text-primary-600">
                {t('العودة لتسجيل الدخول', 'Back to sign in')}
              </a>
            </div>
          </form>
        )}

        <div className="mt-5 text-center">
          <button
            onClick={() => setLang(lang === 'ar' ? 'en' : 'ar')}
            className="text-xs text-surface-500 hover:text-surface-900 transition-colors"
          >
            {lang === 'ar' ? 'Switch to English' : 'التبديل إلى العربية'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-surface-50">
        <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-600" />
      </div>
    }>
      <ResetPasswordContent />
    </Suspense>
  )
}
