'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import clsx from 'clsx'
import { api, type EInvoiceParty, type EInvoiceStatus } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { formatDateTime } from '@/lib/date'
import type { VATInvoice } from '@/types'

// The UAE e-invoice (PINT AE) of a tax invoice or credit note (#88): the buyer data the
// tax invoice does not hold, an XML export, and submission to the company's provider.

const EMIRATES: [string, string, string][] = [
  ['AUH', 'أبوظبي', 'Abu Dhabi'], ['DXB', 'دبي', 'Dubai'], ['SHJ', 'الشارقة', 'Sharjah'],
  ['AJM', 'عجمان', 'Ajman'], ['UAQ', 'أم القيوين', 'Umm Al Quwain'], ['RAK', 'رأس الخيمة', 'Ras Al Khaimah'],
  ['FUJ', 'الفجيرة', 'Fujairah'],
]

const LEGAL_TYPES: [string, string, string][] = [
  ['TL', 'رخصة تجارية', 'Trade licence'], ['EID', 'الهوية الإماراتية', 'Emirates ID'],
  ['PAS', 'جواز السفر', 'Passport'], ['CD', 'قرار مجلس الوزراء', 'Cabinet decision'],
]

// UAE VAT Decree-Law Article 61(1) cases, as coded in PINT AE.
const CREDIT_REASONS: Record<string, [string, string]> = {
  'DL8.61.1.A': ['إلغاء التوريد', 'Supply cancelled'],
  'DL8.61.1.B': ['تغيير طبيعة التوريد', 'Nature of the supply changed'],
  'DL8.61.1.C': ['تعديل المقابل', 'Consideration altered'],
  'DL8.61.1.D': ['إعادة السلع أو الخدمات', 'Goods or services returned'],
  'DL8.61.1.E': ['خطأ في الفاتورة', 'Error in the invoice'],
  VD: ['خصم حجم', 'Volume discount'],
}

const STATUS: Record<string, { ar: string; en: string; cls: string }> = {
  submitted: { ar: 'قيد المعالجة', en: 'Submitted', cls: 'bg-blue-50 text-blue-700' },
  accepted: { ar: 'مقبولة', en: 'Accepted', cls: 'bg-green-50 text-green-700' },
  rejected: { ar: 'مرفوضة', en: 'Rejected', cls: 'bg-red-50 text-red-700' },
  error: { ar: 'تعذر الإرسال', en: 'Not delivered', cls: 'bg-amber-50 text-amber-700' },
}

const inputCls = 'w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500'
const labelCls = 'block text-xs font-medium text-gray-600 mb-1'

