'use client'
import { useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { lang, setLang, t } = useLang()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.auth.forgotPassword(email)
      setSent(true)
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally {
      setLoading(false)
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

        {sent ? (
          <div className="bg-white rounded-2xl shadow-card border border-surface-200/70 p-8 space-y-4 text-center">
            <div className="mx-auto w-12 h-12 rounded-2xl bg-primary-50 text-primary-600 flex items-center justify-center mb-1">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-surface-900">
              {t('تحقق من بريدك', 'Check your email')}
            </h3>
            <p className="text-sm text-surface-500">
              {t(
                'إذا كان البريد الإلكتروني مسجلاً، ستصلك رسالة برابط إعادة تعيين كلمة المرور خلال دقائق.',
                'If that email is registered, a password reset link will arrive in a few minutes.'
              )}
            </p>
            <a href="/login" className="inline-block text-sm font-medium text-primary-600 hover:text-primary-700">
              {t('العودة لتسجيل الدخول', 'Back to sign in')}
            </a>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4">
            <div>
              <h2 className="text-base font-semibold text-surface-900 mb-1">
                {t('إعادة تعيين كلمة المرور', 'Reset your password')}
              </h2>
              <p className="text-xs text-surface-500">
                {t('أدخل بريدك وسنرسل لك رابط إعادة التعيين', 'Enter your email and we\'ll send a reset link')}
              </p>
            </div>

            <div>
              <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                {t('البريد الإلكتروني', 'Email')}
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoFocus
                autoComplete="email"
                className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
                placeholder={t('أدخل بريدك الإلكتروني', 'Enter your email')}
              />
            </div>

            {error && (
              <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed"
            >
              {loading ? t('جاري الإرسال...', 'Sending...') : t('إرسال رابط الإعادة', 'Send reset link')}
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
