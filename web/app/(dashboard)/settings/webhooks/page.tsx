'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'

interface Webhook {
  id: string
  name: string
  url: string
  events: string[]
  is_active: boolean
  last_triggered_at: string | null
  created_at: string
}

const EVENT_OPTIONS = [
  { value: 'lead.created', label: 'Lead Created' },
  { value: 'lead.stage_changed', label: 'Lead Stage Changed' },
  { value: 'deal.created', label: 'Deal Created' },
  { value: 'deal.stage_changed', label: 'Deal Stage Changed' },
  { value: 'contact.created', label: 'Contact Created' },
  { value: 'invoice.created', label: 'Invoice Created' },
  { value: 'invoice.paid', label: 'Invoice Paid' },
]

export default function WebhooksPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [loading, setLoading] = useState(true)
  const [showNew, setShowNew] = useState(false)
  const [newName, setNewName] = useState('')
  const [newUrl, setNewUrl] = useState('')
  const [newEvents, setNewEvents] = useState<string[]>([])
  const [testingId, setTestingId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const loadWebhooks = async () => {
    try {
      const r = await api.settings.webhooks.list() as { data: Webhook[] }
      setWebhooks(r.data || [])
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    loadWebhooks()
  }, [user, router])

  const toggleEvent = (ev: string) => {
    setNewEvents(p => p.includes(ev) ? p.filter(e => e !== ev) : [...p, ev])
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim() || !newUrl.trim() || newEvents.length === 0) return
    setSubmitting(true); setError('')
    try {
      await api.settings.webhooks.create({ name: newName, url: newUrl, events: newEvents })
      setNewName(''); setNewUrl(''); setNewEvents([]); setShowNew(false)
      loadWebhooks()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create webhook')
    } finally { setSubmitting(false) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('حذف هذا الويب هوك؟', 'Delete this webhook?'))) return
    try { await api.settings.webhooks.delete(id); loadWebhooks() } catch {}
  }

  const handleTest = async (id: string) => {
    setTestingId(id); setTestResult(null)
    try {
      const r = await api.settings.webhooks.test(id) as { data: { status: number; body: string } }
      setTestResult(`HTTP ${r.data?.status ?? '—'}`)
    } catch (err: unknown) {
      setTestResult(err instanceof Error ? err.message : 'Failed')
    } finally { setTestingId(null); setTimeout(() => setTestResult(null), 5000) }
  }

  if (user?.role !== 'admin') return null

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('Webhooks', 'Webhooks')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl space-y-6">
          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — webhook management is read-only.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to configure webhooks.
              </span>
            </div>
          )}
          <div className="flex items-center justify-between">
            <p className="text-xs text-gray-500">{t('استقبال أحداث CRM عبر HTTP', 'Receive CRM events via HTTP callbacks')}</p>
            {!isDemo && (
              <button onClick={() => setShowNew(true)}
                className="px-4 py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 transition-colors">
                {t('إضافة', '+ Add Webhook')}
              </button>
            )}
          </div>

          {showNew && (
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('إضافة Webhook جديد', 'New Webhook')}</h3>
              <form onSubmit={handleCreate} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الاسم', 'Name')}</label>
                  <input value={newName} onChange={e => setNewName(e.target.value)} className={inputCls}
                    placeholder={t('مثلاً: تكامل مع نظامنا', 'e.g. Our system integration')} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('URL endpoint', 'URL Endpoint')}</label>
                  <input value={newUrl} onChange={e => setNewUrl(e.target.value)} className={inputCls}
                    placeholder="https://example.com/webhook" type="url" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-2">{t('الأحداث', 'Events')}</label>
                  <div className="flex flex-wrap gap-2">
                    {EVENT_OPTIONS.map(ev => (
                      <button key={ev.value} type="button" onClick={() => toggleEvent(ev.value)}
                        className={`text-xs px-3 py-1.5 rounded-full border font-medium transition-colors ${
                          newEvents.includes(ev.value)
                            ? 'bg-brand-600 text-white border-brand-600'
                            : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                        }`}>
                        {ev.label}
                      </button>
                    ))}
                  </div>
                  {newEvents.length === 0 && <p className="text-xs text-gray-400 mt-1">{t('اختر حدثاً واحداً على الأقل', 'Select at least one event')}</p>}
                </div>
                {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
                <div className="flex gap-2">
                  <button type="submit" disabled={submitting || newEvents.length === 0}
                    className="flex-1 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50">
                    {submitting ? t('جارٍ الإنشاء...', 'Creating...') : t('إنشاء', 'Create')}
                  </button>
                  <button type="button" onClick={() => { setShowNew(false); setError('') }}
                    className="px-4 py-2.5 border border-gray-200 text-sm font-medium text-gray-600 rounded-lg hover:bg-gray-50">
                    {t('إلغاء', 'Cancel')}
                  </button>
                </div>
              </form>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200">
            {loading ? (
              <div className="p-8 text-center text-xs text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : webhooks.length === 0 ? (
              <div className="p-8 text-center text-xs text-gray-400">{t('لا توجد Webhooks', 'No webhooks yet')}</div>
            ) : (
              <div className="divide-y divide-gray-100">
                {webhooks.map(wh => (
                  <div key={wh.id} className="p-4">
                    <div className="flex items-start justify-between">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-gray-800">{wh.name}</p>
                          <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${wh.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                            {wh.is_active ? t('نشط', 'Active') : t('غير نشط', 'Inactive')}
                          </span>
                        </div>
                        <code className="text-xs text-gray-500 font-mono block mt-0.5">{wh.url}</code>
                        <div className="flex flex-wrap gap-1 mt-1.5">
                          {wh.events.map(ev => (
                            <span key={ev} className="text-[10px] bg-gray-100 text-gray-600 px-1.5 py-0.5 rounded-full">{ev}</span>
                          ))}
                        </div>
                        <p className="text-[10px] text-gray-400 mt-1">
                          {t('تاريخ الإنشاء:', 'Created:')} {new Date(wh.created_at).toLocaleDateString()}
                          {wh.last_triggered_at ? ` · ${t('آخر تشغيل:', 'Last triggered:')} ${new Date(wh.last_triggered_at).toLocaleDateString()}` : ''}
                        </p>
                      </div>
                      {!isDemo && (
                        <div className="flex items-center gap-1 shrink-0 ml-3">
                          <button onClick={() => handleTest(wh.id)} disabled={testingId === wh.id}
                            className="text-xs text-gray-500 hover:text-brand-600 font-medium px-2 py-1.5 rounded-lg hover:bg-gray-50 transition-colors">
                            {testingId === wh.id ? '...' : t('اختبار', 'Test')}
                          </button>
                          <button onClick={() => handleDelete(wh.id)}
                            className="text-xs text-red-500 hover:text-red-700 font-medium px-2 py-1.5 rounded-lg hover:bg-red-50 transition-colors">
                            {t('حذف', 'Delete')}
                          </button>
                        </div>
                      )}
                    </div>
                    {testResult && (
                      <p className="text-xs text-green-600 mt-2">{t('نتيجة الاختبار:', 'Test result:')} {testResult}</p>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
