'use client'
import { useCallback, useEffect, useState } from 'react'
import { api, type Consent } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { formatDateTime } from '@/lib/date'
import clsx from 'clsx'

const CHANNELS: { key: Consent['channel']; en: string; ar: string }[] = [
  { key: 'whatsapp', en: 'WhatsApp', ar: 'واتساب' },
  { key: 'email', en: 'Email', ar: 'البريد الإلكتروني' },
  { key: 'sms', en: 'SMS', ar: 'الرسائل النصية' },
  { key: 'phone', en: 'Phone calls', ar: 'المكالمات' },
]

interface Props {
  contactId: string
  canEdit: boolean
  isAdmin: boolean
  erased: boolean
  onErased: () => void
}

/** Consent per channel (PDPL), data export and erasure for one contact. */
export default function PrivacyPanel({ contactId, canEdit, isAdmin, erased, onErased }: Props) {
  const { t, lang } = useLang()
  const [current, setCurrent] = useState<Record<string, Consent>>({})
  const [history, setHistory] = useState<Consent[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [eraseOpen, setEraseOpen] = useState(false)
  const [eraseReason, setEraseReason] = useState('')
  const [eraseConfirm, setEraseConfirm] = useState('')

  const load = useCallback(async () => {
    try {
      const r = await api.privacy.consents(contactId)
      setCurrent(r.current ?? {})
      setHistory(r.history ?? [])
    } catch { /* panel stays empty */ }
  }, [contactId])

  useEffect(() => { load() }, [load])

  const act = async (fn: () => Promise<unknown>) => {
    setBusy(true); setError('')
    try { await fn(); await load() } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setBusy(false) }
  }

  return (
    <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-surface-900">{t('الموافقة والخصوصية', 'Consent & privacy')}</h3>
        {history.length > 0 && (
          <button onClick={() => setShowHistory(s => !s)} className="text-xs text-primary-600 hover:underline">
            {showHistory ? t('إخفاء السجل', 'Hide history') : t('عرض السجل', 'Show history')}
          </button>
        )}
      </div>

      <ul className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {CHANNELS.map(ch => {
          const c = current[ch.key]
          const status = c?.status
          return (
            <li key={ch.key} className="flex items-center justify-between gap-2 rounded-lg border border-surface-100 px-3 py-2">
              <div>
                <p className="text-sm text-surface-900">{lang === 'ar' ? ch.ar : ch.en}</p>
                <p className={clsx('text-xs', status === 'granted' ? 'text-green-700' : status === 'withdrawn' ? 'text-red-600' : 'text-surface-400')}>
                  {status === 'granted' ? t('موافق', 'Opted in') : status === 'withdrawn' ? t('رافض', 'Opted out') : t('لم يُحدد', 'No choice recorded')}
                </p>
              </div>
              {canEdit && !erased && (
                <div className="flex gap-1">
                  <button disabled={busy || status === 'granted'} onClick={() => act(() => api.privacy.recordConsent(contactId, ch.key, 'granted'))}
                    className="px-2 py-1 text-xs rounded border border-green-200 text-green-700 disabled:opacity-40">
                    {t('موافق', 'Opt in')}
                  </button>
                  <button disabled={busy || status === 'withdrawn'} onClick={() => act(() => api.privacy.recordConsent(contactId, ch.key, 'withdrawn'))}
                    className="px-2 py-1 text-xs rounded border border-red-200 text-red-700 disabled:opacity-40">
                    {t('رفض', 'Opt out')}
                  </button>
                </div>
              )}
            </li>
          )
        })}
      </ul>

      {showHistory && (
        <ul className="mt-3 text-xs text-surface-600 divide-y divide-surface-100">
          {history.map(h => (
            <li key={h.id} className="py-1.5 flex justify-between gap-3">
              <span>{h.channel} · {h.status} · {h.source}{h.note ? ` · ${h.note}` : ''}</span>
              <span className="text-surface-400 whitespace-nowrap">{formatDateTime(h.created_at)}</span>
            </li>
          ))}
        </ul>
      )}

      {isAdmin && (
        <div className="mt-4 pt-4 border-t border-surface-100 flex flex-wrap gap-2">
          <button disabled={busy} onClick={() => act(() => api.privacy.exportContact(contactId))}
            className="px-3 py-1.5 text-xs font-medium border border-surface-200 rounded-lg hover:bg-surface-50">
            {t('تصدير البيانات (طلب وصول)', 'Export data (access request)')}
          </button>
          {!erased && (
            <button onClick={() => setEraseOpen(o => !o)}
              className="px-3 py-1.5 text-xs font-medium border border-red-200 text-red-700 rounded-lg hover:bg-red-50">
              {t('محو البيانات الشخصية', 'Erase personal data')}
            </button>
          )}
        </div>
      )}

      {eraseOpen && (
        <div className="mt-3 rounded-lg bg-red-50 border border-red-200 p-3 space-y-2 text-sm">
          <p className="text-red-800">
            {t('لا يمكن التراجع عن هذا. ستُحذف البيانات الشخصية من جهة الاتصال والمحادثات والملاحظات والمرفقات. تبقى الفواتير الصادرة كما هي لأنها سجلات ضريبية.',
              'This cannot be undone. Personal data is removed from the contact, conversations, notes and attachments. Issued invoices are kept unchanged as tax records.')}
          </p>
          <input aria-label={t('السبب', 'Reason')} placeholder={t('السبب (مثلاً: طلب العميل)', 'Reason (e.g. customer request)')}
            className="w-full px-3 py-2 border border-red-200 rounded-lg text-sm bg-white" value={eraseReason} onChange={e => setEraseReason(e.target.value)} />
          <input aria-label={t('اكتب ERASE للتأكيد', 'Type ERASE to confirm')} placeholder={t('اكتب ERASE للتأكيد', 'Type ERASE to confirm')} dir="ltr"
            className="w-full px-3 py-2 border border-red-200 rounded-lg text-sm bg-white" value={eraseConfirm} onChange={e => setEraseConfirm(e.target.value)} />
          <button disabled={busy || eraseConfirm !== 'ERASE' || !eraseReason.trim()}
            onClick={() => act(async () => { await api.privacy.erase(contactId, eraseReason); setEraseOpen(false); onErased() })}
            className="px-3 py-2 text-xs font-medium bg-red-600 text-white rounded-lg disabled:opacity-40">
            {t('محو نهائي', 'Erase permanently')}
          </button>
        </div>
      )}
      {error && <p role="alert" className="mt-2 text-xs text-red-600">{error}</p>}
    </div>
  )
}
