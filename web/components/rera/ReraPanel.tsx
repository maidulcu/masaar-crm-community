'use client'
import { useCallback, useEffect, useState } from 'react'
import { api, type ReraForm } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { formatDate } from '@/lib/date'

const NAMES: Record<string, { en: string; ar: string }> = {
  A: { en: 'Form A (seller/landlord)', ar: 'النموذج أ (البائع/المؤجر)' },
  B: { en: 'Form B (buyer/tenant)', ar: 'النموذج ب (المشتري/المستأجر)' },
  I: { en: 'Form I (broker to broker)', ar: 'النموذج I (بين الوسطاء)' },
  F: { en: 'Form F (MOU)', ar: 'النموذج و (مذكرة التفاهم)' },
}

interface Props {
  listingId?: string
  contactId?: string
  /** Form types that can be generated from this page. */
  types: ('A' | 'B' | 'I')[]
  canEdit: boolean
}

/** Lists, generates and sends RERA forms for a listing or contact. */
export default function ReraPanel({ listingId, contactId, types, canEdit }: Props) {
  const { t, lang } = useLang()
  const [forms, setForms] = useState<ReraForm[]>([])
  const [error, setError] = useState('')
  const [signFor, setSignFor] = useState<string | null>(null)
  const [signer, setSigner] = useState({ name: '', email: '' })
  const [signLink, setSignLink] = useState('')

  const load = useCallback(() => {
    api.rera.list({ listing_id: listingId, contact_id: contactId }).then(setForms).catch(() => {})
  }, [listingId, contactId])
  useEffect(() => { load() }, [load])

  const generate = async (type: string) => {
    setError('')
    try {
      await api.rera.create({ form_type: type, listing_id: type === 'B' ? undefined : listingId, contact_id: type === 'B' ? contactId : undefined })
      load()
    } catch (e: unknown) { setError(e instanceof Error ? e.message : 'Failed') }
  }
  const requestSignature = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!signFor) return
    setError('')
    try {
      const sig = await api.documents.requestSignature(signFor, signer.name, signer.email) as { id: string }
      setSignLink(`${window.location.origin}/sign/${sig.id}`)
      setSigner({ name: '', email: '' })
      load()
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  const name = (k: string) => (lang === 'ar' ? NAMES[k]?.ar : NAMES[k]?.en) ?? k
  const input = 'px-2.5 py-1.5 text-sm border border-gray-200 rounded-lg'

  return (
    <div>
      {canEdit && (
        <div className="flex flex-wrap gap-2 mb-3">
          {types.map(k => (
            <button key={k} onClick={() => generate(k)} className="px-3 py-1.5 text-xs font-medium border border-gray-200 rounded-lg hover:bg-gray-50">
              {t('إنشاء ', 'Generate ')}{name(k)}
            </button>
          ))}
        </div>
      )}
      {error && <p role="alert" className="text-xs text-red-600 mb-2">{error}</p>}
      {forms.length === 0 ? <p className="text-sm text-gray-400">{t('لا توجد نماذج بعد', 'No forms yet')}</p> : (
        <ul className="divide-y divide-gray-100">
          {forms.map(f => (
            <li key={f.document_id} className="py-2">
              <div className="flex flex-wrap items-center gap-3 text-sm">
                <span className="font-medium text-gray-900">{name(f.form_type)}</span>
                <span className="text-xs text-gray-500">{formatDate(f.created_at)}</span>
                <span className={'text-xs px-2 py-0.5 rounded-full ' + (f.signature_status === 'signed' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-600')}>
                  {f.signature_status === 'signed' ? t('موقع', 'Signed') : t('بانتظار التوقيع', 'Awaiting signature')}
                </span>
                <a href={`/rera-forms/${f.document_id}/print`} target="_blank" rel="noopener noreferrer" className="text-xs text-brand-600 hover:underline">{t('عرض / طباعة', 'View / print')}</a>
                {canEdit && f.signature_status !== 'signed' && (
                  <button onClick={() => { setSignFor(f.document_id); setSignLink('') }} className="text-xs text-brand-600 hover:underline">{t('طلب توقيع', 'Request signature')}</button>
                )}
              </div>
              {signFor === f.document_id && (
                <form onSubmit={requestSignature} className="flex flex-wrap gap-2 mt-2">
                  <input required aria-label={t('اسم الموقّع', 'Signer name')} placeholder={t('اسم الموقّع', 'Signer name')} className={input} value={signer.name} onChange={e => setSigner(s => ({ ...s, name: e.target.value }))} />
                  <input required type="email" dir="ltr" aria-label={t('بريد الموقّع', 'Signer email')} placeholder="email@example.com" className={input} value={signer.email} onChange={e => setSigner(s => ({ ...s, email: e.target.value }))} />
                  <button className="px-3 py-1.5 text-xs font-medium bg-brand-600 text-white rounded-lg">{t('إنشاء رابط', 'Create link')}</button>
                  {signLink && (
                    <p className="w-full text-xs text-gray-600">
                      {t('أرسل هذا الرابط للموقّع:', 'Send this link to the signer:')} <span dir="ltr" className="font-mono select-all">{signLink}</span>
                    </p>
                  )}
                </form>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
