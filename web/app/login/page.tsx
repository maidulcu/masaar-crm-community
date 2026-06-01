'use client'
import { useState, useEffect, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { Turnstile } from '@marsidev/react-turnstile'
import type { TurnstileInstance } from '@marsidev/react-turnstile'
import { api } from '@/lib/api'
import { isLoggedIn } from '@/lib/auth'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import type { LoginResponse } from '@/types'
type LoginMode = 'password' | 'magic-link' | 'sms'

const TURNSTILE_SITE_KEY = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY ?? ''
const DEMO_ENABLED = process.env.NEXT_PUBLIC_DEMO_ENABLED === 'true'

async function verifyTurnstile(token: string): Promise<boolean> {
  const res = await fetch('/api/verify-turnstile', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  const data = await res.json()
  return data.success === true
}

export default function LoginPage() {
  const [showLogin, setShowLogin] = useState(!DEMO_ENABLED)
  const [mode, setMode] = useState<LoginMode>('password')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [phone, setPhone] = useState('')
  const [otp, setOtp] = useState('')
  const [otpSent, setOtpSent] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [magicSent, setMagicSent] = useState(false)
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null)
  const turnstileRef = useRef<TurnstileInstance>(null)
  const { setSession, init, token } = useAuthStore()
  const router = useRouter()
  const { lang, setLang, t } = useLang()

  useEffect(() => { init() }, [init])
  useEffect(() => {
    if (token && isLoggedIn()) router.replace('/pipeline')
  }, [token, router])

  const checkTurnstile = async (): Promise<boolean> => {
    if (!TURNSTILE_SITE_KEY) return true
    if (!turnstileToken) {
      setError(t('يرجى إكمال التحقق من الأمان', 'Please complete the security check'))
      return false
    }
    const ok = await verifyTurnstile(turnstileToken)
    if (!ok) {
      setError(t('فشل التحقق الأمني، يرجى المحاولة مرة أخرى', 'Security check failed, please try again'))
      turnstileRef.current?.reset()
      setTurnstileToken(null)
      return false
    }
    return true
  }

  const handlePasswordLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (!await checkTurnstile()) return
      const res = await api.auth.login(email, password) as LoginResponse
      setSession(res.access_token, res.refresh_token, res.user, res.company)
      if (res.user.lang_pref) setLang(res.user.lang_pref)
      router.push('/pipeline')
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
      turnstileRef.current?.reset()
      setTurnstileToken(null)
    } finally {
      setLoading(false)
    }
  }

  const handleDemoLogin = async () => {
    setError('')
    setLoading(true)
    try {
      const res = await api.auth.login('ahmed@masaar.local', 'Demo@1234') as LoginResponse
      setSession(res.access_token, res.refresh_token, res.user, res.company)
      if (res.user.lang_pref) setLang(res.user.lang_pref)
      router.push('/pipeline')
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally {
      setLoading(false)
    }
  }

  const handleMagicLinkRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (!await checkTurnstile()) return
      await api.auth.requestMagicLink(email, lang)
      setMagicSent(true)
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
      turnstileRef.current?.reset()
      setTurnstileToken(null)
    } finally {
      setLoading(false)
    }
  }

  const handleSMSRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (!await checkTurnstile()) return
      await api.auth.requestSMSOTP(phone, lang)
      setOtpSent(true)
    } catch (err: any) {
      setError(err.message || t('حدث خطأ', 'Something went wrong'))
      turnstileRef.current?.reset()
      setTurnstileToken(null)
    } finally {
      setLoading(false)
    }
  }

  const handleSMSVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await api.auth.verifySMSOTP(phone, otp) as LoginResponse
      setSession(res.access_token, res.refresh_token, res.user, res.company)
      if (res.user.lang_pref) setLang(res.user.lang_pref)
      router.push('/pipeline')
    } catch (err: any) {
      setError(err.message || t('رمز التحقق غير صحيح', 'Invalid OTP'))
    } finally {
      setLoading(false)
    }
  }

  const resetMode = (next: LoginMode) => {
    setMode(next)
    setError('')
    setOtpSent(false)
    setOtp('')
    setMagicSent(false)
    turnstileRef.current?.reset()
    setTurnstileToken(null)
  }

  return (
    <div className="relative min-h-screen flex items-center justify-center px-4 bg-gradient-to-br from-surface-50 via-white to-surface-50 overflow-hidden">
      <div aria-hidden="true" className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute -top-40 -start-40 w-[500px] h-[500px] rounded-full bg-gradient-to-br from-primary-200/30 to-primary-300/20 blur-3xl" />
        <div className="absolute -bottom-40 -end-40 w-[500px] h-[500px] rounded-full bg-gradient-to-br from-gold-200/30 to-gold-300/20 blur-3xl" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full bg-primary-100/10 blur-3xl" />
      </div>

      <div className="w-full max-w-sm animate-slide-up">

        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-2.5 mb-3">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-primary-700 text-white text-lg font-bold shadow-card shadow-primary-200">
              M
            </span>
            <span className="text-2xl font-bold tracking-tight text-surface-900">
              {lang === 'ar' ? 'مسار' : 'Masaar'}
            </span>
          </div>
          <p className="text-surface-500 text-sm">
            {t('نظام إدارة علاقات العملاء الإماراتي', 'CRM built for the UAE')}
          </p>
        </div>

        {!showLogin ? (
          <>
            {/* ────── DEMO HERO ────── */}
            <div className="relative bg-white rounded-2xl shadow-card border border-surface-200/70 p-8 text-center animate-scale-in">
              <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-gold-400 to-gold-500 rounded-t-2xl" />

              <div className="mx-auto w-16 h-16 rounded-2xl bg-gradient-to-br from-gold-100 to-gold-200 text-gold-700 flex items-center justify-center mb-4 ring-1 ring-gold-300/50">
                <svg className="w-8 h-8" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z" />
                </svg>
              </div>

              <h2 className="text-xl font-bold text-surface-900 mb-2">
                {t('جرّب مسار الآن', 'Try Masaar Demo')}
              </h2>
              <p className="text-sm text-surface-500 leading-relaxed mb-6">
                {t(
                  'استعرض كامل مميزات النظام ببيانات تجريبية. لا تحتاج إلى إنشاء حساب.',
                  'Explore all features with sample data. No sign-up required.'
                )}
              </p>

              <button
                type="button"
                onClick={handleDemoLogin}
                disabled={loading}
                className="w-full py-3 bg-gradient-to-r from-gold-500 to-gold-600 text-white font-semibold rounded-xl text-sm shadow-card shadow-gold-200 hover:shadow-lg hover:shadow-gold-200 hover:from-gold-600 hover:to-gold-700 active:scale-[0.98] transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed disabled:active:scale-100"
              >
                {loading
                  ? t('جاري التحميل...', 'Loading...')
                  : t('ابدأ التجربة', 'Launch Demo')}
              </button>

              <div className="mt-4 flex items-center justify-center gap-4 text-[11px] text-surface-400">
                <span className="flex items-center gap-1">
                  <svg className="w-3 h-3 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {t('بيانات تجريبية', 'Sample data')}
                </span>
                <span className="flex items-center gap-1">
                  <svg className="w-3 h-3 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {t('بدون تسجيل', 'No registration')}
                </span>
                <span className="flex items-center gap-1">
                  <svg className="w-3 h-3 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {t('قراءة فقط', 'Read-only')}
                </span>
              </div>
            </div>

            <div className="mt-5 text-center space-y-3">
              <button
                type="button"
                onClick={() => setShowLogin(true)}
                className="text-sm font-medium text-primary-600 hover:text-primary-700 transition-colors"
              >
                {t('لديك حساب؟ سجل دخولك', 'Already have an account? Sign in')}
              </button>
              <div>
                <span className="text-xs text-surface-400">
                  {t('أو', 'or')}{' '}
                </span>
                <a href="/signup" className="text-xs font-medium text-primary-600 hover:text-primary-700 transition-colors">
                  {t('أنشئ حساباً جديداً', 'Create a new account')}
                </a>
              </div>
            </div>
          </>
        ) : (
          <>
            {/* ────── AUTH FORM ────── */}

            {/* Magic link sent state */}
            {magicSent ? (
              <div className="bg-white rounded-2xl shadow-card border border-surface-200/70 p-8 space-y-4 text-center animate-scale-in">
                <div className="mx-auto w-12 h-12 rounded-xl bg-gradient-to-br from-primary-50 to-primary-100 text-primary-600 flex items-center justify-center mb-1 ring-1 ring-primary-200/50">
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                  </svg>
                </div>
                <h3 className="text-lg font-semibold text-surface-900">
                  {t('تم إرسال رابط الدخول', 'Magic link sent!')}
                </h3>
                <p className="text-sm text-surface-500">
                  {t('تحقق من بريدك الإلكتروني وانقر على الرابط لتسجيل الدخول', 'Check your email and click the link to sign in')}
                </p>
                <button onClick={() => resetMode('password')} className="text-sm font-medium text-primary-600 hover:text-primary-700 transition-colors">
                  {t('العودة لتسجيل الدخول', 'Back to sign in')}
                </button>
              </div>

            /* OTP entry state */
            ) : otpSent ? (
              <form onSubmit={handleSMSVerify} className="relative bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4 animate-scale-in">
                <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-primary-500 to-primary-400 rounded-t-2xl" />
                <div className="text-center pb-1">
                  <div className="mx-auto w-12 h-12 rounded-xl bg-gradient-to-br from-primary-50 to-primary-100 text-primary-600 flex items-center justify-center mb-3 ring-1 ring-primary-200/50">
                    <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <h3 className="text-base font-semibold text-surface-900">
                    {t('أدخل رمز التحقق', 'Enter your OTP')}
                  </h3>
                  <p className="text-xs text-surface-500 mt-1">
                    {t(`تم الإرسال إلى ${phone}`, `Sent to ${phone}`)}
                  </p>
                </div>

                <div>
                  <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                    {t('رمز التحقق', 'One-time code')}
                  </label>
                  <input
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]{6}"
                    maxLength={6}
                    value={otp}
                    onChange={(e) => setOtp(e.target.value.replace(/\D/g, ''))}
                    required
                    autoFocus
                    className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm text-center tracking-[0.3em] font-mono bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-all duration-200"
                    placeholder="000000"
                  />
                </div>

                {error && (
                  <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg animate-slide-up">{error}</p>
                )}

                <button
                  type="submit"
                  disabled={loading || otp.length !== 6}
                  className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover active:scale-[0.98] transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed disabled:active:scale-100"
                >
                  {loading ? t('جاري التحقق...', 'Verifying...') : t('تحقق وادخل', 'Verify & sign in')}
                </button>

                <div className="text-center">
                  <button type="button" onClick={() => { setOtpSent(false); setOtp(''); setError('') }}
                    className="text-xs font-medium text-primary-600 hover:text-primary-700 transition-colors">
                    {t('تغيير رقم الهاتف', 'Change phone number')}
                  </button>
                </div>
              </form>

            /* Magic link form */
            ) : mode === 'magic-link' ? (
              <form
                onSubmit={handleMagicLinkRequest}
                className="relative bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4 animate-scale-in"
              >
                <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-primary-500 to-primary-400 rounded-t-2xl" />

                <div className="text-center pb-1">
                  <div className="mx-auto w-10 h-10 rounded-xl bg-gradient-to-br from-primary-50 to-primary-100 text-primary-600 flex items-center justify-center mb-2 ring-1 ring-primary-200/50">
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <h3 className="text-base font-semibold text-surface-900">
                    {t('تسجيل الدخول برابط البريد', 'Magic link sign in')}
                  </h3>
                  <p className="text-xs text-surface-500 mt-1">
                    {t('أدخل بريدك الإلكتروني وسنرسل لك رابطاً للدخول', 'Enter your email and we\'ll send you a sign-in link')}
                  </p>
                </div>

                <div>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    autoComplete="email"
                    autoFocus
                    className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-all duration-200"
                    placeholder={t('أدخل بريدك الإلكتروني', 'Enter your email')}
                  />
                </div>

                {error && (
                  <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg animate-slide-up">{error}</p>
                )}

                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover active:scale-[0.98] transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed disabled:active:scale-100"
                >
                  {loading ? t('جاري الإرسال...', 'Sending...') : t('إرسال رابط الدخول', 'Send magic link')}
                </button>

                <div className="text-center">
                  <button type="button" onClick={() => resetMode('password')}
                    className="text-xs font-medium text-surface-500 hover:text-primary-600 transition-colors">
                    {t('العودة لتسجيل الدخول', 'Back to sign in')}
                  </button>
                </div>
              </form>

            /* SMS form */
            ) : mode === 'sms' ? (
              <form
                onSubmit={handleSMSRequest}
                className="relative bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4 animate-scale-in"
              >
                <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-primary-500 to-primary-400 rounded-t-2xl" />

                <div className="text-center pb-1">
                  <div className="mx-auto w-10 h-10 rounded-xl bg-gradient-to-br from-primary-50 to-primary-100 text-primary-600 flex items-center justify-center mb-2 ring-1 ring-primary-200/50">
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <h3 className="text-base font-semibold text-surface-900">
                    {t('تسجيل الدخول برقم الهاتف', 'Phone sign in')}
                  </h3>
                  <p className="text-xs text-surface-500 mt-1">
                    {t('أدخل رقم هاتفك وسنرسل لك رمز تحقق', 'Enter your phone and we\'ll send a verification code')}
                  </p>
                </div>

                <div>
                  <input
                    type="tel"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    required
                    autoComplete="tel"
                    autoFocus
                    className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-all duration-200"
                    placeholder="+971 50 123 4567"
                  />
                  <p className="mt-1 text-[11px] text-surface-400">
                    {t('أدخل الرقم بصيغة دولية مثل +971501234567', 'Include country code, e.g. +971501234567')}
                  </p>
                </div>

                {TURNSTILE_SITE_KEY && (
                  <div className="flex justify-center">
                    <Turnstile
                      ref={turnstileRef}
                      siteKey={TURNSTILE_SITE_KEY}
                      onSuccess={setTurnstileToken}
                      onExpire={() => setTurnstileToken(null)}
                      onError={() => setTurnstileToken(null)}
                      options={{ theme: 'light', language: lang === 'ar' ? 'ar' : 'en' }}
                    />
                  </div>
                )}

                {error && (
                  <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg animate-slide-up">{error}</p>
                )}

                <button
                  type="submit"
                  disabled={loading || (!!TURNSTILE_SITE_KEY && !turnstileToken)}
                  className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover active:scale-[0.98] transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed disabled:active:scale-100"
                >
                  {loading ? t('جاري الإرسال...', 'Sending...') : t('إرسال رمز التحقق', 'Send OTP')}
                </button>

                <div className="text-center">
                  <button type="button" onClick={() => resetMode('password')}
                    className="text-xs font-medium text-surface-500 hover:text-primary-600 transition-colors">
                    {t('العودة لتسجيل الدخول', 'Back to sign in')}
                  </button>
                </div>
              </form>

            /* Password login form */
            ) : (
              <form
                onSubmit={handlePasswordLogin}
                className="relative bg-white rounded-2xl shadow-card border border-surface-200/70 p-7 space-y-4 animate-scale-in"
              >
                <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-primary-500 to-primary-400 rounded-t-2xl" />

                <div>
                  <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                    {t('البريد الإلكتروني', 'Email')}
                  </label>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    autoComplete="email"
                    autoFocus
                    className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-all duration-200"
                    placeholder={t('أدخل بريدك الإلكتروني', 'Enter your email')}
                  />
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label className="block text-[13px] font-medium text-surface-700">
                      {t('كلمة المرور', 'Password')}
                    </label>
                    <a href="/forgot-password" className="text-[12px] text-primary-600 hover:text-primary-700 font-medium transition-colors">
                      {t('نسيت كلمة المرور؟', 'Forgot password?')}
                    </a>
                  </div>
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    autoComplete="current-password"
                    className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-all duration-200"
                    placeholder="••••••••"
                  />
                </div>

                {TURNSTILE_SITE_KEY && (
                  <div className="flex justify-center">
                    <Turnstile
                      ref={turnstileRef}
                      siteKey={TURNSTILE_SITE_KEY}
                      onSuccess={setTurnstileToken}
                      onExpire={() => setTurnstileToken(null)}
                      onError={() => setTurnstileToken(null)}
                      options={{ theme: 'light', language: lang === 'ar' ? 'ar' : 'en' }}
                    />
                  </div>
                )}

                {error && (
                  <p className="text-red-600 text-xs bg-red-50 border border-red-100 px-3 py-2 rounded-lg animate-slide-up">{error}</p>
                )}

                <button
                  type="submit"
                  disabled={loading || (!!TURNSTILE_SITE_KEY && !turnstileToken)}
                  className="w-full py-2.5 bg-primary-600 text-white font-medium rounded-xl text-sm shadow-card hover:bg-primary-700 hover:shadow-card-hover active:scale-[0.98] transition-all duration-200 ease-soft disabled:opacity-60 disabled:cursor-not-allowed disabled:active:scale-100"
                >
                  {loading
                    ? t('جاري المعالجة...', 'Processing...')
                    : t('تسجيل الدخول', 'Sign in')}
                </button>

                <div className="flex items-center gap-3 pt-1">
                  <div className="flex-1 h-px bg-surface-200" />
                  <span className="text-[11px] font-medium text-surface-400 uppercase tracking-wider">
                    {t('أو', 'or')}
                  </span>
                  <div className="flex-1 h-px bg-surface-200" />
                </div>

                <div className="flex justify-center gap-4">
                  <button type="button" onClick={() => resetMode('magic-link')}
                    className="text-xs font-medium text-surface-500 hover:text-primary-600 transition-colors">
                    {t('رابط البريد', 'Magic link')}
                  </button>
                  <span className="text-surface-300 text-xs">·</span>
                  <button type="button" onClick={() => resetMode('sms')}
                    className="text-xs font-medium text-surface-500 hover:text-primary-600 transition-colors">
                    {t('رسالة نصية', 'SMS')}
                  </button>
                </div>
              </form>
            )}

            <div className="mt-4 text-center space-y-3">
              <p className="text-xs text-surface-500">
                {t('ليس لديك حساب؟', "Don't have an account?")}{' '}
                <a href="/signup" className="text-primary-600 hover:text-primary-700 font-medium">
                  {t('إنشاء حساب', 'Sign up')}
                </a>
              </p>
              <button
                type="button"
                onClick={() => { setShowLogin(false); resetMode('password') }}
                className="text-xs font-medium text-surface-400 hover:text-primary-600 transition-colors"
              >
                {t('العودة للصفحة الرئيسية', 'Back to home')}
              </button>
            </div>
          </>
        )}

        {/* Footer */}
        <div className="mt-6 flex items-center justify-center gap-3 text-[11px] text-surface-400">
          <button
            onClick={() => setLang(lang === 'ar' ? 'en' : 'ar')}
            className="hover:text-surface-900 transition-colors"
          >
            {lang === 'ar' ? 'English' : 'العربية'}
          </button>
          <span className="text-surface-300">·</span>
          <span>
            {t('مصنوع للسوق الإماراتي', 'Built for UAE')}
          </span>
        </div>
      </div>
    </div>
  )
}
