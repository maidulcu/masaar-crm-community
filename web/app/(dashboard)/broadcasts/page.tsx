'use client'
import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type Broadcast, type BroadcastSegment } from '@/lib/api'
import { formatDateTime } from '@/lib/date'
import { leadSourceLabel, MANUAL_SOURCES } from '@/lib/leadSource'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

const input = 'px-2.5 py-1.5 border border-gray-300 rounded-lg text-sm'
const STAGES = ['new', 'contacted', 'qualified', 'proposal', 'won', 'lost']

// WhatsApp broadcasts: an approved template to a segment of contacts who agreed to receive
// WhatsApp messages. Opted-out contacts and those who never agreed are skipped.
export default function BroadcastsPage() {
  const { user } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const [list, setList] = useState<Broadcast[]>([])
  const [form, setForm] = useState({ name: '', template_name: '', language: 'en', params: '{{first_name}}' })
  const [segment, setSegment] = useState<BroadcastSegment>({})
  const [preview, setPreview] = useState<{ total: number; eligible: number; no_consent: number; opted_out: number } | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => { api.broadcasts.list().then(setList).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed')) }, [])
  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    load()
  }, [user, router, load])
  // Refresh while something is sending.
  useEffect(() => {
    if (!list.some((b) => b.status === 'sending')) return
    const id = setInterval(load, 15000)
    return () => clearInterval(id)
  }, [list, load])
  useEffect(() => {
    if (user?.role !== 'admin') return
    api.broadcasts.preview(segment).then(setPreview).catch(() => setPreview(null))
  }, [segment, user])

  if (user?.role !== 'admin') return null

  const run = async (fn: () => Promise<unknown>) => {
    setError('')
    try { await fn(); load() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }
  const seg = (k: keyof BroadcastSegment) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setSegment((s) => ({ ...s, [k]: e.target.value || undefined }))

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('رسائل واتساب الجماعية', 'WhatsApp broadcasts')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-4">
          <form className="bg-white rounded-xl border border-gray-200 p-5 space-y-3" onSubmit={(e) => {
            e.preventDefault()
            run(async () => {
              await api.broadcasts.create({ ...form, params: form.params.split('\n').map((p) => p.trim()).filter(Boolean), segment })
              setForm({ ...form, name: '' })
            })
          }}>
            <h2 className="text-sm font-semibold text-gray-900">{t('رسالة جماعية جديدة', 'New broadcast')}</h2>
            <p className="text-xs text-gray-500">
              {t('تُرسل الرسائل الجماعية بقالب معتمد من Meta فقط، ولجهات الاتصال التي وافقت على تلقي رسائل واتساب. يمكن للمستلم الرد بـ «إيقاف» أو STOP لإلغاء الاشتراك.',
                'Broadcasts use a template approved by Meta, and only reach contacts who agreed to WhatsApp messages. Recipients can reply STOP or «إيقاف» to opt out.')}
            </p>
            <div className="grid sm:grid-cols-3 gap-3">
              <label className="text-xs text-gray-600">{t('الاسم', 'Name')}<input required className={`${input} block w-full mt-1`} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></label>
              <label className="text-xs text-gray-600">{t('اسم القالب المعتمد', 'Approved template name')}<input required dir="ltr" pattern="[a-z0-9_]+" className={`${input} block w-full mt-1`} value={form.template_name} onChange={(e) => setForm({ ...form, template_name: e.target.value })} /></label>
              <label className="text-xs text-gray-600">{t('لغة القالب', 'Template language')}<input required dir="ltr" className={`${input} block w-full mt-1`} value={form.language} onChange={(e) => setForm({ ...form, language: e.target.value })} /></label>
            </div>
            <label className="block text-xs text-gray-600">{t('متغيرات القالب (سطر لكل متغير؛ {{name}} و{{first_name}} تُستبدل باسم جهة الاتصال)', 'Template variables (one per line; {{name}} and {{first_name}} become the contact’s name)')}
              <textarea rows={3} dir="ltr" className={`${input} block w-full mt-1 font-mono`} value={form.params} onChange={(e) => setForm({ ...form, params: e.target.value })} />
            </label>
            <div className="grid sm:grid-cols-3 gap-3">
              <label className="text-xs text-gray-600">{t('وسم العميل', 'Lead tag')}<input className={`${input} block w-full mt-1`} value={segment.tag ?? ''} onChange={seg('tag')} /></label>
              <label className="text-xs text-gray-600">{t('مرحلة العميل', 'Lead stage')}
                <select className={`${input} block w-full mt-1`} value={segment.lead_stage ?? ''} onChange={seg('lead_stage')}>
                  <option value="">{t('الكل', 'Any')}</option>
                  {STAGES.map((s) => <option key={s} value={s}>{s}</option>)}
                </select>
              </label>
              <label className="text-xs text-gray-600">{t('المصدر', 'Source')}
                <select className={`${input} block w-full mt-1`} value={segment.source ?? ''} onChange={seg('source')}>
                  <option value="">{t('الكل', 'Any')}</option>
                  {MANUAL_SOURCES.map((s) => <option key={s} value={s}>{leadSourceLabel(s, t)}</option>)}
                </select>
              </label>
            </div>
            {preview && (
              <p className="text-xs text-gray-700">
                {t(`سيصل إلى ${preview.eligible} من ${preview.total} جهة اتصال. يُتجاوز ${preview.no_consent} بلا موافقة و${preview.opted_out} ألغوا الاشتراك.`,
                  `Reaches ${preview.eligible} of ${preview.total} contacts. ${preview.no_consent} without consent and ${preview.opted_out} who opted out are skipped.`)}
              </p>
            )}
            <button type="submit" className="px-4 py-2 text-sm rounded-lg bg-primary-600 text-white">{t('حفظ كمسودة', 'Save as draft')}</button>
          </form>
          {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}

          <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
            {list.length === 0 && <p className="p-6 text-center text-sm text-gray-400">{t('لا توجد رسائل جماعية', 'No broadcasts yet')}</p>}
            {list.map((b) => {
              const c = b.counts
              return (
                <div key={b.id} className="p-4 flex flex-wrap items-center gap-3">
                  <div className="flex-1 min-w-[14rem]">
                    <p className="text-sm font-medium text-gray-900">{b.name} <span className="text-xs text-gray-500 font-mono">{b.template_name}</span></p>
                    <p className="text-xs text-gray-500">
                      {formatDateTime(b.created_at)} · {b.status}
                      {b.status !== 'draft' && ` · ${t('أُرسل', 'sent')} ${(c.sent ?? 0) + (c.delivered ?? 0) + (c.read ?? 0)} · ${t('وصل', 'delivered')} ${(c.delivered ?? 0) + (c.read ?? 0)} · ${t('قُرئ', 'read')} ${c.read ?? 0} · ${t('فشل', 'failed')} ${c.failed ?? 0} · ${t('تُجوز', 'skipped')} ${c.skipped ?? 0} · ${t('بالانتظار', 'queued')} ${(c.pending ?? 0) + (c.sending ?? 0)}`}
                    </p>
                  </div>
                  {b.status === 'draft' && (
                    <button type="button" onClick={() => { if (confirm(t('بدء الإرسال؟', 'Start sending?'))) run(() => api.broadcasts.start(b.id)) }} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">{t('إرسال', 'Send')}</button>
                  )}
                  {(b.status === 'draft' || b.status === 'sending') && (
                    <button type="button" onClick={() => run(() => api.broadcasts.cancel(b.id))} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
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
