'use client'
import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api, type EInvoiceSettings } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const input = 'w-full px-3 py-2 border border-gray-300 rounded-lg text-sm'
const label = 'block text-xs font-medium text-gray-600 mb-1'

export default function EInvoicingSettingsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [s, setS] = useState<EInvoiceSettings | null>(null)
  const [providers, setProviders] = useState<string[]>(['off'])
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.einvoice.settings().then((r) => { setS(r.settings); setProviders(r.providers) })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [user, router])

  if (user?.role !== 'admin' || !s) return null
  const set = (k: keyof EInvoiceSettings) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setS({ ...s, [k]: e.target.value })
  const providerLabel = (p: string) =>
    p === 'off' ? t('إيقاف (PDF فقط)', 'Off (PDF only)') : p === 'mock' ? t('تجريبي (لا يرسل شيئاً)', 'Mock (sends nothing)') : p

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('الفوترة الإلكترونية', 'E-invoicing')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <form className="max-w-3xl space-y-4" onSubmit={async (e) => {
          e.preventDefault(); setMsg(''); setError('')
          try { setS(await api.einvoice.updateSettings(s)); setMsg(t('تم الحفظ', 'Saved')) } catch (err) { setError(err instanceof Error ? err.message : 'Failed') }
        }}>
          <section className="bg-white rounded-xl border border-gray-200 p-5 space-y-3 text-sm text-gray-600">
            <p>
              {t('تُنشئ هذه الميزة فواتير إلكترونية بصيغة PINT AE (Peppol) للإمارات، وتُرسلها عبر مزود خدمة معتمد. لا يتضمن المشروع مزوداً حقيقياً: يمكن إضافة موصّل لمزودك (انظر docs/einvoicing.md). تحقق من المتطلبات وجداولها الزمنية من وزارة المالية والهيئة الاتحادية للضرائب؛ هذا ليس استشارة قانونية أو ضريبية.',
                'Builds UAE e-invoices in the PINT AE (Peppol) format and submits them through an accredited service provider. No real provider ships with the project: add a connector for yours (see docs/einvoicing.md). Check the requirements and timelines with the Ministry of Finance and the FTA; this is not legal or tax advice.')}
            </p>
            <p>
              {t('يأتي اسم البائع ورقم التسجيل الضريبي والعنوان والإمارة والآيبان من ', 'The seller name, TRN, address, emirate and IBAN come from ')}
              <Link href="/settings/company" className="text-brand-600 hover:underline">{t('إعدادات الشركة', 'Company settings')}</Link>.
            </p>
          </section>

          <section className="bg-white rounded-xl border border-gray-200 p-5 grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="ei-provider" className={label}>{t('المزود', 'Provider')}</label>
              <select id="ei-provider" className={input} value={s.provider} onChange={set('provider')}>
                {providers.map((p) => <option key={p} value={p}>{providerLabel(p)}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="ei-tin" className={label}>{t('رقم التعريف الضريبي (10 أرقام)', 'TIN (10 digits)')}</label>
              <input id="ei-tin" dir="ltr" inputMode="numeric" maxLength={10} className={input} value={s.tin} onChange={set('tin')} />
              <p className="text-xs text-gray-500 mt-1">{t('معرّف المشارك في الفوترة الإلكترونية.', 'Your e-invoicing participant ID.')}</p>
            </div>
            <div>
              <label htmlFor="ei-ltype" className={label}>{t('نوع التسجيل القانوني', 'Legal registration')}</label>
              <select id="ei-ltype" className={input} value={s.legal_id_type} onChange={set('legal_id_type')}>
                <option value="TL">{t('رخصة تجارية', 'Trade licence')}</option>
                <option value="EID">{t('الهوية الإماراتية', 'Emirates ID')}</option>
                <option value="PAS">{t('جواز السفر', 'Passport')}</option>
                <option value="CD">{t('قرار مجلس الوزراء', 'Cabinet decision')}</option>
              </select>
              <p className="text-xs text-gray-500 mt-1">{t('الرقم نفسه هو رقم الرخصة التجارية في إعدادات الشركة.', 'The number is the trade licence number in Company settings.')}</p>
            </div>
            <div>
              <label htmlFor="ei-auth" className={label}>{t('جهة الترخيص (أو بلد الجواز)', 'Licensing authority (or passport country)')}</label>
              <input id="ei-auth" className={input} value={s.trade_license_authority} onChange={set('trade_license_authority')} placeholder="Dubai Economy and Tourism" />
            </div>
            <div>
              <label htmlFor="ei-city" className={label}>{t('المدينة', 'City')}</label>
              <input id="ei-city" className={input} value={s.city} onChange={set('city')} />
            </div>
            <div>
              <label htmlFor="ei-sac" className={label}>{t('رمز محاسبة الخدمة (SAC) للبنود', 'Service accounting code (SAC) of the lines')}</label>
              <input id="ei-sac" dir="ltr" className={input} value={s.service_code} onChange={set('service_code')} />
              <p className="text-xs text-gray-500 mt-1">{t('اتركه فارغاً إن لم يطلبه مزودك.', 'Leave empty unless your provider asks for it.')}</p>
            </div>
            <div>
              <label htmlFor="ei-vatex" className={label}>{t('رمز سبب الإعفاء من الضريبة', 'VAT exemption reason code')}</label>
              <input id="ei-vatex" dir="ltr" className={input} value={s.exemption_code} onChange={set('exemption_code')} />
              <p className="text-xs text-gray-500 mt-1">{t('مطلوب فقط للفواتير التي تحتوي على بنود معفاة. استخدم الرمز الذي يحدده مزودك.', 'Needed only for invoices with exempt lines. Use the code your provider specifies.')}</p>
            </div>
          </section>

          <div className="flex items-center gap-3">
            <button type="submit" className="px-4 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium">{t('حفظ', 'Save')}</button>
            {msg && <span className="text-sm text-green-700">{msg}</span>}
            {error && <span role="alert" className="text-sm text-red-600">{error}</span>}
          </div>
        </form>
      </main>
    </div>
  )
}
