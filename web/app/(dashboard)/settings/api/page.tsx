'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'

export default function APISettingsPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [bos24Token, setBos24Token] = useState('')
  const [bos24TokenDisplay, setBos24TokenDisplay] = useState('')
  const [showToken, setShowToken] = useState(false)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  // Check for admin access
  useEffect(() => {
    if (user && user.role !== 'admin') {
      router.push('/dashboard')
    }
  }, [user, router])

  // Load current settings
  useEffect(() => {
    const loadSettings = async () => {
      try {
        const data = await api.settings.getBOS24() as any
        setBos24Token(data.setting_value || '')
        if (data.setting_value) {
          const masked = data.setting_value.substring(0, 4) + '****' +
                        data.setting_value.substring(data.setting_value.length - 4)
          setBos24TokenDisplay(masked)
        }
      } catch (err) {
        console.error('Failed to load settings:', err)
      } finally {
        setLoading(false)
      }
    }

    loadSettings()
  }, [])

  const handleUpdateToken = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setSuccess(false)

    if (!bos24Token.trim()) {
      setError(t('الرجاء إدخال توكن API', 'Please enter an API token'))
      return
    }

    setSubmitting(true)
    try {
      const data = await api.settings.updateBOS24({ token: bos24Token }) as any
      setSuccess(true)
      setBos24Token(data.setting_value || '')
      const masked = data.setting_value.substring(0, 4) + '****' +
                    data.setting_value.substring(data.setting_value.length - 4)
      setBos24TokenDisplay(masked)
      setShowToken(false)

      setTimeout(() => setSuccess(false), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setSubmitting(false)
    }
  }

  if (user?.role !== 'admin') {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-red-600">{t('ليس لديك صلاحيات كافية', 'You do not have permission')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('إعدادات API', 'API Settings')} />

      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-lg space-y-6">

          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — API settings are read-only.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to configure integrations.
              </span>
            </div>
          )}

          {/* BuyOrSell24 API Settings */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="mb-6">
              <h2 className="text-sm font-semibold text-gray-700 mb-2">
                {t('بيانات العقارات - BuyOrSell24', 'Real Estate Data - BuyOrSell24')}
              </h2>
              <p className="text-xs text-gray-500">
                {t('خدمة بيانات العقارات من Dynamic Web Lab', 'Real estate data service by Dynamic Web Lab')}
              </p>
            </div>

            <form onSubmit={handleUpdateToken} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-2">
                  {t('API Token', 'API Token')}
                </label>
                <div className="flex gap-2">
                  <input
                    type={showToken ? 'text' : 'password'}
                    value={loading ? t('جارٍ التحميل...', 'Loading...') : (showToken ? bos24Token : bos24TokenDisplay)}
                    onChange={(e) => setBos24Token(e.target.value)}
                    disabled={loading}
                    className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent disabled:bg-gray-100"
                    placeholder={t('أدخل API token من Dynamic Web Lab', 'Enter API token from Dynamic Web Lab')}
                  />
                  <button
                    type="button"
                    onClick={() => setShowToken(!showToken)}
                    disabled={loading || !bos24Token}
                    className="px-3 py-2.5 border border-gray-200 rounded-lg text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50"
                  >
                    {showToken ? t('إخفاء', 'Hide') : t('عرض', 'Show')}
                  </button>
                </div>
                <p className="mt-2 text-xs text-gray-500">
                  {t('احصل على توكن من:', 'Get token from:')} <br/>
                  <a
                    href="https://dynamicweblab.com/products/real-estate-data-api/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-brand-600 hover:underline font-medium"
                  >
                    https://dynamicweblab.com/products/real-estate-data-api/
                  </a>
                </p>
              </div>

              {error && (
                <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>
              )}
              {success && (
                <p className="text-xs text-green-600 bg-green-50 p-2 rounded">
                  {t('تم تحديث الإعدادات بنجاح', 'Settings updated successfully')}
                </p>
              )}

              <button
                type="submit"
                disabled={submitting || loading || isDemo}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                {submitting
                  ? t('جارٍ الحفظ...', 'Saving...')
                  : isDemo
                  ? t('غير متاح في الحساب التجريبي', 'Not available in demo')
                  : t('حفظ الإعدادات', 'Save Settings')}
              </button>
            </form>

            {/* Info box */}
            <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
              <p className="text-xs text-blue-700">
                <span className="font-semibold">{t('ملاحظة:', 'Note:')}</span> {t('عند إدخال API token، تنشيط ميزات بحث العقارات لجميع الوكلاء', 'When you add an API token, real estate features will be enabled for all agents')}
              </p>
            </div>
          </div>

        </div>
      </main>
    </div>
  )
}
