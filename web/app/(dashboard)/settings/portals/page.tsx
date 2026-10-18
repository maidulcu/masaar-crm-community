'use client'
import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type PortalConnection, type PortalLinks } from '@/lib/api'
import { formatDateTime } from '@/lib/date'
import { leadSourceLabel } from '@/lib/leadSource'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const PORTALS = ['property_finder', 'bayut', 'dubizzle', 'portal']

export default function PortalSettingsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [conns, setConns] = useState<Record<string, PortalConnection>>({})
  const [links, setLinks] = useState<PortalLinks | null>(null)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api.portals.list()
      .then((r) => setConns(Object.fromEntries(r.connections.map((c) => [c.portal, c]))))
      .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [])

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    load()
  }, [user, router, load])

  if (user?.role !== 'admin') return null

  const run = async (portal: string, fn: () => Promise<unknown>) => {
    setBusy(portal); setError('')
    try { await fn(); load() } catch (err) { setError(err instanceof Error ? err.message : 'Failed') } finally { setBusy('') }
  }
  const connect = (portal: string) => {
    if (conns[portal] && !confirm(t('سيؤدي هذا إلى إبطال الروابط الحالية. هل تريد المتابعة؟', 'This replaces the current URLs; the portal must be updated with the new ones. Continue?'))) return
    run(portal, async () => setLinks(await api.portals.connect(portal)))
  }
  const disconnect = (portal: string) => {
    if (!confirm(t('ستتوقف الروابط عن العمل. هل تريد المتابعة؟', 'Both URLs stop working. Continue?'))) return
    run(portal, () => api.portals.disconnect(portal))
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('البوابات العقارية', 'Property Portals')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-4">
          <p className="text-sm text-gray-600">
            {t('اربط البوابة لاستقبال العملاء تلقائياً ولنشر إعلاناتك المنشورة كملف XML. لا تُدرج الإعلانات التي ينقصها تصريح إعلان ساري.',
              'Connect a portal to receive its leads automatically and to publish your published listings as an XML feed. Listings without a valid advertising permit are left out of the feed.')}
          </p>
          <p className="text-xs text-gray-500">
            {t('صيغ البوابات غير منشورة للعموم: يتعرف الرابط على أسماء الحقول الشائعة، وقد تحتاج البوابة إلى ربط الحقول من جهتها. راجع الوثائق في صفحة المطورين.',
              'Portal formats are not public: the lead URL recognises common field names, and the portal may need to map fields on its side. See the API docs for both formats.')}
          </p>
          {error && <p role="alert" className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}

          {links && (
            <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 space-y-3">
              <p className="text-sm font-medium text-amber-900">
                {t('انسخ الروابط الآن، لن تظهر مرة أخرى.', 'Copy these URLs now; they will not be shown again.')} ({leadSourceLabel(links.portal, t)})
              </p>
              {([[t('رابط استقبال العملاء (POST)', 'Lead webhook URL (POST)'), links.lead_webhook_url], [t('رابط ملف الإعلانات (GET)', 'Listing feed URL (GET)'), links.feed_url]] as const).map(([label, url]) => (
                <div key={label}>
                  <p className="text-xs text-amber-800 mb-1">{label}</p>
                  <div className="flex gap-2">
                    <input readOnly value={url} dir="ltr" className="flex-1 px-2 py-1.5 text-xs font-mono border border-amber-200 rounded bg-white" onFocus={(e) => e.target.select()} />
                    <button type="button" className="px-3 py-1.5 text-xs rounded bg-white border border-amber-300" onClick={() => navigator.clipboard?.writeText(url)}>
                      {t('نسخ', 'Copy')}
                    </button>
                  </div>
                </div>
              ))}
              <button type="button" className="text-xs text-amber-900 underline" onClick={() => setLinks(null)}>{t('تم', 'Done')}</button>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
            {PORTALS.map((p) => {
              const c = conns[p]
              return (
                <div key={p} className="flex flex-wrap items-center gap-3 px-4 py-3">
                  <div className="flex-1 min-w-[12rem]">
                    <p className="text-sm font-medium text-gray-900">{leadSourceLabel(p, t)}</p>
                    <p className="text-xs text-gray-500">
                      {!c ? t('غير متصل', 'Not connected')
                        : !c.enabled ? t('متوقف مؤقتاً', 'Paused')
                        : c.last_lead_at ? `${t('آخر عميل', 'Last lead')}: ${formatDateTime(c.last_lead_at)}`
                        : t('متصل، لم يصل أي عميل بعد', 'Connected, no leads yet')}
                    </p>
                  </div>
                  <button type="button" disabled={busy === p} onClick={() => connect(p)}
                    className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">
                    {c ? t('إنشاء روابط جديدة', 'New URLs') : t('اتصال', 'Connect')}
                  </button>
                  {c && (
                    <>
                      <button type="button" disabled={busy === p} onClick={() => run(p, () => api.portals.setEnabled(p, !c.enabled))}
                        className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 disabled:opacity-50">
                        {c.enabled ? t('إيقاف مؤقت', 'Pause') : t('استئناف', 'Resume')}
                      </button>
                      <button type="button" disabled={busy === p} onClick={() => disconnect(p)}
                        className="px-3 py-1.5 text-xs rounded-lg border border-red-200 text-red-600 disabled:opacity-50">
                        {t('فصل', 'Disconnect')}
                      </button>
                    </>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      </main>
    </div>
  )
}
