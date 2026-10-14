'use client'
import { safeUrl } from '@/lib/safeUrl'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { api, type JurisdictionRules } from '@/lib/api'
import { CURRENCIES } from '@/lib/money'
import { useAuthStore } from '@/store/auth'
import { useLocaleStore } from '@/store/locale'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'

// Field names match the API (domain.CompanySettings).
interface CompanyData {
  name: string
  business_address: string
  business_phone: string
  business_email: string
  vat_number: string
  bank_name: string
  bank_account: string
  bank_iban: string
  logo_url: string
  disclaimer: string
  primary_color: string
  country: string
  region: string
  currency: string
  timezone: string
}

const EMPTY: CompanyData = {
  name: '', business_address: '', business_phone: '', business_email: '',
  vat_number: '', bank_name: '', bank_account: '', bank_iban: '',
  logo_url: '', disclaimer: '', primary_color: '#1a3a5c',
  country: 'AE', region: '', currency: 'AED', timezone: 'Asia/Dubai',
}

export default function CompanySettingsPage() {
  const { user, company } = useAuthStore()
  const loadLocale = useLocaleStore((s) => s.load)
  const { t, lang } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [form, setForm] = useState<CompanyData>(EMPTY)
  const [countries, setCountries] = useState<JurisdictionRules[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    Promise.all([
      api.settings.getCompany().then((r) => setForm({ ...EMPTY, ...(r as Partial<CompanyData>) })),
      api.settings.jurisdictions().then(setCountries),
    ]).catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load'))
      .finally(() => setLoading(false))
  }, [user, router])

  const rules = countries.find((c) => c.country === form.country)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(''); setSuccess(false); setSubmitting(true)
    try {
      const saved = await api.settings.updateCompany(form)
      setForm({ ...EMPTY, ...(saved as Partial<CompanyData>) })
      await loadLocale() // currency and timezone are used across the app
      setSuccess(true)
      setTimeout(() => setSuccess(false), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally { setSubmitting(false) }
  }

  const update = (k: keyof CompanyData) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) =>
    setForm(p => ({ ...p, [k]: e.target.value }))

  const changeCountry = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const next = countries.find((c) => c.country === e.target.value)
    setForm(p => ({
      ...p,
      country: e.target.value,
      region: '',
      currency: next?.currency ?? p.currency,
      timezone: next?.timezone ?? p.timezone,
    }))
  }

  if (user?.role !== 'admin') return null

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'
  const labelCls = 'block text-xs font-medium text-gray-600 mb-1'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('إعدادات الشركة', 'Company Settings')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl space-y-6">
          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — company settings are read-only.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to configure your workspace.
              </span>
            </div>
          )}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-4">
              {t('معلومات الشركة', 'Company Information')}
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label htmlFor="cs-name" className={labelCls}>{t('اسم الشركة', 'Company Name')} *</label>
                  <input id="cs-name" required className={inputCls} value={form.name} onChange={update('name')} />
                </div>
                <div className="col-span-2">
                  <label htmlFor="cs-address" className={labelCls}>{t('العنوان', 'Address')} *</label>
                  <textarea id="cs-address" required className={inputCls + ' resize-none'} rows={2} value={form.business_address} onChange={update('business_address')} />
                </div>
                <div>
                  <label htmlFor="cs-phone" className={labelCls}>{t('الهاتف', 'Phone')}</label>
                  <input id="cs-phone" dir="ltr" className={inputCls} value={form.business_phone} onChange={update('business_phone')} />
                </div>
                <div>
                  <label htmlFor="cs-email" className={labelCls}>{t('البريد الإلكتروني', 'Email')}</label>
                  <input id="cs-email" type="email" dir="ltr" className={inputCls} value={form.business_email} onChange={update('business_email')} />
                </div>
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('الدولة والعملة', 'Country & Currency')}</h3>
              <p className="text-xs text-gray-500">
                {t('تحدد الدولة نسب ضريبة القيمة المضافة وأيام العطلة. تُعرض المبالغ والتواريخ بالعملة والمنطقة الزمنية المختارة.',
                  'The country sets VAT rates and weekend days. Amounts and dates are shown in the currency and timezone chosen here.')}
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="cs-country" className={labelCls}>{t('الدولة', 'Country')}</label>
                  <select id="cs-country" className={inputCls} value={form.country} onChange={changeCountry}>
                    {countries.map((c) => (
                      <option key={c.country} value={c.country}>{lang === 'ar' ? c.name_ar : c.name_en}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label htmlFor="cs-region" className={labelCls}>{t('الإمارة / المنطقة', 'Emirate / Region')}</label>
                  <select id="cs-region" className={inputCls} value={form.region} onChange={update('region')}>
                    <option value="">{t('— غير محدد —', '— Not set —')}</option>
                    {rules?.regions.map((r) => (
                      <option key={r.code} value={r.code}>{lang === 'ar' ? r.name_ar : r.name_en}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label htmlFor="cs-currency" className={labelCls}>{t('العملة الافتراضية', 'Default Currency')}</label>
                  <select id="cs-currency" className={inputCls} value={form.currency} onChange={update('currency')}>
                    {[...new Set([form.currency, ...CURRENCIES])].map((c) => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="cs-tz" className={labelCls}>{t('المنطقة الزمنية', 'Timezone')}</label>
                  <input id="cs-tz" dir="ltr" className={inputCls} value={form.timezone} onChange={update('timezone')} placeholder="Asia/Dubai" />
                </div>
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('الضرائب والفواتير', 'Tax & Invoicing')}</h3>
              <div>
                <label htmlFor="cs-trn" className={labelCls}>{t('رقم التسجيل الضريبي (TRN)', 'Tax Registration Number (TRN)')}</label>
                <input id="cs-trn" dir="ltr" className={inputCls} value={form.vat_number} onChange={update('vat_number')}
                  placeholder={t('اتركه فارغاً إذا لم تكن مسجلاً لضريبة القيمة المضافة', 'Leave empty if not VAT-registered')} />
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('معلومات البنك', 'Bank Information')}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="cs-bank" className={labelCls}>{t('اسم البنك', 'Bank Name')}</label>
                  <input id="cs-bank" className={inputCls} value={form.bank_name} onChange={update('bank_name')} />
                </div>
                <div>
                  <label htmlFor="cs-account" className={labelCls}>{t('رقم الحساب', 'Account Number')}</label>
                  <input id="cs-account" dir="ltr" className={inputCls} value={form.bank_account} onChange={update('bank_account')} />
                </div>
                <div className="col-span-2">
                  <label htmlFor="cs-iban" className={labelCls}>{t('IBAN', 'IBAN')}</label>
                  <input id="cs-iban" dir="ltr" className={inputCls} value={form.bank_iban} onChange={update('bank_iban')} />
                </div>
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('العلامة التجارية', 'Branding')}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="cs-logo" className={labelCls}>{t('رابط الشعار', 'Logo URL')}</label>
                  <input id="cs-logo" dir="ltr" className={inputCls} value={form.logo_url} onChange={update('logo_url')} placeholder="https://..." />
                  {form.logo_url && <img src={safeUrl(form.logo_url)} alt="" className="mt-2 h-10 object-contain rounded border border-gray-100" />}
                </div>
                <div>
                  <label htmlFor="cs-color" className={labelCls}>{t('اللون الأساسي', 'Primary Color')}</label>
                  <div className="flex gap-2 items-center">
                    <input type="color" aria-label={t('اختيار اللون', 'Pick colour')} value={form.primary_color} onChange={update('primary_color')}
                      className="w-10 h-10 rounded border border-gray-200 cursor-pointer p-0.5" />
                    <input id="cs-color" dir="ltr" className={inputCls} value={form.primary_color} onChange={update('primary_color')} placeholder="#1a3a5c" />
                  </div>
                </div>
              </div>
              <div>
                <label htmlFor="cs-disclaimer" className={labelCls}>{t('إخلاء المسؤولية', 'Disclaimer (on brochures)')}</label>
                <textarea id="cs-disclaimer" className={inputCls + ' resize-none'} rows={2} value={form.disclaimer} onChange={update('disclaimer')}
                  placeholder="All information subject to change. Contact agent for verification." />
              </div>

              {error && <p role="alert" className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              {success && <p role="status" className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'Saved successfully')}</p>}

              <button type="submit" disabled={submitting || loading || isDemo}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                {submitting
                  ? t('جارٍ الحفظ...', 'Saving...')
                  : isDemo
                  ? t('غير متاح في الحساب التجريبي', 'Not available in demo')
                  : t('حفظ الإعدادات', 'Save Settings')}
              </button>
            </form>
          </div>
        </div>
      </main>
    </div>
  )
}
