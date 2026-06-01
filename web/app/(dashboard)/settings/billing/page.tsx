'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { Company } from '@/types'

interface Quota {
  used: number
  limit: number
  available: boolean
  unlimited?: boolean
  pct?: number
}

interface PlanOption {
  id: string
  name: string
  price: number
  current: boolean
  features: string[]
  quotas: {
    bos24_monthly: number
    ai_monthly: number
    pdf_monthly: number
  }
}

interface BillingData {
  plan: {
    id: string
    name: string
    price: number
    started_at: string | null
    expires_at: string | null
    stripe_enabled: boolean
    has_sub: boolean
  }
  usage: {
    bos24: Quota
    ai: Quota
    pdf: Quota
    reset: string
  }
  plans: PlanOption[]
}

const PLAN_COLORS: Record<string, string> = {
  community: 'border-gray-200 bg-white',
  starter:   'border-blue-200 bg-blue-50',
  pro:       'border-purple-200 bg-purple-50',
  business:  'border-amber-200 bg-amber-50',
}

const PLAN_BADGE: Record<string, string> = {
  community: 'bg-gray-100 text-gray-600',
  starter:   'bg-blue-100 text-blue-700',
  pro:       'bg-purple-100 text-purple-700',
  business:  'bg-amber-100 text-amber-700',
}

const QUOTA_LABEL: Record<string, string> = {
  bos24: 'Market data calls',
  ai:    'AI requests',
  pdf:   'PDF reports',
}

