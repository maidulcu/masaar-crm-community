'use client'
import { useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useLocaleStore } from '@/store/locale'
import { formatMoney, currencyDecimals } from '@/lib/money'
import { VAT_CATEGORIES, VAT_CATEGORY_LABELS } from '@/lib/invoice'
import type { InvoiceLine, VATCategory } from '@/types'

interface Props {
  dealId: string
  currency: string
  defaultDescription: string
  defaultAmount?: number
  defaultCustomerName?: string
  onCreated: () => void
  onCancel: () => void
}

const round = (n: number, d: number) => Math.round(n * 10 ** d) / 10 ** d

/**
 * Issues an invoice for a deal. The totals shown are a preview; the server computes the
 * stored amounts with the same per-line rounding.
 */
export default function InvoiceForm({ dealId, currency, defaultDescription, defaultAmount, defaultCustomerName, onCreated, onCancel }: Props) {
  const { t, lang } = useLang()
  const rates = useLocaleStore((s) => s.locale.rules.vat_rates)
  const [lines, setLines] = useState<InvoiceLine[]>([
    { description: defaultDescription, quantity: 1, unit_price: defaultAmount ?? 0, vat_category: 'standard' },
  ])
  const [customer, setCustomer] = useState({ name: defaultCustomerName ?? '', address: '', trn: '' })
  const [supplyDate, setSupplyDate] = useState('')
  const [notes, setNotes] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const d = currencyDecimals(currency)
  const computed = lines.map((l) => {
    const net = round((l.quantity || 0) * (l.unit_price || 0), d)
    return { net, vat: round(net * (rates[l.vat_category] ?? 0), d) }
  })
  const subtotal = computed.reduce((s, c) => s + c.net, 0)
  const vat = computed.reduce((s, c) => s + c.vat, 0)

  const setLine = (i: number, patch: Partial<InvoiceLine>) =>
    setLines((ls) => ls.map((l, j) => (j === i ? { ...l, ...patch } : l)))

  const submit = async () => {
    setSaving(true); setError('')
    try {
      await api.invoices.create({
        deal_id: dealId,
        currency,
        supply_date: supplyDate || undefined,
        customer: customer.name || customer.address || customer.trn ? customer : undefined,
        lines: lines.map(({ description, quantity, unit_price, vat_category }) => ({ description, quantity, unit_price, vat_category })),
        notes,
      })
      onCreated()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally {
      setSaving(false)
    }
  }

  const input = 'w-full px-2.5 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-300 bg-white'
  const label = 'text-xs text-gray-500 mb-1 block'

  return (
    <div className="px-5 py-4 bg-gray-50 border-b border-gray-100 space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <div>
          <label htmlFor="inv-cust" className={label}>{t('اسم العميل', 'Customer name')}</label>
          <input id="inv-cust" className={input} value={customer.name} onChange={(e) => setCustomer({ ...customer, name: e.target.value })}
            placeholder={t('افتراضياً: جهة اتصال الصفقة', 'Defaults to the deal contact')} />
        </div>
        <div>
          <label htmlFor="inv-addr" className={label}>{t('عنوان العميل', 'Customer address')}</label>
          <input id="inv-addr" className={input} value={customer.address} onChange={(e) => setCustomer({ ...customer, address: e.target.value })} />
        </div>
        <div>
          <label htmlFor="inv-trn" className={label}>{t('الرقم الضريبي للعميل (للشركات)', 'Customer TRN (B2B)')}</label>
          <input id="inv-trn" dir="ltr" className={input} value={customer.trn} onChange={(e) => setCustomer({ ...customer, trn: e.target.value })} />
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-xs text-gray-500">
            <tr>
              <th scope="col" className="text-start font-medium pb-1">{t('الوصف', 'Description')}</th>
              <th scope="col" className="text-start font-medium pb-1 w-20">{t('الكمية', 'Qty')}</th>
              <th scope="col" className="text-start font-medium pb-1 w-32">{t('سعر الوحدة', 'Unit price')}</th>
              <th scope="col" className="text-start font-medium pb-1 w-44">{t('فئة الضريبة', 'VAT category')}</th>
              <th scope="col" className="text-end font-medium pb-1 w-32">{t('المبلغ', 'Net')}</th>
              <th scope="col" className="w-8"><span className="sr-only">{t('حذف', 'Remove')}</span></th>
            </tr>
          </thead>
          <tbody>
            {lines.map((l, i) => (
              <tr key={i}>
                <td className="pe-2 py-1"><input aria-label={t('الوصف', 'Description')} className={input} value={l.description} onChange={(e) => setLine(i, { description: e.target.value })} /></td>
                <td className="pe-2 py-1"><input aria-label={t('الكمية', 'Qty')} type="number" min={0} step="any" className={input} value={l.quantity} onChange={(e) => setLine(i, { quantity: parseFloat(e.target.value) || 0 })} /></td>
                <td className="pe-2 py-1"><input aria-label={t('سعر الوحدة', 'Unit price')} type="number" min={0} step="any" className={input} value={l.unit_price} onChange={(e) => setLine(i, { unit_price: parseFloat(e.target.value) || 0 })} /></td>
                <td className="pe-2 py-1">
                  <select aria-label={t('فئة الضريبة', 'VAT category')} className={input} value={l.vat_category} onChange={(e) => setLine(i, { vat_category: e.target.value as VATCategory })}>
                    {VAT_CATEGORIES.map((c) => (
                      <option key={c} value={c}>{lang === 'ar' ? VAT_CATEGORY_LABELS[c].ar : VAT_CATEGORY_LABELS[c].en} ({Math.round((rates[c] ?? 0) * 100)}%)</option>
                    ))}
                  </select>
                </td>
                <td className="py-1 text-end tabular-nums">{formatMoney(computed[i].net, currency)}</td>
                <td className="py-1 text-end">
                  {lines.length > 1 && (
                    <button type="button" onClick={() => setLines((ls) => ls.filter((_, j) => j !== i))}
                      aria-label={t('حذف السطر', 'Remove line')} className="text-gray-400 hover:text-red-600 px-1">✕</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <button type="button" onClick={() => setLines((ls) => [...ls, { description: '', quantity: 1, unit_price: 0, vat_category: 'standard' }])}
          className="mt-2 text-xs font-medium text-brand-600 hover:underline">
          {t('+ إضافة سطر', '+ Add line')}
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3 items-start">
        <div>
          <label htmlFor="inv-supply" className={label}>{t('تاريخ التوريد (اختياري)', 'Supply date (optional)')}</label>
          <input id="inv-supply" type="date" className={input} value={supplyDate} onChange={(e) => setSupplyDate(e.target.value)} />
        </div>
        <div>
          <label htmlFor="inv-notes" className={label}>{t('ملاحظات', 'Notes')}</label>
          <input id="inv-notes" className={input} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
        <dl className="text-sm grid grid-cols-2 gap-x-3">
          <dt className="text-gray-500">{t('المجموع', 'Subtotal')}</dt><dd className="text-end tabular-nums">{formatMoney(subtotal, currency)}</dd>
          <dt className="text-gray-500">{t('الضريبة', 'VAT')}</dt><dd className="text-end tabular-nums">{formatMoney(vat, currency)}</dd>
          <dt className="font-semibold">{t('الإجمالي', 'Total')}</dt><dd className="text-end tabular-nums font-semibold">{formatMoney(subtotal + vat, currency)}</dd>
        </dl>
      </div>

      {error && <p role="alert" className="text-xs text-red-600">{error}</p>}
      <div className="flex justify-end gap-3">
        <button type="button" onClick={onCancel} className="px-4 py-2 text-sm text-gray-500 hover:text-gray-700">{t('إلغاء', 'Cancel')}</button>
        <button type="button" onClick={submit} disabled={saving || subtotal + vat <= 0}
          className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50">
          {saving ? t('جاري الإصدار...', 'Issuing...') : t('إصدار الفاتورة', 'Issue invoice')}
        </button>
      </div>
    </div>
  )
}
