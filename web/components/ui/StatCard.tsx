'use client'

import Link from 'next/link'
import type { ReactNode } from 'react'
import clsx from 'clsx'
import { useLang } from '@/context/LangContext'

/**
 * Visual emphasis for the card's value.
 *  - "default" : neutral (gray)
 *  - "primary" : brand blue        (for actionable KPIs)
 *  - "gold"    : UAE gold / amber  (for premium / high-value figures)
 *  - "success" : green             (for won deals, positive deltas)
 *  - "warning" : amber-light       (for stale threads, attention)
 */
export type StatTone = 'default' | 'primary' | 'gold' | 'success' | 'warning'

export interface StatCardProps {
  /** Short, translated label shown above the value. */
  label: string
  /** The headline figure (already formatted by the caller). */
  value: string | number
  /** Optional supporting line (e.g. "12 this week", "AED 42,000"). */
  sub?: string
  /** Optional leading icon (20x20 recommended). */
  icon?: ReactNode
  /** If provided, the card becomes a clickable drill-down link. */
  href?: string
  /** Visual emphasis for the headline value. */
  tone?: StatTone
  /** Bilingual aria-label — falls back to `${label} ${value}` if omitted. */
  ariaLabel?: { ar: string; en: string }
  /** Extra classes appended to the outer container. */
  className?: string
}

const toneToValueClass: Record<StatTone, string> = {
  default: 'text-gray-900',
  primary: 'text-primary-600',
  gold:    'text-gold-600',
  success: 'text-emerald-600',
  warning: 'text-gold-500',
}

const toneToIconBg: Record<StatTone, string> = {
  default: 'bg-gray-100 text-gray-600',
  primary: 'bg-primary-50 text-primary-600',
  gold:    'bg-gold-50 text-gold-600',
  success: 'bg-emerald-50 text-emerald-600',
  warning: 'bg-gold-50 text-gold-500',
}

/**
 * StatCard — a single KPI tile used on the dashboard grid.
 *
 * Accessibility:
 *   - Renders as a native <a> when `href` is set, else a plain div.
 *   - Accepts a bilingual `ariaLabel` so screen readers announce the same
 *     semantics in Arabic and English.
 *   - All spacing uses logical properties (ms-*, me-*, ps-*, pe-*) so RTL
 *     mirrors automatically via `dir="rtl"` on <html>.
 */
export function StatCard({
  label,
  value,
  sub,
  icon,
  href,
  tone = 'default',
  ariaLabel,
  className,
}: StatCardProps) {
  const { t, lang } = useLang()

  const resolvedAriaLabel = ariaLabel
    ? (lang === 'ar' ? ariaLabel.ar : ariaLabel.en)
    : `${label}: ${value}${sub ? `, ${sub}` : ''}`

  const shell = (
    <div
      className={clsx(
        'group relative h-full rounded-2xl border border-gray-200 bg-white p-5',
        'shadow-card transition-all duration-200 ease-soft',
        href && 'hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-card-hover focus-within:border-primary-400',
        className,
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
          {label}
        </p>
        {icon && (
          <span
            aria-hidden="true"
            className={clsx(
              'inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl',
              toneToIconBg[tone],
            )}
          >
            {icon}
          </span>
        )}
      </div>

      <p className={clsx('mt-3 text-3xl font-bold leading-none tabular-nums', toneToValueClass[tone])}>
        {value}
      </p>

      {sub && (
        <p className="mt-2 text-xs text-gray-500">
          {sub}
        </p>
      )}

      {href && (
        <span
          aria-hidden="true"
          className="pointer-events-none absolute bottom-4 end-5 text-gray-300 transition-colors group-hover:text-primary-500 rtl:rotate-180"
        >
          {/* chevron — mirrored in RTL via rtl:rotate-180 */}
          <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
            <path
              fillRule="evenodd"
              d="M7.21 14.77a.75.75 0 010-1.06L10.94 10 7.21 6.29a.75.75 0 111.06-1.06l4.25 4.24a.75.75 0 010 1.06l-4.25 4.24a.75.75 0 01-1.06 0z"
              clipRule="evenodd"
            />
          </svg>
        </span>
      )}
    </div>
  )

  if (href) {
    return (
      <Link
        href={href}
        aria-label={resolvedAriaLabel}
        className="block h-full rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-gray-50"
      >
        {shell}
      </Link>
    )
  }

  return (
    <div role="group" aria-label={resolvedAriaLabel} className="h-full">
      {shell}
    </div>
  )
}

/**
 * Matching skeleton — used during api.stats.overview() loading.
 * Shape + padding match StatCard exactly so there is no layout shift.
 */
export function StatCardSkeleton() {
  return (
    <div
      aria-hidden="true"
      className="h-full rounded-2xl border border-gray-200 bg-white p-5 shadow-card"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="h-3 w-24 animate-pulse rounded bg-gray-100" />
        <div className="h-9 w-9 animate-pulse rounded-xl bg-gray-100" />
      </div>
      <div className="mt-4 h-7 w-20 animate-pulse rounded bg-gray-100" />
      <div className="mt-3 h-3 w-16 animate-pulse rounded bg-gray-100" />
    </div>
  )
}

export default StatCard
