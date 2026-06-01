'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import type { LoginResponse } from '@/types'

/**
 * /demo — instantly logs the visitor into the demo account.
 * Safe to link from ads, Product Hunt, emails, etc.
 * Skips Turnstile because the demo credentials are intentionally public.
 */
export default function DemoPage() {
  const router = useRouter()
  const { setSession } = useAuthStore()
  const { setLang } = useLang()
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false

    const login = async () => {
      try {
        const res = await api.auth.login('ahmed@masaar.local', 'Demo@1234') as LoginResponse
        if (cancelled) return
        setSession(res.access_token, res.refresh_token, res.user, res.company)
        if (res.user.lang_pref) setLang(res.user.lang_pref)
        router.replace('/pipeline')
      } catch (err: any) {
        if (!cancelled) setError(err.message || 'Demo login failed. Please try again.')
      }
    }

    login()
    return () => { cancelled = true }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-br from-surface-50 via-white to-surface-50 px-4">
      {!error ? (
        <div className="text-center space-y-4">
          {/* Spinner */}
          <div className="mx-auto w-12 h-12 rounded-2xl bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center shadow-lg shadow-primary-200 animate-pulse">
            <span className="text-white text-xl font-bold">M</span>
          </div>
          <p className="text-surface-600 text-sm font-medium">Loading demo…</p>
        </div>
      ) : (
        <div className="text-center space-y-4 max-w-xs">
          <div className="text-4xl">⚠️</div>
          <p className="text-surface-700 text-sm">{error}</p>
          <a
            href="/login"
            className="inline-block text-sm font-medium text-primary-600 hover:text-primary-700"
          >
            Back to sign in
          </a>
        </div>
      )}
    </div>
  )
}
