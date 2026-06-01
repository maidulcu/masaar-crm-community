'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'

interface RotationSettings {
  mode: 'manual' | 'round_robin' | 'capacity'
  enabled: boolean
  max_per_agent: number
  agent_count: number
  updated_at: string
}

export default function LeadRotationPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [settings, setSettings] = useState<RotationSettings>({
    mode: 'manual', enabled: false, max_per_agent: 0, agent_count: 0, updated_at: '',
  })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    api.settings.leadRotation.get()
      .then((r: any) => setSettings(r))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [user, router])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isDemo) return
    setSaving(true); setError(''); setSaved(false)
    try {
      await api.settings.leadRotation.update({
        mode: settings.mode,
        enabled: settings.enabled,
        max_per_agent: settings.max_per_agent,
      })
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
      <Header title={t('توزيع العملاء المحتملين', 'Lead Rotation')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-lg space-y-6">

          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span><strong>Demo account</strong> — settings are read-only. <a href="/signup" className="underline font-semibold">Start a free trial</a> to configure lead rotation.</span>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-1">{t('توزيع العملاء المحتملين تلقائياً', 'Automatic Lead Assignment')}</h2>
            <p className="text-xs text-gray-500 mb-4">
              {settings.agent_count} {t('وكلاء نشطون', 'active agent(s)')} available for assignment.
            </p>

            <form onSubmit={handleSave} className="space-y-5">

              {/* Enable toggle */}
              <div className="flex items-center justify-between py-2">
                <div>
                  <p className="text-sm font-medium text-gray-800">{t('تفعيل التوزيع التلقائي', 'Enable Auto-Assignment')}</p>
                  <p className="text-xs text-gray-500 mt-0.5">{t('توزيع العملاء الجدد تلقائياً على الوكلاء', 'New leads are automatically assigned to agents')}</p>
                </div>
                <button type="button" onClick={() => !isDemo && setSettings(s => ({ ...s, enabled: !s.enabled }))}
                  className={`relative w-11 h-6 rounded-full transition-colors ${settings.enabled ? 'bg-brand-600' : 'bg-gray-300'} ${isDemo ? 'cursor-not-allowed' : 'cursor-pointer'}`}>
                  <div className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${settings.enabled ? 'translate-x-5' : ''}`} />
                </button>
              </div>

              {/* Mode selector */}
              <div className={settings.enabled ? '' : 'opacity-50 pointer-events-none'}>
                <label className="block text-xs font-medium text-gray-600 mb-2">{t('طريقة التوزيع', 'Distribution Mode')}</label>
                <div className="space-y-2">
                  {[
                    {
                      value: 'round_robin',
                      label: t('التناوب', 'Round Robin'),
                      desc: t('كل وكيل بالتسلسل — أكثر الطرق توازناً', 'Each agent takes turns in order — most balanced'),
                    },
                    {
                      value: 'capacity',
                      label: t('حسب الطاقة', 'Capacity-Based'),
                      desc: t('يُسند للوكيل الأقل مشغولاً — يتوقف عند الحد الأقصى', 'Goes to the least-loaded agent, stops at max limit'),
                    },
                  ].map(opt => (
                    <label key={opt.value}
                      className={`flex items-start gap-3 p-3 border rounded-lg cursor-pointer transition-colors ${
                        settings.mode === opt.value ? 'border-brand-400 bg-brand-50' : 'border-gray-200 hover:border-gray-300'
                      }`}>
                      <input type="radio" name="mode" value={opt.value} checked={settings.mode === opt.value}
                        onChange={() => !isDemo && setSettings(s => ({ ...s, mode: opt.value as any }))}
                        className="mt-0.5 accent-brand-600" disabled={isDemo} />
                      <div>
                        <p className="text-sm font-medium text-gray-800">{opt.label}</p>
                        <p className="text-xs text-gray-500 mt-0.5">{opt.desc}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              {/* Max per agent (capacity mode only) */}
              {settings.mode === 'capacity' && settings.enabled && (
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">
                    {t('الحد الأقصى للعملاء لكل وكيل', 'Max Leads Per Agent')}
                    <span className="text-gray-400 font-normal ml-1">({t('0 = غير محدود', '0 = unlimited')})</span>
                  </label>
                  <input type="number" min={0} max={500} value={settings.max_per_agent}
                    onChange={e => setSettings(s => ({ ...s, max_per_agent: parseInt(e.target.value) || 0 }))}
                    disabled={isDemo}
                    className={inputCls + (isDemo ? ' bg-gray-50 cursor-not-allowed' : '')} />
                  <p className="text-xs text-gray-400 mt-1">
                    {t('عندما يصل وكيل للحد، يُوزَّع العميل على التالي المتاح.', 'When an agent reaches the limit, the lead goes to the next available agent.')}
                  </p>
                </div>
              )}

              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              {saved && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'Settings saved')}</p>}

              <button type="submit" disabled={saving || isDemo}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                {saving ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ الإعدادات', 'Save Settings')}
              </button>
            </form>
          </div>

          {/* How it works info */}
          <div className="bg-white rounded-xl border border-gray-200 p-5 text-xs text-gray-500 space-y-2">
            <p className="font-semibold text-gray-700 text-sm">{t('كيف يعمل؟', 'How it works')}</p>
            <p>When a new lead arrives (via WhatsApp, BOS24 webhook, or the public form), the system automatically assigns it to the next eligible agent. Agents can still manually re-assign leads at any time.</p>
            <p className="pt-1"><strong className="text-gray-700">Round Robin:</strong> Agent A → B → C → A → B → ...</p>
            <p><strong className="text-gray-700">Capacity:</strong> Always picks the agent with the fewest active leads. Stops assigning to agents at their max limit.</p>
          </div>

        </div>
      </main>
    </div>
  )
}
