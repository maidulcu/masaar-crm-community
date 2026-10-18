'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import { api, type DealKYC } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { PAYMENT_KEYS, missingLabel, paymentLabel, reasonLabel } from './labels'

// On a deal: the buyer's KYC status (a deal is won only with complete KYC or an admin's
// override) and how the deal is paid, which may send it to the AML review queue.
export default function DealCompliancePanel({ dealId, isAdmin, canEdit }: { dealId: string; isAdmin: boolean; canEdit: boolean }) {
  const { t } = useLang()
  const [k, setK] = useState<DealKYC | null>(null)
  const [method, setMethod] = useState('')
  const [cash, setCash] = useState('')
  const [aml, setAml] = useState<{ status: string; reasons: string[] }>({ status: '', reasons: [] })
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api.kyc.deal(dealId).then((v) => {
      setK(v)
      setMethod(v.payment_method)
      setCash(v.cash_amount ? String(v.cash_amount) : '')
      setAml({ status: v.aml_status, reasons: v.aml_reasons })
    }).catch(() => setK(null))
  }, [dealId])
  useEffect(() => { load() }, [load])

  const run = async (fn: () => Promise<void>) => {
    setError('')
    try { await fn(); load() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  return (
    <div className="bg-white rounded-xl border border-gray-100 p-5 space-y-3">
      <h3 className="text-sm font-semibold text-gray-900">{t('الامتثال', 'Compliance')}</h3>
      {k && (
        <div className="text-sm">
          <Link href={`/contacts/${k.contact_id}`} className="text-primary-600 hover:underline">{k.contact_name}</Link>{' '}
          {k.missing.length === 0
            ? <span className="text-xs text-green-700">· {t('هوية العميل مكتملة', 'identity record complete')}</span>
            : <span className="text-xs text-amber-700">· {k.missing.map((m) => missingLabel(m, t)).join(', ')}</span>}
          {k.override_at && <p className="text-xs text-gray-500 mt-1">{t('تجاوز من المسؤول', 'Admin override')}: {k.override_reason}</p>}
          {!k.can_win && (
            <p className="text-xs text-gray-600 mt-1">
              {t('لا يمكن إغلاق الصفقة كرابحة قبل اكتمال هوية العميل.', 'The deal cannot be marked won until the client’s identity record is complete.')}
              {isAdmin && (
                <button type="button" className="ms-2 text-primary-600 hover:underline" onClick={() => {
                  const r = window.prompt(t('سبب التجاوز (سيُسجَّل)', 'Reason for the override (it is logged)'))
                  if (r) run(() => api.kyc.override(dealId, r))
                }}>{t('تجاوز', 'Override')}</button>
              )}
            </p>
          )}
        </div>
      )}
      <div className="flex flex-wrap items-end gap-2">
        <label className="text-xs text-gray-600">{t('طريقة الدفع', 'Payment method')}
          <select disabled={!canEdit} value={method} onChange={(e) => setMethod(e.target.value)} className="block mt-1 px-2 py-1.5 border border-gray-300 rounded-lg text-sm">
            {PAYMENT_KEYS.map((m) => <option key={m} value={m}>{paymentLabel(m, t)}</option>)}
          </select>
        </label>
        {(method === 'mixed' || method === 'cash') && (
          <label className="text-xs text-gray-600">{t('المبلغ النقدي', 'Cash part')}
            <input disabled={!canEdit} type="number" min={0} value={cash} onChange={(e) => setCash(e.target.value)} className="block mt-1 w-32 px-2 py-1.5 border border-gray-300 rounded-lg text-sm" />
          </label>
        )}
        {canEdit && (
          <button type="button" className="px-3 py-1.5 text-xs rounded-lg border border-gray-300" onClick={() => run(async () => {
            const r = await api.kyc.setPayment(dealId, method, Number(cash) || 0)
            setAml({ status: r.aml_status, reasons: r.aml_reasons })
          })}>{t('حفظ', 'Save')}</button>
        )}
      </div>
      {aml.status === 'flagged' && (
        <p className="text-xs text-amber-800 bg-amber-50 rounded p-2">
          {t('أُحيلت الصفقة لمراجعة مكافحة غسل الأموال', 'The deal was sent to the AML review queue')}
          {aml.reasons.length > 0 && `: ${aml.reasons.map((r) => reasonLabel(r, t)).join(', ')}`}
        </p>
      )}
      {(aml.status === 'reported' || aml.status === 'no_report') && (
        <p className="text-xs text-gray-600">{t('تمت مراجعة مكافحة غسل الأموال', 'AML review done')}</p>
      )}
      {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
    </div>
  )
}
