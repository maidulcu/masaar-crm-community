'use client'

import { useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Turnstile } from '@marsidev/react-turnstile'
import type { TurnstileInstance } from '@marsidev/react-turnstile'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'
import type { LoginResponse } from '@/types'

const TURNSTILE_SITE_KEY = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY ?? ''

async function verifyTurnstile(token: string): Promise<boolean> {
  const res = await fetch('/api/verify-turnstile', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token }),
  })
  const data = await res.json()
  return data.success === true
}

export default function SignupPage() {
  const { setSession } = useAuthStore()
  const router = useRouter()
  const turnstileRef = useRef<TurnstileInstance>(null)

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [companyName, setCompanyName] = useState('')
  const [subdomain, setSubdomain] = useState('')
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!name || !email || !password || !companyName || !subdomain) {
      setError('All fields are required')
      return
    }

    // Verify Turnstile before hitting the backend (only when configured)
    if (TURNSTILE_SITE_KEY) {
      if (!turnstileToken) {
        setError('Please complete the security check')
        return
      }
      const ok = await verifyTurnstile(turnstileToken)
      if (!ok) {
        setError('Security check failed — please try again')
        turnstileRef.current?.reset()
        setTurnstileToken(null)
        return
      }
    }

    setLoading(true)
    try {
      const res = await api.auth.register({
        name,
        email,
        password,
        company_name: companyName,
        subdomain,
        turnstile_token: turnstileToken ?? '',
      }) as LoginResponse
      setSession(res.access_token, res.refresh_token, res.user, res.company)
      router.push('/onboarding')
    } catch (err: any) {
      setError(err.message || 'Registration failed')
      turnstileRef.current?.reset()
      setTurnstileToken(null)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-blue-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Masaar CRM</h1>
          <p className="text-gray-500 mt-1">{'Create your account'}</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-white rounded-2xl shadow-xl border border-gray-100 p-8 space-y-5">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-4 py-3">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">{'Company Name'}</label>
            <input
              type="text"
              value={companyName}
              onChange={e => setCompanyName(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
              placeholder="e.g. Dubai Properties LLC"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">{'Subdomain'}</label>
            <div className="flex rounded-lg border border-gray-300 focus-within:ring-2 focus-within:ring-blue-500 focus-within:border-transparent overflow-hidden">
              <input
                type="text"
                value={subdomain}
                onChange={e => setSubdomain(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                className="flex-1 px-4 py-2.5 text-sm outline-none"
                placeholder="your-company"
                required
              />
              <span className="bg-gray-50 text-gray-400 text-sm px-3 py-2.5 border-l border-gray-200 whitespace-nowrap">
                .masaar.app
              </span>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">{'Full Name'}</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
              placeholder="Ahmed Al Maktoum"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">{'Email'}</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
              placeholder="ahmed@example.com"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">{'Password'}</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
              placeholder="Min. 8 characters, 1 uppercase, 1 number"
              minLength={8}
              required
            />
            {password && password.length < 8 && (
              <p className="text-xs text-red-500 mt-1">Password must be at least 8 characters</p>
            )}
          </div>

          {TURNSTILE_SITE_KEY && (
            <div className="flex justify-center">
              <Turnstile
                ref={turnstileRef}
                siteKey={TURNSTILE_SITE_KEY}
                onSuccess={setTurnstileToken}
                onExpire={() => setTurnstileToken(null)}
                onError={() => setTurnstileToken(null)}
              />
            </div>
          )}

          <button
            type="submit"
            disabled={loading || (!!TURNSTILE_SITE_KEY && !turnstileToken)}
            className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white font-medium rounded-lg py-2.5 text-sm transition"
          >
            {loading ? 'Creating account…' : 'Create account'}
          </button>

          <p className="text-center text-sm text-gray-500">
            {'Already have an account?'}{' '}
            <Link href="/login" className="text-blue-600 hover:underline font-medium">
              {'Sign in'}
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}
