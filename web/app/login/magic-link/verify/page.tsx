'use client'
import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

function MagicLinkVerifyContent() {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [error, setError] = useState('')
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setSession } = useAuthStore()
  const { setLang, t } = useLang()

  useEffect(() => {
    const token = searchParams.get('token')
    if (!token) {
      setStatus('error')
      setError(t('رابط غير صالح', 'Invalid or missing token'))
      return
    }

    const verifyToken = async () => {
      try {
        const res = await api.auth.verifyMagicLink(token) as any
        setSession(res.access_token, res.refresh_token, res.user, res.company)
        if (res.user.lang_pref) setLang(res.user.lang_pref)
        setStatus('success')
        setTimeout(() => {
          router.push('/pipeline')
        }, 1500)
      } catch (err: any) {
        setStatus('error')
        setError(err.message || t('حدث خطأ', 'Something went wrong'))
      }
    }

    verifyToken()
  }, [searchParams, setSession, setLang, router])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-sm bg-white rounded-2xl shadow-sm border border-gray-100 p-8 text-center">
        {status === 'loading' && (
          <>
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-600 mx-auto mb-4"></div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              {t('جاري التحقق...', 'Verifying...')}
            </h3>
            <p className="text-sm text-gray-600">
              {t('الرجاء الانتظار', 'Please wait')}
            </p>
          </>
        )}

        {status === 'success' && (
          <>
            <div className="text-green-500 text-4xl mb-4">✓</div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              {t('تم تسجيل الدخول بنجاح', 'Successfully signed in!')}
            </h3>
            <p className="text-sm text-gray-600">
              {t('جاري تحويلك...', 'Redirecting you...')}
            </p>
          </>
        )}

        {status === 'error' && (
          <>
            <div className="text-red-500 text-4xl mb-4">✗</div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              {t('فشل تسجيل الدخول', 'Login failed')}
            </h3>
            <p className="text-sm text-red-600 mb-4">{error}</p>
            <button
              onClick={() => router.push('/login')}
              className="text-sm text-brand-600 hover:text-brand-700"
            >
              {t('العودة لتسجيل الدخول', 'Back to sign in')}
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export default function MagicLinkVerifyPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-brand-600"></div>
      </div>
    }>
      <MagicLinkVerifyContent />
    </Suspense>
  )
}