export function EInvoicePanel({ invoice, isAdmin }: { invoice: VATInvoice; isAdmin: boolean }) {
  const { t, lang } = useLang()
  const [state, setState] = useState<EInvoiceStatus | null>(null)
  const [open, setOpen] = useState(false)
  const [buyer, setBuyer] = useState<EInvoiceParty>({})
  const [creditReason, setCreditReason] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const isCredit = invoice.kind === 'credit_note'

  useEffect(() => {
    api.einvoice.status(invoice.id).then((s) => {
      setState(s)
      const prev = s.einvoice
      setBuyer({
        name: invoice.customer?.name ?? '', trn: invoice.customer?.trn ?? '', country: 'AE', legal_id_type: 'TL',
        ...(prev?.buyer ?? {}),
      })
      if (prev?.credit_reason) setCreditReason(prev.credit_reason)
      if (prev?.due_date) setDueDate(prev.due_date.slice(0, 10))
    }).catch(() => setState(null))
  }, [invoice.id, invoice.customer?.name, invoice.customer?.trn])

  if (!state || invoice.status === 'void') return null
  const rec = state.einvoice
  const enabled = state.provider !== 'off' && state.provider !== ''
  const canSubmit = isAdmin && enabled && (!rec || rec.status === 'rejected' || rec.status === 'error')

  const set = (k: keyof EInvoiceParty) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setBuyer((b) => ({ ...b, [k]: e.target.value }))
  const body = () => ({ buyer, credit_reason: isCredit ? creditReason : undefined, due_date: !isCredit && dueDate ? dueDate : undefined })

  const run = async (f: () => Promise<unknown>) => {
    setBusy(true)
    setError('')
    try {
      await f()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  const submit = () => run(async () => {
    try {
      await api.einvoice.submit(invoice.id, body())
    } finally {
      setState(await api.einvoice.status(invoice.id)) // a 502 still records the attempt
    }
    setOpen(false)
  })

  return (
    <div className="mt-6 bg-white rounded-2xl border border-gray-100 overflow-hidden">
      <div className="px-6 py-3 flex flex-wrap items-center gap-3 border-b border-gray-100">
        <h2 className="text-sm font-semibold text-gray-700">{t('الفاتورة الإلكترونية', 'E-invoice')}</h2>
        {rec && (
          <span className={clsx('px-2 py-0.5 rounded-full text-xs font-medium', STATUS[rec.status]?.cls)}>
            {t(STATUS[rec.status]?.ar ?? rec.status, STATUS[rec.status]?.en ?? rec.status)}
          </span>
        )}
        {!enabled && (
          <span className="text-xs text-gray-500">
            {t('الإرسال غير مفعّل — ', 'Submission is off — ')}
            {isAdmin
              ? <Link href="/settings/einvoicing" className="text-brand-600 hover:underline">{t('الإعدادات', 'settings')}</Link>
              : t('اطلب من المسؤول تفعيله', 'ask an admin to enable it')}
          </span>
        )}
        <div className="ms-auto flex gap-2">
          {rec && (
            <button type="button" className="text-xs text-brand-600 hover:underline"
              onClick={() => run(() => api.einvoice.storedXML(invoice.id, `${invoice.invoice_no}-einvoice.xml`))}>
              {t('تنزيل XML المرسل', 'Download submitted XML')}
            </button>
          )}
          <button type="button" className="text-xs text-brand-600 hover:underline" onClick={() => setOpen((o) => !o)}>
            {open ? t('إخفاء', 'Hide') : t('بيانات المشتري والتصدير', 'Buyer details & export')}
          </button>
        </div>
      </div>

      {rec && (
        <div className="px-6 py-3 text-xs text-gray-600 space-y-1">
          <p dir="ltr" className={lang === 'ar' ? 'text-end' : ''}>
            {rec.provider} · {rec.provider_ref || '—'} · {formatDateTime(rec.submitted_at)}
          </p>
          {rec.reason && <p role="alert" className="text-red-700">{rec.reason}</p>}
        </div>
      )}

      {open && (
        <div className="px-6 py-4 space-y-3 border-t border-gray-50">
          <p className="text-xs text-gray-500">
            {t('تتطلب الفاتورة الإلكترونية بيانات المشتري التالية بالإضافة إلى الفاتورة الضريبية. تحقق من المتطلبات مع مزود الخدمة المعتمد.',
              "An e-invoice needs these buyer details on top of the tax invoice. Check the requirements with your accredited service provider.")}
          </p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label htmlFor="ei-name" className={labelCls}>{t('اسم المشتري', 'Buyer name')}</label>
              <input id="ei-name" className={inputCls} value={buyer.name ?? ''} onChange={set('name')} />
            </div>
            <div>
              <label htmlFor="ei-tin" className={labelCls}>{t('رقم التعريف الضريبي (10 أرقام)', 'TIN (10 digits)')}</label>
              <input id="ei-tin" dir="ltr" inputMode="numeric" className={inputCls} value={buyer.tin ?? ''} onChange={set('tin')} />
            </div>
            <div>
              <label htmlFor="ei-trn" className={labelCls}>{t('رقم التسجيل الضريبي (إن وجد)', 'TRN (if VAT-registered)')}</label>
              <input id="ei-trn" dir="ltr" inputMode="numeric" className={inputCls} value={buyer.trn ?? ''} onChange={set('trn')} />
            </div>
            <div>
              <label htmlFor="ei-email" className={labelCls}>{t('البريد الإلكتروني', 'Email')}</label>
              <input id="ei-email" dir="ltr" type="email" className={inputCls} value={buyer.email ?? ''} onChange={set('email')} />
            </div>
            <div>
              <label htmlFor="ei-ltype" className={labelCls}>{t('نوع التسجيل القانوني', 'Legal registration')}</label>
              <select id="ei-ltype" className={inputCls} value={buyer.legal_id_type ?? 'TL'} onChange={set('legal_id_type')}>
                {LEGAL_TYPES.map(([v, ar, en]) => <option key={v} value={v}>{t(ar, en)}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="ei-lid" className={labelCls}>{t('رقم التسجيل', 'Registration number')}</label>
              <input id="ei-lid" dir="ltr" className={inputCls} value={buyer.legal_id ?? ''} onChange={set('legal_id')} />
            </div>
            {(buyer.legal_id_type === 'TL' || buyer.legal_id_type === 'PAS' || !buyer.legal_id_type) && (
              <div>
                <label htmlFor="ei-auth" className={labelCls}>
                  {buyer.legal_id_type === 'PAS' ? t('بلد الجواز', 'Passport country') : t('جهة الترخيص', 'Licensing authority')}
                </label>
                <input id="ei-auth" className={inputCls} value={buyer.legal_authority ?? ''} onChange={set('legal_authority')} />
              </div>
            )}
            <div>
              <label htmlFor="ei-street" className={labelCls}>{t('الشارع', 'Street')}</label>
              <input id="ei-street" className={inputCls} value={buyer.street ?? ''} onChange={set('street')} />
            </div>
            <div>
              <label htmlFor="ei-city" className={labelCls}>{t('المدينة', 'City')}</label>
              <input id="ei-city" className={inputCls} value={buyer.city ?? ''} onChange={set('city')} />
            </div>
            <div>
              <label htmlFor="ei-country" className={labelCls}>{t('الدولة (رمز ISO)', 'Country (ISO code)')}</label>
              <input id="ei-country" dir="ltr" maxLength={2} className={inputCls} value={buyer.country ?? ''} onChange={set('country')} />
            </div>
            {(buyer.country ?? 'AE').toUpperCase() === 'AE' && (
              <div>
                <label htmlFor="ei-emirate" className={labelCls}>{t('الإمارة', 'Emirate')}</label>
                <select id="ei-emirate" className={inputCls} value={buyer.emirate ?? ''} onChange={set('emirate')}>
                  <option value="">—</option>
                  {EMIRATES.map(([v, ar, en]) => <option key={v} value={v}>{t(ar, en)}</option>)}
                </select>
              </div>
            )}
            {isCredit ? (
              <div>
                <label htmlFor="ei-reason" className={labelCls}>{t('سبب الإشعار الدائن', 'Credit note reason')}</label>
                <select id="ei-reason" className={inputCls} value={creditReason} onChange={(e) => setCreditReason(e.target.value)}>
                  <option value="">—</option>
                  {state.credit_reasons.map((c) => (
                    <option key={c} value={c}>{CREDIT_REASONS[c] ? `${t(...CREDIT_REASONS[c])} (${c})` : c}</option>
                  ))}
                </select>
              </div>
            ) : (
              <div>
                <label htmlFor="ei-due" className={labelCls}>{t('تاريخ الاستحقاق (افتراضياً بعد 30 يوماً)', 'Due date (default: 30 days)')}</label>
                <input id="ei-due" type="date" className={inputCls} value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              </div>
            )}
          </div>
          <div className="flex flex-wrap gap-2 pt-1">
            <button type="button" disabled={busy}
              onClick={() => run(() => api.einvoice.exportXML(invoice.id, body(), `${invoice.invoice_no}.xml`))}
              className="px-4 py-2 text-sm font-medium border border-gray-200 rounded-lg hover:bg-gray-50 disabled:opacity-50">
              {t('تنزيل PINT AE XML', 'Download PINT AE XML')}
            </button>
            {canSubmit && (
              <button type="button" disabled={busy} onClick={submit}
                className="px-4 py-2 text-sm font-medium bg-gray-900 text-white rounded-lg disabled:opacity-50">
                {rec ? t('إعادة الإرسال', 'Submit again') : t('إرسال إلى المزود', 'Submit to provider')}
              </button>
            )}
          </div>
        </div>
      )}
      {error && <p role="alert" className="px-6 pb-4 text-xs text-red-600">{error}</p>}
    </div>
  )
}
