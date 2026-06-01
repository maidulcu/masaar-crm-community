'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'

interface BOS24Settings {
  api_key_set: boolean
  api_key_preview: string
  webhook_registered: boolean
  webhook_url: string
  webhook_secret: string
  webhook_id: string
  last_sync_at: string | null
}

export default function BOS24IntegrationPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [settings, setSettings] = useState<BOS24Settings | null>(null)
  const [loading, setLoading] = useState(true)
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)

  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [saveSuccess, setSaveSuccess] = useState(false)

  const [registering, setRegistering] = useState(false)
  const [registerError, setRegisterError] = useState('')
  const [registerSuccess, setRegisterSuccess] = useState(false)

  const [syncing, setSyncing] = useState(false)
  const [syncError, setSyncError] = useState('')
  const [syncResult, setSyncResult] = useState<{ listings_imported: number; inquiries_created: number } | null>(null)

  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    loadSettings()
  }, [user, router])

  const loadSettings = async () => {
    try {
      const r = await api.settings.bos24Integration.get() as { data?: BOS24Settings } & BOS24Settings
      // API returns fields directly (not wrapped in data)
      const s = (r as any).api_key_set !== undefined ? r as unknown as BOS24Settings : (r as any).data as BOS24Settings
      setSettings(s)
    } catch {
      // no settings yet — ok
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!apiKey.trim()) return
    setSaving(true); setSaveError(''); setSaveSuccess(false)
    try {
      await api.settings.bos24Integration.update({ api_key: apiKey })
      setSaveSuccess(true)
      setApiKey('')
      setShowKey(false)
      await loadSettings()
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save')
    } finally { setSaving(false) }
  }

  const handleRegister = async () => {
    setRegistering(true); setRegisterError(''); setRegisterSuccess(false)
    try {
      await api.settings.bos24Integration.registerWebhook()
      setRegisterSuccess(true)
      await loadSettings()
    } catch (err: unknown) {
      setRegisterError(err instanceof Error ? err.message : 'Failed to register')
    } finally { setRegistering(false) }
  }

  const handleSync = async () => {
    setSyncing(true); setSyncError(''); setSyncResult(null)
    try {
      const r = await api.settings.bos24Integration.syncNow() as any
      setSyncResult({ listings_imported: r.listings_imported ?? 0, inquiries_created: r.inquiries_created ?? 0 })
      await loadSettings()
    } catch (err: unknown) {
      setSyncError(err instanceof Error ? err.message : 'Sync failed')
    } finally { setSyncing(false) }
  }

  const copyWebhookURL = () => {
    if (!settings?.webhook_url) return
    navigator.clipboard.writeText(settings.webhook_url)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (user?.role !== 'admin') return null

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'
  const labelCls = 'block text-xs font-medium text-gray-600 mb-1'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('مزامنة BOS24', 'BOS24 Integration')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl space-y-6">

          {/* Demo notice */}
          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — BOS24 integration settings are read-only.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to connect your BOS24 account.
              </span>
            </div>
          )}

          {/* Explainer */}
          <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 text-sm text-blue-800">
            <p className="font-semibold mb-1">What this does</p>
            <p className="text-xs leading-relaxed">
              Connect your BuyOrSell24 account to automatically import listings into Masaar and
              turn buyer inquiries into leads. BOS24 sends real-time webhook events; a nightly
              sync acts as a safety net.
            </p>
          </div>

          {/* ── Step 1: API Key ── */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="flex items-center gap-2 mb-4">
              <span className="w-6 h-6 rounded-full bg-brand-600 text-white text-xs font-bold flex items-center justify-center">1</span>
              <h2 className="text-sm font-semibold text-gray-700">
                {t('مفتاح API للتكامل', 'Integration API Key')}
              </h2>
              {settings?.api_key_set && (
                <span className="ml-auto text-xs text-green-600 font-medium flex items-center gap-1">
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                  Configured ({settings.api_key_preview})
                </span>
              )}
            </div>
            <p className="text-xs text-gray-500 mb-4">
              {t('احصل على مفتاح API من إدارة BOS24. يختلف هذا عن مفتاح بيانات العقارات.', 'Get this key from your BOS24 admin. It is different from the real estate data API key.')}
            </p>
            <form onSubmit={handleSave} className="space-y-3">
              <div>
                <label className={labelCls}>{t('مفتاح API الجديد', 'New API Key')}</label>
                <div className="flex gap-2">
                  <input
                    type={showKey ? 'text' : 'password'}
                    value={apiKey}
                    onChange={e => setApiKey(e.target.value)}
                    placeholder="bos24_live_..."
                    disabled={isDemo}
                    className={inputCls + (isDemo ? ' bg-gray-50 cursor-not-allowed' : '')}
                  />
                  <button type="button" onClick={() => setShowKey(!showKey)}
                    className="px-3 py-2.5 border border-gray-200 rounded-lg text-xs font-medium text-gray-600 hover:bg-gray-50 shrink-0">
                    {showKey ? t('إخفاء', 'Hide') : t('عرض', 'Show')}
                  </button>
                </div>
              </div>
              {saveError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{saveError}</p>}
              {saveSuccess && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'API key saved successfully')}</p>}
              <button type="submit" disabled={saving || isDemo || !apiKey.trim()}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                {saving ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ المفتاح', 'Save API Key')}
              </button>
            </form>
          </div>

          {/* ── Step 2: Webhook Registration ── */}
          <div className={`bg-white rounded-xl border border-gray-200 p-6 ${!settings?.api_key_set ? 'opacity-60' : ''}`}>
            <div className="flex items-center gap-2 mb-4">
              <span className="w-6 h-6 rounded-full bg-brand-600 text-white text-xs font-bold flex items-center justify-center">2</span>
              <h2 className="text-sm font-semibold text-gray-700">
                {t('تسجيل Webhook', 'Register Webhook')}
              </h2>
              {settings?.webhook_registered && (
                <span className="ml-auto text-xs text-green-600 font-medium flex items-center gap-1">
                  <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                  Registered (ID: {settings.webhook_id})
                </span>
              )}
            </div>

            {settings?.webhook_url && (
              <div className="mb-4">
                <label className={labelCls}>Your webhook URL</label>
                <div className="flex gap-2">
                  <code className="flex-1 text-xs bg-gray-50 border border-gray-200 rounded-lg px-3 py-2.5 text-gray-600 font-mono truncate">
                    {settings.webhook_url}
                  </code>
                  <button type="button" onClick={copyWebhookURL}
                    className="px-3 py-2.5 border border-gray-200 rounded-lg text-xs font-medium text-gray-600 hover:bg-gray-50 shrink-0 transition-colors">
                    {copied ? '✓ Copied' : t('نسخ', 'Copy')}
                  </button>
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  BOS24 will call this URL for real-time events. The secret token is embedded in the URL.
                </p>
              </div>
            )}

            <p className="text-xs text-gray-500 mb-4">
              Clicking "Register" calls BOS24 automatically and subscribes to listing and inquiry events.
              You only need to do this once (or again if you regenerate your API key).
            </p>

            {registerError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded mb-3">{registerError}</p>}
            {registerSuccess && <p className="text-xs text-green-600 bg-green-50 p-2 rounded mb-3">Webhook registered successfully!</p>}

            <button
              type="button"
              onClick={handleRegister}
              disabled={registering || isDemo || !settings?.api_key_set}
              className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
            >
              {registering
                ? t('جارٍ التسجيل...', 'Registering...')
                : settings?.webhook_registered
                ? t('إعادة تسجيل', 'Re-register Webhook')
                : t('تسجيل Webhook مع BOS24', 'Register Webhook with BOS24')}
            </button>
          </div>

          {/* ── Step 3: Manual Sync ── */}
          <div className={`bg-white rounded-xl border border-gray-200 p-6 ${!settings?.api_key_set ? 'opacity-60' : ''}`}>
            <div className="flex items-center gap-2 mb-1">
              <span className="w-6 h-6 rounded-full bg-brand-600 text-white text-xs font-bold flex items-center justify-center">3</span>
              <h2 className="text-sm font-semibold text-gray-700">
                {t('مزامنة يدوية', 'Manual Sync')}
              </h2>
            </div>
            <p className="text-xs text-gray-500 mb-4 ml-8">
              {settings?.last_sync_at
                ? `Last synced: ${new Date(settings.last_sync_at).toLocaleString()}`
                : 'Never synced — run an initial import to pull all existing listings and inquiries.'}
              {' '}A nightly sync runs automatically at 3 AM.
            </p>

            {syncError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded mb-3">{syncError}</p>}
            {syncResult && (
              <div className="bg-green-50 border border-green-200 rounded-lg p-3 mb-3 text-xs text-green-800">
                Sync complete — <strong>{syncResult.listings_imported}</strong> listings imported,{' '}
                <strong>{syncResult.inquiries_created}</strong> new leads created.
              </div>
            )}

            <button
              type="button"
              onClick={handleSync}
              disabled={syncing || isDemo || !settings?.api_key_set}
              className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors flex items-center justify-center gap-2"
            >
              {syncing ? (
                <>
                  <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
                  </svg>
                  {t('جارٍ المزامنة...', 'Syncing...')}
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  {settings?.last_sync_at ? t('مزامنة الآن', 'Sync Now') : t('استيراد أولي', 'Run Initial Import')}
                </>
              )}
            </button>
          </div>

          {/* What gets synced info box */}
          <div className="bg-white rounded-xl border border-gray-200 p-5 text-xs text-gray-500 space-y-2">
            <p className="font-semibold text-gray-700 text-sm">What gets synced</p>
            <div className="grid grid-cols-2 gap-3">
              <div className="flex items-start gap-2">
                <svg className="w-4 h-4 text-brand-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                </svg>
                <div>
                  <p className="font-medium text-gray-700">Listings</p>
                  <p>Published BOS24 listings → Masaar listings table with price, location, images</p>
                </div>
              </div>
              <div className="flex items-start gap-2">
                <svg className="w-4 h-4 text-brand-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
                <div>
                  <p className="font-medium text-gray-700">Inquiries → Leads</p>
                  <p>Buyer inquiries create a contact + lead (source: bos24) with the buyer's message</p>
                </div>
              </div>
            </div>
          </div>

        </div>
      </main>
    </div>
  )
}
