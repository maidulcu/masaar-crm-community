'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import type { ApprovalConfig } from '@/types'

export default function ApprovalSettingsPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [config, setConfig] = useState<ApprovalConfig>({
    id: '', company_id: '', listing_approval: false,
    deal_approval_above: 0, offer_approval_above: 0, updated_at: '',
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.approvals.getConfig()
      .then((r: any) => { if (r) setConfig(r) })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [user, router])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isDemo) return
    setSaving(true); setError(''); setSaved(false)
    try {
      const res = await api.approvals.saveConfig({
        listing_approval: config.listing_approval,
        deal_approval_above: config.deal_approval_above,
        offer_approval_above: config.offer_approval_above,
      }) as ApprovalConfig
      setConfig(res)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    } catch (err: any) {
      setError(err.message || 'Failed to save')
    } finally { setSaving(false) }
  }

  if (user?.role !== 'admin') return null

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('إعدادات الموافقات', 'Approval Workflows')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-lg space-y-6">

          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span><strong>Demo account</strong> — settings are read-only.</span>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-1">{t('الموافقات', 'Approval Workflows')}</h2>
            <p className="text-xs text-gray-500 mb-4">{t('حدد متى تتطلب الإجراءات موافقة المشرف', 'Configure when manager approval is required')}</p>

            <form onSubmit={handleSave} className="space-y-5">

              {/* Listing approval toggle */}
              <div className="flex items-center justify-between py-2">
                <div>
                  <p className="text-sm font-medium text-gray-800">{t('الموافقة على نشر الإعلانات', 'Listing Publishing Approval')}</p>
                  <p className="text-xs text-gray-500 mt-0.5">{t('يتطلب موافقة المدير قبل نشر أي إعلان', 'Requires admin approval before any listing can be published')}</p>
                </div>
                <button type="button" onClick={() => !isDemo && setConfig(s => ({ ...s, listing_approval: !s.listing_approval }))}
                  className={`relative w-11 h-6 rounded-full transition-colors ${config.listing_approval ? 'bg-brand-600' : 'bg-gray-300'} ${isDemo ? 'cursor-not-allowed' : 'cursor-pointer'}`}>
                  <div className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${config.listing_approval ? 'translate-x-5' : ''}`} />
                </button>
              </div>

              {/* Deal approval threshold */}
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  {t('الموافقة على الصفقات (أكثر من)', 'Deal Approval Threshold')}
                  <span className="text-gray-400 font-normal ml-1">({t('0 = لا يوجد حد', '0 = no threshold')})</span>
                </label>
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">AED</span>
                  <input type="number" min={0} step={1000} value={config.deal_approval_above}
                    onChange={e => setConfig(s => ({ ...s, deal_approval_above: parseFloat(e.target.value) || 0 }))}
                    disabled={isDemo}
                    className={`${inputCls} pl-12 ${isDemo ? 'bg-gray-50 cursor-not-allowed' : ''}`} />
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  {t('الصفقات التي تتجاوز هذا المبلغ تتطلب موافقة المدير.', 'Deals above this amount require admin approval.')}
                </p>
              </div>

              {/* Offer approval threshold */}
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  {t('الموافقة على العروض (أكثر من)', 'Offer Approval Threshold')}
                  <span className="text-gray-400 font-normal ml-1">({t('0 = لا يوجد حد', '0 = no threshold')})</span>
                </label>
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">AED</span>
                  <input type="number" min={0} step={1000} value={config.offer_approval_above}
                    onChange={e => setConfig(s => ({ ...s, offer_approval_above: parseFloat(e.target.value) || 0 }))}
                    disabled={isDemo}
                    className={`${inputCls} pl-12 ${isDemo ? 'bg-gray-50 cursor-not-allowed' : ''}`} />
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  {t('العروض التي تتجاوز هذا المبلغ تتطلب موافقة المدير.', 'Offers above this amount require admin approval.')}
                </p>
              </div>

              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              {saved && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'Settings saved')}</p>}

              <button type="submit" disabled={saving || isDemo}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                {saving ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ الإعدادات', 'Save Settings')}
              </button>
            </form>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 p-5 text-xs text-gray-500 space-y-2">
            <p className="font-semibold text-gray-700 text-sm">{t('كيف يعمل؟', 'How it works')}</p>
            <p>When listing approval is enabled, agents who try to publish a listing will create an <strong>approval request</strong> instead. Admins review and approve/reject these requests from the <strong>Approvals</strong> page.</p>
            <p>Deal and offer thresholds work similarly — if a deal value or offer amount exceeds the configured limit, an approval request is created automatically.</p>
          </div>

        </div>
      </main>
    </div>
  )
}
