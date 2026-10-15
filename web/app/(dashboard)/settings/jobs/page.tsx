'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type JobRun } from '@/lib/api'
import { formatDateTime } from '@/lib/date'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const JOB_LABELS: Record<string, { en: string; ar: string }> = {
  viewing_reminders: { en: 'Viewing reminders', ar: 'تذكيرات المعاينات' },
  offer_expiry: { en: 'Offer expiry', ar: 'انتهاء صلاحية العروض' },
  expiry_alerts: { en: 'Licence & permit expiry alerts', ar: 'تنبيهات انتهاء التراخيص والتصاريح' },
  retention: { en: 'Data retention clean-up', ar: 'تنظيف البيانات بعد مدة الاحتفاظ' },
  task_reminders: { en: 'Task reminders', ar: 'تذكيرات المهام' },
  cheque_deposits: { en: 'Cheques to deposit', ar: 'الشيكات المستحقة للإيداع' },
  lease_alerts: { en: 'Lease, Ejari and rent-notice alerts', ar: 'تنبيهات العقود وإيجاري وإشعارات الإيجار' },
  rent_reminders: { en: 'Rent reminders to tenants', ar: 'تذكيرات الإيجار للمستأجرين' },
  whatsapp_broadcasts: { en: 'WhatsApp broadcasts', ar: 'رسائل واتساب الجماعية' },
  einvoice_status: { en: 'E-invoice status', ar: 'حالة الفواتير الإلكترونية' },
}

export default function JobRunsPage() {
  const { user } = useAuthStore()
  const { t, lang } = useLang()
  const router = useRouter()
  const [runs, setRuns] = useState<JobRun[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.jobs.runs()
      .then(setRuns)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false))
  }, [user, router])

  if (user?.role !== 'admin') return null

  const label = (name: string) => {
    const l = JOB_LABELS[name]
    return l ? (lang === 'ar' ? l.ar : l.en) : name
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('المهام الخلفية', 'Background Jobs')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-4">
          <p className="text-sm text-gray-600">
            {t('تعمل المهام تلقائياً (تذكيرات المعاينات، انتهاء صلاحية العروض). تُسجَّل هنا فقط عمليات التشغيل التي عالجت عناصر أو فشلت، لمدة 30 يوماً.',
              'Jobs run automatically (viewing reminders, offer expiry). Only runs that handled something or failed are listed here, for 30 days.')}
          </p>
          {error && <p role="alert" className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
          <div className="bg-white rounded-xl border border-gray-200 overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs text-gray-500">
                <tr>
                  <th scope="col" className="px-4 py-3 text-start">{t('المهمة', 'Job')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('البدء', 'Started')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('الحالة', 'Status')}</th>
                  <th scope="col" className="px-4 py-3 text-end">{t('العناصر', 'Items')}</th>
                  <th scope="col" className="px-4 py-3 text-start">{t('الخطأ', 'Error')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {loading && (
                  <tr><td colSpan={5} className="px-4 py-6 text-center text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</td></tr>
                )}
                {!loading && runs.length === 0 && (
                  <tr><td colSpan={5} className="px-4 py-6 text-center text-gray-400">{t('لا توجد عمليات تشغيل مسجلة بعد', 'No runs recorded yet')}</td></tr>
                )}
                {runs.map((r) => (
                  <tr key={r.id}>
                    <td className="px-4 py-3 text-gray-900">{label(r.job_name)}</td>
                    <td className="px-4 py-3 text-gray-600 whitespace-nowrap">{formatDateTime(r.started_at)}</td>
                    <td className="px-4 py-3">
                      <span className={r.status === 'ok'
                        ? 'px-2 py-0.5 rounded-full text-xs bg-green-50 text-green-700'
                        : 'px-2 py-0.5 rounded-full text-xs bg-red-50 text-red-700'}>
                        {r.status === 'ok' ? t('ناجح', 'OK') : t('فشل', 'Failed')}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-end text-gray-700">{r.items}</td>
                    <td className="px-4 py-3 text-xs text-red-600 break-all">{r.error}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  )
}
