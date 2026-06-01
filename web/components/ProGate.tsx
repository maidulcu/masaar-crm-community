'use client'
import { useLang } from '@/context/LangContext'

interface ProGateProps {
  feature: string
  featureAr?: string
  children: React.ReactNode
}

export function ProGate({ feature, featureAr, children }: ProGateProps) {
  const isPro = process.env.NEXT_PUBLIC_EDITION === 'pro'
  const { lang } = useLang()

  if (isPro) return <>{children}</>

  const label = lang === 'ar' ? (featureAr ?? feature) : feature

  return (
    <div className="flex flex-col items-center justify-center min-h-[400px] text-center p-8">
      <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-amber-50 text-amber-500 mb-5 shadow-sm">
        <svg className="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
        </svg>
      </div>

      <h3 className="text-lg font-semibold text-surface-900 mb-2">
        {lang === 'ar' ? 'ميزة احترافية' : 'Pro Feature'}
      </h3>

      <p className="text-sm text-surface-500 mb-1 max-w-xs">
        {lang === 'ar'
          ? `${label} متاح في الخطة الاحترافية.`
          : `${label} is available on the Pro plan.`}
      </p>
      <p className="text-sm text-surface-400 mb-6 max-w-xs">
        {lang === 'ar'
          ? 'الخطة المجتمعية مجانية للأبد وتشمل: الواتساب، إدارة العملاء المحتملين، جهات الاتصال، والمزيد.'
          : 'The Community plan is free forever and includes WhatsApp pipeline, leads, contacts, and more.'}
      </p>

      <a
        href="https://masaar.io/pricing"
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-2 px-5 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 transition-colors"
      >
        {lang === 'ar' ? 'عرض الخطط' : 'View Plans'}
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
        </svg>
      </a>
    </div>
  )
}
