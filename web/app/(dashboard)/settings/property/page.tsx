'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type PropertySettings } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const input = 'block mt-1 px-2.5 py-1.5 border border-gray-300 rounded-lg text-sm'

export default function PropertySettingsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [s, setS] = useState<PropertySettings | null>(null)
  const [alertDays, setAlertDays] = useState('')
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.leaseOps.settings().then((v) => { setS(v); setAlertDays(v.lease_alert_days.join(', ')) })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [user, router])

  if (user?.role !== 'admin' || !s) return null

  const save = async (e: React.FormEvent) => {
    e.preventDefault(); setMsg(''); setError('')
    const days = alertDays.split(/[,\s،]+/).filter(Boolean).map(Number)
    if (days.some((d) => !Number.isInteger(d) || d < 1 || d > 365)) { setError(t('أيام التنبيه بين 1 و365', 'Alert days must be between 1 and 365')); return }
    try {
      const saved = await api.leaseOps.updateSettings({ ...s, lease_alert_days: days.sort((a, b) => b - a) })
      setS(saved); setAlertDays(saved.lease_alert_days.join(', ')); setMsg(t('تم الحفظ', 'Saved'))
    } catch (err) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('إعدادات إدارة العقارات', 'Property management settings')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <form onSubmit={save} className="max-w-2xl space-y-5">
          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-3">
            <h2 className="text-sm font-semibold text-gray-900">{t('تنبيهات العقود', 'Lease alerts')}</h2>
            <label className="block text-xs text-gray-600">{t('التنبيه قبل انتهاء العقد بـ (أيام، مفصولة بفواصل)', 'Alert before a lease ends (days, comma-separated)')}
              <input className={`${input} w-64`} value={alertDays} onChange={(e) => setAlertDays(e.target.value)} />
            </label>
            <label className="block text-xs text-gray-600">{t('مدة إشعار تعديل الإيجار (أيام قبل انتهاء العقد)', 'Rent-change notice period (days before the lease ends)')}
              <input type="number" min={0} max={365} className={`${input} w-28`} value={s.rent_notice_days} onChange={(e) => setS({ ...s, rent_notice_days: Number(e.target.value) })} />
            </label>
            <p className="text-[11px] text-gray-500">
              {t('في دبي تبلغ مدة الإشعار بتعديل شروط العقد 90 يوماً قبل انتهائه. تختلف القواعد بين الإمارات وقد تتغير؛ تحقق منها.',
                'In Dubai, a change to the tenancy terms must be notified 90 days before the contract ends. Rules differ between emirates and change; check them.')}
            </p>
          </section>

          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-3">
            <h2 className="text-sm font-semibold text-gray-900">{t('تذكيرات الإيجار للمستأجرين', 'Rent reminders to tenants')}</h2>
            <label className="flex items-center gap-2 text-sm text-gray-700">
              <input type="checkbox" checked={s.rent_reminders_enabled} onChange={(e) => setS({ ...s, rent_reminders_enabled: e.target.checked })} />
              {t('إرسال تذكيرات بالبريد الإلكتروني وواتساب', 'Send reminders by email and WhatsApp')}
            </label>
            <label className="block text-xs text-gray-600">{t('التذكير قبل الاستحقاق بـ (أيام)', 'Remind this many days before rent is due')}
              <input type="number" min={0} max={30} className={`${input} w-28`} value={s.rent_reminder_days_before} onChange={(e) => setS({ ...s, rent_reminder_days_before: Number(e.target.value) })} />
            </label>
            <div className="flex flex-wrap gap-3">
              <label className="block text-xs text-gray-600">{t('اسم قالب واتساب المعتمد', 'Approved WhatsApp template name')}
                <input className={`${input} w-64`} dir="ltr" value={s.rent_reminder_wa_template} onChange={(e) => setS({ ...s, rent_reminder_wa_template: e.target.value })} />
              </label>
              <label className="block text-xs text-gray-600">{t('لغة القالب', 'Template language')}
                <input className={`${input} w-24`} dir="ltr" value={s.rent_reminder_wa_language} onChange={(e) => setS({ ...s, rent_reminder_wa_language: e.target.value })} />
              </label>
            </div>
            <p className="text-[11px] text-gray-500">
              {t('يُرسل تذكير قبل الاستحقاق وآخر عند التأخر، بين 9 صباحاً و8 مساءً. لا تُرسل تذكيرات للدفعات المغطاة بشيك مستلم، ولا للمستأجرين الذين ألغوا الاشتراك. يحتاج واتساب إلى قالب معتمد من Meta بأربعة متغيرات: اسم المستأجر، المبلغ، تاريخ الاستحقاق، العقار.',
                'One reminder before rent is due and one when it is late, between 9:00 and 20:00. Payments covered by a cheque in hand are skipped, and so are tenants who opted out. WhatsApp needs a template approved by Meta with four body variables: tenant name, amount, due date, property.')}
            </p>
          </section>

          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
          {msg && <p className="text-xs text-green-700">{msg}</p>}
          <button type="submit" className="px-4 py-2 text-sm rounded-lg bg-primary-600 text-white">{t('حفظ', 'Save')}</button>
        </form>
      </main>
    </div>
  )
}
