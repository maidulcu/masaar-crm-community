'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type HoursSpan, type HoursSpecial, type InboxSettings } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'
const DAYS: [string, string][] = [['الأحد', 'Sunday'], ['الإثنين', 'Monday'], ['الثلاثاء', 'Tuesday'], ['الأربعاء', 'Wednesday'], ['الخميس', 'Thursday'], ['الجمعة', 'Friday'], ['السبت', 'Saturday']]
const defaultWeek = (): HoursSpan[] => DAYS.map((_, i) => (i === 0 || i === 6 ? { open: '', close: '' } : { open: '09:00', close: '18:00' }))

export default function InboxSettingsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [s, setS] = useState<InboxSettings | null>(null)
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.broadcasts.inboxSettings().then((v) => setS({ ...v, hours: v.hours ?? { week: defaultWeek(), specials: [] } }))
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [user, router])

  if (user?.role !== 'admin' || !s || !s.hours) return null
  const hours = s.hours
  const setDay = (i: number, v: Partial<HoursSpan>) => setS({ ...s, hours: { ...hours, week: hours.week.map((d, j) => (j === i ? { ...d, ...v } : d)) } })
  const setSpecial = (i: number, v: Partial<HoursSpecial>) => setS({ ...s, hours: { ...hours, specials: hours.specials.map((d, j) => (j === i ? { ...d, ...v } : d)) } })

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('إعدادات صندوق واتساب', 'WhatsApp inbox settings')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <form className="max-w-3xl space-y-4" onSubmit={async (e) => {
          e.preventDefault(); setMsg(''); setError('')
          try { setS(await api.broadcasts.updateInboxSettings(s)); setMsg(t('تم الحفظ', 'Saved')) } catch (err) { setError(err instanceof Error ? err.message : 'Failed') }
        }}>
          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-3">
            <h2 className="text-sm font-semibold text-gray-900">{t('الرد التلقائي خارج ساعات العمل', 'Auto-reply outside business hours')}</h2>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={s.auto_reply_enabled} onChange={(e) => setS({ ...s, auto_reply_enabled: e.target.checked })} />
              {t('إرسال رد تلقائي (مرة كل 12 ساعة لكل محادثة)', 'Send an automatic reply (once every 12 hours per conversation)')}
            </label>
            <textarea dir="ltr" rows={2} className={`${input} w-full`} placeholder="Thanks for your message. We are open Monday–Friday 9:00–18:00 and will reply then." value={s.auto_reply_en} onChange={(e) => setS({ ...s, auto_reply_en: e.target.value })} aria-label="English" />
            <textarea dir="rtl" rows={2} className={`${input} w-full`} placeholder="شكراً لرسالتك. ساعات العمل من الإثنين إلى الجمعة 9:00–18:00 وسنرد عليك حينها." value={s.auto_reply_ar} onChange={(e) => setS({ ...s, auto_reply_ar: e.target.value })} aria-label="العربية" />
          </section>

          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-2">
            <h2 className="text-sm font-semibold text-gray-900">{t('ساعات العمل (بتوقيت الشركة)', 'Business hours (company timezone)')}</h2>
            {hours.week.map((d, i) => (
              <div key={i} className="flex flex-wrap items-center gap-2 text-sm">
                <span className="w-24 text-gray-700">{t(...DAYS[i])}</span>
                <label className="flex items-center gap-1 text-xs text-gray-600">
                  <input type="checkbox" checked={!d.open} onChange={(e) => setDay(i, e.target.checked ? { open: '', close: '' } : { open: '09:00', close: '18:00' })} />
                  {t('مغلق', 'Closed')}
                </label>
                {d.open && (
                  <>
                    <input type="time" className={input} value={d.open} onChange={(e) => setDay(i, { open: e.target.value })} aria-label={t('يفتح', 'Opens')} />
                    <span>–</span>
                    <input type="time" className={input} value={d.close === '24:00' ? '23:59' : d.close} onChange={(e) => setDay(i, { close: e.target.value })} aria-label={t('يغلق', 'Closes')} />
                  </>
                )}
              </div>
            ))}
            <h3 className="text-xs font-semibold text-gray-700 pt-3">{t('فترات خاصة (رمضان، العطل)', 'Special periods (Ramadan, holidays)')}</h3>
            {hours.specials.map((sp, i) => (
              <div key={i} className="flex flex-wrap items-center gap-2">
                <input className={`${input} w-32`} placeholder={t('الاسم', 'Label')} value={sp.label} onChange={(e) => setSpecial(i, { label: e.target.value })} />
                <input type="date" className={input} value={sp.from} onChange={(e) => setSpecial(i, { from: e.target.value })} aria-label={t('من', 'From')} />
                <input type="date" className={input} value={sp.to} onChange={(e) => setSpecial(i, { to: e.target.value })} aria-label={t('إلى', 'To')} />
                <input type="time" className={input} value={sp.open} onChange={(e) => setSpecial(i, { open: e.target.value })} aria-label={t('يفتح', 'Opens')} />
                <input type="time" className={input} value={sp.close} onChange={(e) => setSpecial(i, { close: e.target.value })} aria-label={t('يغلق', 'Closes')} />
                <span className="text-[11px] text-gray-500">{t('(اترك الوقت فارغاً للإغلاق)', '(leave times empty: closed)')}</span>
                <button type="button" onClick={() => setS({ ...s, hours: { ...hours, specials: hours.specials.filter((_, j) => j !== i) } })} className="text-xs text-red-600">{t('حذف', 'Remove')}</button>
              </div>
            ))}
            <button type="button" className="text-xs text-primary-600 hover:underline"
              onClick={() => setS({ ...s, hours: { ...hours, specials: [...hours.specials, { label: t('رمضان', 'Ramadan'), from: '', to: '', open: '09:00', close: '15:00' }] } })}>
              {t('إضافة فترة خاصة', 'Add special period')}
            </button>
          </section>

          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-2">
            <h2 className="text-sm font-semibold text-gray-900">{t('الرسائل الجماعية', 'Broadcasts')}</h2>
            <label className="text-xs text-gray-600">{t('الحد الأقصى للرسائل في الدقيقة', 'Messages per minute')}
              <input type="number" min={1} max={1000} className={`${input} block w-28 mt-1`} value={s.broadcast_per_minute} onChange={(e) => setS({ ...s, broadcast_per_minute: Number(e.target.value) })} />
            </label>
            <p className="text-[11px] text-gray-500">{t('ابقَ ضمن حدود فئة المراسلة لرقمك في واتساب.', 'Stay within your WhatsApp number’s messaging tier.')}</p>
          </section>

          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
          {msg && <p className="text-xs text-green-700">{msg}</p>}
          <button type="submit" className="px-4 py-2 text-sm rounded-lg bg-primary-600 text-white">{t('حفظ', 'Save')}</button>
        </form>
      </main>
    </div>
  )
}
