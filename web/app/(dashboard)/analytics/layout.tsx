'use client'
import React from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useLang } from '@/context/LangContext'
import clsx from 'clsx'

const tabs = [
  { href: '/analytics', label: { en: 'Overview', ar: 'نظرة عامة' }, exact: true },
  { href: '/analytics/financial', label: { en: 'Financial', ar: 'المالية' } },
  { href: '/analytics/properties', label: { en: 'Properties', ar: 'العقارات' } },
  { href: '/analytics/tenants', label: { en: 'Tenants', ar: 'المستأجرون' } },
]

export default function AnalyticsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const { lang } = useLang()
  const ar = lang === 'ar'

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-6 pt-6 pb-0 border-b border-gray-200 bg-white">
        <h1 className="text-xl font-bold text-gray-900 mb-4">
          {ar ? 'التحليلات' : 'Analytics'}
        </h1>
        <nav className="flex gap-1">
          {tabs.map((tab) => {
            const active = tab.exact
              ? pathname === tab.href
              : pathname.startsWith(tab.href)
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className={clsx(
                  'px-4 py-2 text-sm font-medium rounded-t-lg border-b-2 transition-colors',
                  active
                    ? 'border-brand-600 text-brand-600 bg-brand-50'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                )}
              >
                {ar ? tab.label.ar : tab.label.en}
              </Link>
            )
          })}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {children}
      </div>
    </div>
  )
}
