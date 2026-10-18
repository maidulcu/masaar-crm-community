'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api, type LeaseNotice } from '@/lib/api'
import { useLang } from '@/context/LangContext'

// Printable bilingual notice: English and Arabic side by side.
export default function NoticePrintPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useLang()
  const [notice, setNotice] = useState<LeaseNotice | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api.leaseOps.notice(id).then(setNotice).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [id])

  if (error) return <p className="p-6 text-sm text-red-600">{error}</p>
  if (!notice) return <p className="p-6 text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</p>

  return (
    <div className="flex-1 overflow-y-auto bg-white p-8 print:p-0">
      <div className="print:hidden mb-4 flex gap-2">
        <button type="button" onClick={() => window.print()} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">
          {t('طباعة / حفظ PDF', 'Print / Save as PDF')}
        </button>
      </div>
      <div className="grid grid-cols-2 gap-8 max-w-5xl text-[13px] leading-relaxed text-gray-900">
        <div dir="ltr" className="whitespace-pre-wrap">{notice.body_en}</div>
        <div dir="rtl" lang="ar" className="whitespace-pre-wrap">{notice.body_ar}</div>
      </div>
      <div className="grid grid-cols-2 gap-8 max-w-5xl mt-16 text-xs text-gray-600">
        <div>Received by tenant: ______________________ Date: __________</div>
        <div dir="rtl">استلمه المستأجر: ______________________ التاريخ: __________</div>
      </div>
    </div>
  )
}
