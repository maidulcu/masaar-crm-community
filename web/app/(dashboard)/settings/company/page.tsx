'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'

interface CompanyData {
  company_name: string
  address: string
  phone: string
  email: string
  vat_number: string
  trn: string
  bank_name: string
  bank_account: string
  iban: string
  invoice_footer: string
  logo_url: string
  disclaimer: string
  primary_color: string
}

export default function CompanySettingsPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [form, setForm] = useState<CompanyData>({
    company_name: '', address: '', phone: '', email: '',
    vat_number: '', trn: '', bank_name: '', bank_account: '',
    iban: '', invoice_footer: '', logo_url: '',
    disclaimer: '', primary_color: '#1a3a5c',
  })
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.settings.getCompany().then((r: any) => {
      if (r?.data) setForm(r.data)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [user, router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(''); setSuccess(false); setSubmitting(true)
    try {
      await api.settings.updateCompany(form)
      setSuccess(true)
      setTimeout(() => setSuccess(false), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally { setSubmitting(false) }
  }

  const update = (k: keyof CompanyData) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm(p => ({ ...p, [k]: e.target.value }))

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
                  <label className={labelCls}>{t('اسم الشركة', 'Company Name')}</label>
                  <input className={inputCls} value={form.company_name} onChange={update('company_name')} />
                </div>
                <div className="col-span-2">
                  <label className={labelCls}>{t('العنوان', 'Address')}</label>
                  <textarea className={inputCls + ' resize-none'} rows={2} value={form.address} onChange={update('address')} />
                </div>
                <div>
                  <label className={labelCls}>{t('الهاتف', 'Phone')}</label>
                  <input className={inputCls} value={form.phone} onChange={update('phone')} />
                </div>
                <div>
                  <label className={labelCls}>{t('البريد الإلكتروني', 'Email')}</label>
                  <input className={inputCls} value={form.email} onChange={update('email')} />
                </div>
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('الضرائب والفواتير', 'Tax & Invoicing')}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('الرقم الضريبي (VAT)', 'VAT Number')}</label>
                  <input className={inputCls} value={form.vat_number} onChange={update('vat_number')} />
                </div>
                <div>
                  <label className={labelCls}>{t('TRN', 'TRN')}</label>
                  <input className={inputCls} value={form.trn} onChange={update('trn')} />
                </div>
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('معلومات البنك', 'Bank Information')}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('اسم البنك', 'Bank Name')}</label>
                  <input className={inputCls} value={form.bank_name} onChange={update('bank_name')} />
                </div>
                <div>
                  <label className={labelCls}>{t('رقم الحساب', 'Account Number')}</label>
                  <input className={inputCls} value={form.bank_account} onChange={update('bank_account')} />
                </div>
                <div className="col-span-2">
                  <label className={labelCls}>{t('IBAN', 'IBAN')}</label>
                  <input className={inputCls} value={form.iban} onChange={update('iban')} />
                </div>
              </div>

              <hr className="border-gray-100" />
              <div>
                <label className={labelCls}>{t('تذييل الفاتورة', 'Invoice Footer')}</label>
                <textarea className={inputCls + ' resize-none'} rows={2} value={form.invoice_footer} onChange={update('invoice_footer')} />
              </div>

              <hr className="border-gray-100" />
              <h3 className="text-sm font-semibold text-gray-700">{t('العلامة التجارية', 'Branding')}</h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>{t('رابط الشعار', 'Logo URL')}</label>
                  <input className={inputCls} value={form.logo_url} onChange={update('logo_url')} placeholder="https://..." />
                  {form.logo_url && <img src={form.logo_url} alt="logo preview" className="mt-2 h-10 object-contain rounded border border-gray-100" />}
                </div>
                <div>
                  <label className={labelCls}>{t('اللون الأساسي', 'Primary Color')}</label>
                  <div className="flex gap-2 items-center">
                    <input type="color" value={form.primary_color} onChange={update('primary_color')}
                      className="w-10 h-10 rounded border border-gray-200 cursor-pointer p-0.5" />
                    <input className={inputCls} value={form.primary_color} onChange={update('primary_color')} placeholder="#1a3a5c" />
                  </div>
                </div>
              </div>
              <div>
                <label className={labelCls}>{t('إخلاء المسؤولية', 'Disclaimer (on brochures)')}</label>
                <textarea className={inputCls + ' resize-none'} rows={2} value={form.disclaimer} onChange={update('disclaimer')}
                  placeholder="All information subject to change. Contact agent for verification." />
              </div>

              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              {success && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'Saved successfully')}</p>}

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
