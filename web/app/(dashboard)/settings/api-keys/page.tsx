'use client'
import { useState, useEffect } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'

interface ApiKey {
  id: string
  name: string
  key_prefix: string
  scopes: string
  created_at: string
  last_used_at: string | null
}

interface Scope {
  name: string
  description: string
}

// Arabic labels for the scopes the API offers; the English text comes from the API.
const SCOPE_AR: Record<string, string> = {
  'lead:create': 'إرسال عملاء محتملين إلى نقطة الاستقبال',
  'leads:read': 'قراءة العملاء المحتملين',
  'leads:write': 'إنشاء العملاء المحتملين وتحديث مراحلهم وملاحظاتهم',
  'contacts:read': 'قراءة جهات الاتصال',
  'contacts:write': 'إنشاء جهات الاتصال وتحديثها',
  'listings:read': 'قراءة العقارات المعروضة',
  'webhooks:manage': 'إدارة اشتراكات Webhook',
}

export default function ApiKeysPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [keys, setKeys] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(true)
  const [newKeyName, setNewKeyName] = useState('')
  const [scopes, setScopes] = useState<Scope[]>([])
  const [newKeyScopes, setNewKeyScopes] = useState<string[]>(['lead:create'])
  const [showNewKey, setShowNewKey] = useState(false)
  const [createdKey, setCreatedKey] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const loadKeys = async () => {
    try {
      const r = await api.settings.apiKeys.list() as ApiKey[] | null
      setKeys(r || [])
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    loadKeys()
    api.settings.apiKeys.scopes().then(s => setScopes(s || [])).catch(() => {})
  }, [user, router])

  const toggleScope = (name: string) =>
    setNewKeyScopes(cur => cur.includes(name) ? cur.filter(s => s !== name) : [...cur, name])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newKeyName.trim() || newKeyScopes.length === 0) return
    setSubmitting(true); setError(''); setCreatedKey('')
    try {
      const r = await api.settings.apiKeys.create({ name: newKeyName, scopes: newKeyScopes.join(',') }) as { plaintext: string }
      setCreatedKey(r.plaintext)
      setNewKeyName(''); setNewKeyScopes(['lead:create']); setShowNewKey(false)
      loadKeys()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create key')
    } finally { setSubmitting(false) }
  }

  const handleRevoke = async (id: string) => {
    if (!confirm(t('هل أنت متأكد من إلغاء هذا المفتاح؟', 'Revoke this API key? This cannot be undone.'))) return
    try { await api.settings.apiKeys.revoke(id); loadKeys() }
    catch {}
  }

  if (user?.role !== 'admin') return null

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('مفاتيح API', 'API Keys')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl space-y-6">
          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — API key management is read-only.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to create and manage API keys.
              </span>
            </div>
          )}
          <div className="flex items-center justify-between">
            <p className="text-xs text-gray-500">{t('مفاتيح API للتكامل الخارجي', 'API keys for external integrations')}</p>
            {!isDemo && (
              <button onClick={() => setShowNewKey(true)}
                className="px-4 py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 transition-colors">
                {t('مفتاح جديد', '+ New Key')}
              </button>
            )}
          </div>

          {showNewKey && (
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('إنشاء مفتاح API', 'Create API Key')}</h3>
              <form onSubmit={handleCreate} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('الاسم', 'Name')}</label>
                  <input value={newKeyName} onChange={e => setNewKeyName(e.target.value)}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                    placeholder={t('مثلاً: تكامل مع موقعي', 'e.g. Website integration')} />
                </div>
                <div>
                  <span className="block text-xs font-medium text-gray-600 mb-1">{t('الصلاحيات', 'Permissions')}</span>
                  <div className="space-y-1.5">
                    {scopes.map(s => (
                      <label key={s.name} className="flex items-start gap-2 text-xs text-gray-700 cursor-pointer">
                        <input type="checkbox" checked={newKeyScopes.includes(s.name)} onChange={() => toggleScope(s.name)}
                          className="mt-0.5 rounded border-gray-300 text-brand-600 focus:ring-brand-500" />
                        <span>
                          <code className="font-mono text-[11px] text-gray-800">{s.name}</code>
                          <span className="text-gray-500"> — {t(SCOPE_AR[s.name] || s.description, s.description)}</span>
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
                {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
                <div className="flex gap-2">
                  <button type="submit" disabled={submitting || newKeyScopes.length === 0}
                    className="flex-1 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50">
                    {submitting ? t('جارٍ الإنشاء...', 'Creating...') : t('إنشاء', 'Create Key')}
                  </button>
                  <button type="button" onClick={() => { setShowNewKey(false); setError('') }}
                    className="px-4 py-2.5 border border-gray-200 text-sm font-medium text-gray-600 rounded-lg hover:bg-gray-50">
                    {t('إلغاء', 'Cancel')}
                  </button>
                </div>
              </form>
            </div>
          )}

          {createdKey && (
            <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg">
              <p className="text-xs font-semibold text-amber-700 mb-1">
                {t('تم إنشاء المفتاح! انسخه الآن، لن تتمكن من رؤيته مرة أخرى.', 'Key created! Copy it now — you won\'t be able to see it again.')}
              </p>
              <code className="text-xs bg-white px-2 py-1 rounded border border-amber-200 block mt-1 break-all select-all font-mono">
                {createdKey}
              </code>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200">
            {loading ? (
              <div className="p-8 text-center text-xs text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : keys.length === 0 ? (
              <div className="p-8 text-center text-xs text-gray-400">{t('لا توجد مفاتيح API', 'No API keys yet')}</div>
            ) : (
              <div className="divide-y divide-gray-100">
                {keys.map(key => (
                  <div key={key.id} className="flex items-center justify-between p-4">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-gray-800">{key.name}</p>
                      <div className="flex items-center gap-2 mt-0.5">
                        <code className="text-xs text-gray-500 font-mono">{key.key_prefix}...</code>
                        <span className="text-[10px] text-gray-400">· {key.scopes}</span>
                      </div>
                      <p className="text-[10px] text-gray-400 mt-0.5">
                        {t('تاريخ الإنشاء:', 'Created:')} {new Date(key.created_at).toLocaleDateString()}
                        {key.last_used_at ? ` · ${t('آخر استخدام:', 'Last used:')} ${new Date(key.last_used_at).toLocaleDateString()}` : ''}
                      </p>
                    </div>
                    {!isDemo && (
                      <button onClick={() => handleRevoke(key.id)}
                        className="text-xs text-red-500 hover:text-red-700 font-medium px-3 py-1.5 rounded-lg hover:bg-red-50 transition-colors shrink-0">
                        {t('إلغاء', 'Revoke')}
                      </button>
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
