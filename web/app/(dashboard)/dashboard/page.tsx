'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface Stats {
  total_contacts: number
  active_leads: number
  new_leads_week: number
  open_threads: number
  open_deals: number
  open_deals_value: number
  won_deals: number
  won_deals_value: number
}

function StatCard({
  label,
  value,
  sub,
  href,
  color,
}: {
  label: string
  value: string | number
  sub?: string
  href: string
  color: string
}) {
  return (
    <Link href={href} className="block group">
      <div className="bg-white rounded-2xl border border-surface-200/70 p-5 shadow-card hover:shadow-card-hover hover:-translate-y-0.5 hover:border-primary-200 transition-all duration-200 ease-soft">
        <p className="text-[11px] font-medium uppercase tracking-wide text-surface-500 mb-2">{label}</p>
        <p className={`text-3xl font-bold tabular-nums leading-none ${color}`}>{value}</p>
        {sub && <p className="text-xs text-surface-500 mt-2.5">{sub}</p>}
      </div>
    </Link>
  )
}

function SkeletonCard() {
  return (
    <div className="bg-white rounded-2xl border border-surface-200/70 p-5 shadow-card animate-pulse">
      <div className="h-3 w-24 bg-surface-100 rounded mb-3" />
      <div className="h-7 w-20 bg-surface-100 rounded" />
    </div>
  )
}

export default function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(true)
  const { t } = useLang()

  useEffect(() => {
    api.stats.overview()
      .then((data) => setStats(data as Stats))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const fmt = (n: number) =>
    new Intl.NumberFormat('en-AE', { maximumFractionDigits: 0 }).format(n)

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('لوحة التحكم', 'Dashboard')} />

      <main className="flex-1 overflow-y-auto px-6 py-8 bg-surface-50">
        <div className="max-w-5xl space-y-8">

          {/* Top row */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            {loading ? (
              Array.from({ length: 4 }).map((_, i) => <SkeletonCard key={i} />)
            ) : (
              <>
                <StatCard
                  label={t('إجمالي جهات الاتصال', 'Total Contacts')}
                  value={fmt(stats?.total_contacts ?? 0)}
                  href="/contacts"
                  color="text-surface-900"
                />
                <StatCard
                  label={t('Leads النشطة', 'Active Leads')}
                  value={fmt(stats?.active_leads ?? 0)}
                  sub={t(`${stats?.new_leads_week ?? 0} هذا الأسبوع`, `${stats?.new_leads_week ?? 0} this week`)}
                  href="/pipeline"
                  color="text-primary-600"
                />
                <StatCard
                  label={t('محادثات مفتوحة', 'Open Threads')}
                  value={fmt(stats?.open_threads ?? 0)}
                  href="/inbox"
                  color="text-sky-600"
                />
                <StatCard
                  label={t('صفقات مفتوحة', 'Open Deals')}
                  value={fmt(stats?.open_deals ?? 0)}
                  sub={stats?.open_deals_value ? `AED ${fmt(stats.open_deals_value)}` : undefined}
                  href="/deals"
                  color="text-gold-600"
                />
              </>
            )}
          </div>

          {/* Won deals highlight */}
          {!loading && (stats?.won_deals ?? 0) > 0 && (
            <Link href="/deals" className="block group">
              <div className="relative overflow-hidden rounded-2xl border border-emerald-100 bg-gradient-to-br from-emerald-50 to-emerald-50/30 p-6 flex items-center justify-between gap-4 shadow-card hover:shadow-card-hover transition-all duration-200 ease-soft">
                <div>
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-emerald-600 mb-1.5">
                    {t('الصفقات المكتسبة', 'Won Deals')}
                  </p>
                  <p className="text-3xl font-bold text-emerald-700 tabular-nums leading-none">{fmt(stats?.won_deals ?? 0)}</p>
                </div>
                <div className="text-end">
                  <p className="text-[11px] text-emerald-600 mb-1.5">{t('إجمالي القيمة', 'Total Value')}</p>
                  <p className="text-xl font-bold text-emerald-700 tabular-nums">AED {fmt(stats?.won_deals_value ?? 0)}</p>
                </div>
                <div className="w-11 h-11 rounded-2xl bg-emerald-100 text-emerald-600 flex items-center justify-center ms-2 shrink-0">
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
              </div>
            </Link>
          )}

          {/* Quick links */}
          <div>
            <p className="text-[11px] font-semibold text-surface-500 uppercase tracking-[0.12em] mb-3">
              {t('روابط سريعة', 'Quick Links')}
            </p>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {[
                { href: '/pipeline', label: { en: 'Pipeline', ar: 'خط الأنابيب' }, icon: '📊' },
                { href: '/inbox',    label: { en: 'Inbox',    ar: 'الرسائل'     }, icon: '💬' },
                { href: '/contacts', label: { en: 'Contacts', ar: 'جهات الاتصال' }, icon: '👤' },
                { href: '/deals',    label: { en: 'Deals',    ar: 'الصفقات'     }, icon: '🤝' },
              ].map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="group flex items-center gap-2.5 bg-white border border-surface-200/70 rounded-xl px-4 py-3 text-sm font-medium text-surface-700 shadow-card hover:shadow-card-hover hover:border-primary-200 hover:text-primary-700 hover:-translate-y-0.5 transition-all duration-200 ease-soft"
                >
                  <span className="text-base">{item.icon}</span>
                  <span>{t(item.label.ar, item.label.en)}</span>
                </Link>
              ))}
            </div>
          </div>

        </div>
      </main>
    </div>
  )
}