function QuotaMeter({ label, quota }: { label: string; quota: Quota }) {
  if (!quota.available) {
    return (
      <div className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
        <span className="text-sm text-gray-500">{label}</span>
        <span className="text-xs text-gray-400">Not included</span>
      </div>
    )
  }
  if (quota.unlimited) {
    return (
      <div className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
        <span className="text-sm text-gray-600">{label}</span>
        <span className="text-xs font-medium text-green-600">{quota.used} used · Unlimited</span>
      </div>
    )
  }
  const pct = quota.limit > 0 ? Math.min(100, Math.round((quota.used / quota.limit) * 100)) : 0
  const barColor = pct >= 90 ? 'bg-red-500' : pct >= 70 ? 'bg-amber-500' : 'bg-blue-500'

  return (
    <div className="py-2 border-b border-gray-100 last:border-0">
      <div className="flex items-center justify-between mb-1">
        <span className="text-sm text-gray-600">{label}</span>
        <span className="text-xs text-gray-500">{quota.used} / {quota.limit}</span>
      </div>
      <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
        <div className={`h-full rounded-full transition-all ${barColor}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

function formatQuota(n: number) {
  if (n === -1) return 'Unlimited'
  if (n === 0) return 'Not included'
  return n.toLocaleString()
}

export default function BillingPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()

  const [data, setData] = useState<BillingData | null>(null)
  const [loading, setLoading] = useState(true)
  const [checkingOut, setCheckingOut] = useState<string | null>(null)
  const [openingPortal, setOpeningPortal] = useState(false)
  const [error, setError] = useState('')

  const isAdmin = user?.role === 'admin'
  const isDemo = !!company?.is_demo

  useEffect(() => {
    if (user && !isAdmin) {
      // Viewers and agents can see usage but not change plan
    }
    api.billing.get().then((r: any) => {
      setData(r.data)
      setLoading(false)
    }).catch(() => {
      setError('Failed to load billing information')
      setLoading(false)
    })
  }, [user])

  // Handle Stripe return redirects
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('success')) {
      setError('')
      // Reload data after successful payment
      setTimeout(() => api.billing.get().then((r: any) => setData(r.data)), 2000)
    }
  }, [])

  async function handleUpgrade(planID: string) {
    if (!isAdmin) return
    setCheckingOut(planID)
    setError('')
    try {
      const res = await api.billing.checkout(planID) as { data: { checkout_url: string } }
      window.location.href = res.data.checkout_url
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to start checkout')
      setCheckingOut(null)
    }
  }

  async function handlePortal() {
    if (!isAdmin) return
    setOpeningPortal(true)
    setError('')
    try {
      const res = await api.billing.portal() as { data: { portal_url: string } }
      window.location.href = res.data.portal_url
    } catch (e: any) {
      setError(e.response?.data?.error || 'Failed to open billing portal')
      setOpeningPortal(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (!data) {
    return <div className="p-8 text-red-600">{error || 'Failed to load billing data'}</div>
  }

  const currentPlan = data.plans.find(p => p.id === data.plan.id) || data.plans[0]

  return (
    <div className="max-w-5xl mx-auto px-4 py-8 space-y-8">

      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Plans & Billing</h1>
        <p className="text-gray-500 mt-1">Manage your Masaar CRM subscription and view usage.</p>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Demo notice */}
      {isDemo && (
        <div className="flex items-center gap-2.5 p-4 bg-amber-50 border border-amber-200 rounded-xl text-sm text-amber-800">
          <svg className="w-5 h-5 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
          </svg>
          <span>
            <strong>Demo account</strong> — billing actions are disabled.{' '}
            <a href="/signup" className="underline font-semibold">Start a free trial</a> to manage your subscription.
          </span>
        </div>
      )}

      {/* Trial banner */}
      {company?.on_trial && company.days_remaining > 0 && (
        <div className={`rounded-xl border px-5 py-4 text-sm ${
          company.days_remaining <= 7
            ? 'bg-red-50 border-red-200 text-red-800'
            : company.days_remaining <= 30
              ? 'bg-yellow-50 border-yellow-200 text-yellow-800'
              : 'bg-green-50 border-green-200 text-green-800'
        }`}>
          <div className="flex items-center justify-between">
            <span>
              <strong>Trial:</strong> {company.days_remaining} day{company.days_remaining !== 1 ? 's' : ''} remaining on <strong>{company.plan}</strong> plan
            </span>
            {company.days_remaining <= 7 && (
              <span className="font-semibold">Upgrade now →</span>
            )}
          </div>
        </div>
      )}

      {/* Current plan + usage */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

        {/* Current plan card */}
        <div className={`rounded-xl border-2 p-6 ${PLAN_COLORS[data.plan.id] || PLAN_COLORS.community}`}>
          <div className="flex items-center justify-between mb-4">
            <div>
              <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-1">Current plan</div>
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-bold text-gray-900">{data.plan.name}</h2>
                <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${PLAN_BADGE[data.plan.id]}`}>
                  Active
                </span>
              </div>
            </div>
            {data.plan.price > 0 && (
              <div className="text-right">
                <div className="text-2xl font-bold text-gray-900">${data.plan.price}</div>
                <div className="text-xs text-gray-500">/month</div>
              </div>
            )}
          </div>

          {data.plan.expires_at && (
            <p className="text-xs text-gray-500 mb-4">
              Renews {new Date(data.plan.expires_at).toLocaleDateString('en-AE', { day: 'numeric', month: 'long', year: 'numeric' })}
            </p>
          )}

          <ul className="space-y-1.5 mb-6">
            {currentPlan.features.map(f => (
              <li key={f} className="flex items-start gap-2 text-sm text-gray-700">
                <svg className="w-4 h-4 text-green-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/>
                </svg>
                {f}
              </li>
            ))}
          </ul>

          {isAdmin && !isDemo && data.plan.has_sub && data.plan.stripe_enabled && (
            <button
              onClick={handlePortal}
              disabled={openingPortal}
              className="w-full text-sm font-medium text-gray-600 border border-gray-300 rounded-lg py-2 hover:bg-gray-50 transition disabled:opacity-50"
            >
              {openingPortal ? 'Opening portal…' : 'Manage billing & invoices →'}
            </button>
          )}
        </div>

        {/* Usage card */}
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-900">This month's usage</h3>
            <span className="text-xs text-gray-400">Resets {data.usage.reset}</span>
          </div>
          <div className="space-y-1">
            <QuotaMeter label={QUOTA_LABEL.bos24} quota={data.usage.bos24} />
            <QuotaMeter label={QUOTA_LABEL.ai}    quota={data.usage.ai}    />
            <QuotaMeter label={QUOTA_LABEL.pdf}   quota={data.usage.pdf}   />
          </div>
          {!data.plan.stripe_enabled && data.plan.id === 'community' && (
            <p className="mt-4 text-xs text-gray-400">
              Self-hosted — upgrade requires Stripe to be configured by your administrator.
            </p>
          )}
        </div>
      </div>

      {/* Plan comparison */}
      <div>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Compare plans</h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {data.plans.map(plan => (
            <div
              key={plan.id}
              className={`rounded-xl border-2 p-5 flex flex-col ${
                plan.current
                  ? `${PLAN_COLORS[plan.id]} ring-2 ring-offset-1 ring-blue-400`
                  : 'border-gray-200 bg-white hover:border-gray-300 transition'
              }`}
            >
              <div className="flex items-center justify-between mb-3">
                <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${PLAN_BADGE[plan.id]}`}>
                  {plan.name}
                </span>
                {plan.current && (
                  <span className="text-xs text-green-600 font-medium">Current</span>
                )}
              </div>

              <div className="mb-4">
                {plan.price === 0 ? (
                  <span className="text-2xl font-bold text-gray-900">Free</span>
                ) : (
                  <>
                    <span className="text-2xl font-bold text-gray-900">${plan.price}</span>
                    <span className="text-sm text-gray-500">/mo</span>
                  </>
                )}
              </div>

              {/* Quota summary */}
              <div className="text-xs text-gray-500 space-y-1 mb-4 flex-1">
                <div className="flex justify-between">
                  <span>Market data</span>
                  <span className="font-medium text-gray-700">{formatQuota(plan.quotas.bos24_monthly)}</span>
                </div>
                <div className="flex justify-between">
                  <span>AI requests</span>
                  <span className="font-medium text-gray-700">{formatQuota(plan.quotas.ai_monthly)}</span>
                </div>
                <div className="flex justify-between">
                  <span>PDF reports</span>
                  <span className="font-medium text-gray-700">{formatQuota(plan.quotas.pdf_monthly)}</span>
                </div>
              </div>

              <ul className="text-xs text-gray-600 space-y-1 mb-5 flex-1">
                {plan.features.slice(0, 4).map(f => (
                  <li key={f} className="flex items-start gap-1.5">
                    <svg className="w-3 h-3 text-green-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/>
                    </svg>
                    {f}
                  </li>
                ))}
                {plan.features.length > 4 && (
                  <li className="text-gray-400">+{plan.features.length - 4} more</li>
                )}
              </ul>

              {isAdmin && !isDemo && !plan.current && plan.price > 0 && (
                data.plan.stripe_enabled ? (
                  <button
                    onClick={() => handleUpgrade(plan.id)}
                    disabled={checkingOut === plan.id}
                    className="w-full text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-lg py-2 transition disabled:opacity-50"
                  >
                    {checkingOut === plan.id ? 'Redirecting…' : `Upgrade to ${plan.name}`}
                  </button>
                ) : (
                  <div className="text-xs text-center text-gray-400 border border-dashed border-gray-300 rounded-lg py-2">
                    Configure Stripe to upgrade
                  </div>
                )
              )}

              {plan.current && (
                <div className="text-xs text-center text-gray-400 py-1">Your current plan</div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Self-hosted note */}
      <p className="text-xs text-gray-400 text-center">
        Masaar CRM is open-source and self-hosted. Plans unlock cloud features (BOS24 market data, AI, PDFs)
        provided by Masaar Pro services. Community plan is always free with no limits on your own data.
      </p>
    </div>
  )
}
