'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import { api } from '@/lib/api'
import type { VATInvoice, CompanySettings } from '@/types'
import { formatMoney } from '@/lib/money'
import { VAT_CATEGORY_LABELS, documentTitle } from '@/lib/invoice'
import { safeUrl } from '@/lib/safeUrl'

// Printable, bilingual (English | Arabic) invoice. Use the browser's "Print → Save as PDF".
// The labels are printed in both languages side by side, as is usual for UAE tax invoices.

function Bi({ en, ar, className = '' }: { en: string; ar: string; className?: string }) {
  return (
    <span className={'inline-flex flex-wrap gap-x-2 ' + className}>
      <span>{en}</span>
      <span dir="rtl" lang="ar">{ar}</span>
    </span>
  )
}

function fmtDay(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export default function InvoicePrintPage() {
  const { id } = useParams<{ id: string }>()
  const [inv, setInv] = useState<VATInvoice | null>(null)
  const [original, setOriginal] = useState<VATInvoice | null>(null)
  const [branding, setBranding] = useState<Partial<CompanySettings>>({})
  const [error, setError] = useState('')

  useEffect(() => {
    api.invoices.get(id).then(async (d) => {
      const doc = d as VATInvoice
      setInv(doc)
      if (doc.original_invoice_id) {
        setOriginal(await api.invoices.get(doc.original_invoice_id) as VATInvoice)
      }
    }).catch((err: unknown) => setError(err instanceof Error ? err.message : 'Failed to load'))
    // Logo and colour are optional decoration; agents cannot read company settings.
    api.settings.getCompany().then((s) => setBranding(s as Partial<CompanySettings>)).catch(() => {})
  }, [id])

  if (error) return <p role="alert" className="p-8 text-sm text-red-600">{error}</p>
  if (!inv) return <p className="p-8 text-sm text-gray-400">Loading… جارٍ التحميل</p>

  const money = (n: number) => formatMoney(n, inv.currency)
  const title = documentTitle(inv.kind, inv.is_tax_invoice)
  // VAT summary per category, as the FTA expects amounts per rate to be visible.
  const byCategory = new Map<string, { net: number; vat: number; rate: number }>()
  for (const l of inv.lines ?? []) {
    const cur = byCategory.get(l.vat_category) ?? { net: 0, vat: 0, rate: l.vat_rate ?? 0 }
    cur.net += l.net_amount ?? 0
    cur.vat += l.vat_amount ?? 0
    byCategory.set(l.vat_category, cur)
  }

  return (
    <div className="flex-1 overflow-y-auto bg-gray-100 print:bg-white print:overflow-visible">
      <div className="max-w-[210mm] mx-auto my-6 print:my-0 bg-white shadow print:shadow-none p-10 print:p-0 text-[13px] text-gray-900">
        <div className="flex justify-end mb-4 print:hidden">
          <button onClick={() => window.print()} className="px-4 py-2 text-sm bg-gray-900 text-white rounded-lg">
            Print / Save as PDF · طباعة
          </button>
        </div>

        <header className="flex items-start justify-between gap-6 border-b-2 pb-4" style={{ borderColor: branding.primary_color || '#1a3a5c' }}>
          <div>
            {branding.logo_url && <img src={safeUrl(branding.logo_url)} alt="" className="h-12 mb-2 object-contain" />}
            <p className="font-semibold text-base">{inv.supplier.name}</p>
            <p className="whitespace-pre-line text-gray-600">{inv.supplier.address}</p>
            {inv.supplier.trn && <p className="text-gray-700" dir="ltr">TRN · الرقم الضريبي: {inv.supplier.trn}</p>}
          </div>
          <div className="text-end">
            <h1 className="text-xl font-bold"><Bi en={title.en} ar={title.ar} className="justify-end" /></h1>
            {inv.status === 'void' && <p className="text-red-600 font-semibold">VOID · ملغاة</p>}
          </div>
        </header>

        <section className="grid grid-cols-2 gap-6 py-4 border-b">
          <div>
            <p className="text-xs text-gray-500"><Bi en="Bill to" ar="فاتورة إلى" /></p>
            <p className="font-medium">{inv.customer.name}</p>
            {inv.customer.address && <p className="whitespace-pre-line text-gray-600">{inv.customer.address}</p>}
            {inv.customer.trn && <p dir="ltr">TRN · الرقم الضريبي: {inv.customer.trn}</p>}
          </div>
          <dl className="grid grid-cols-2 gap-x-3 gap-y-1">
            <dt className="text-gray-500"><Bi en="Number" ar="الرقم" /></dt><dd className="font-mono">{inv.invoice_no}</dd>
            <dt className="text-gray-500"><Bi en="Date" ar="التاريخ" /></dt><dd>{fmtDay(inv.issued_at)}</dd>
            {inv.supply_date && (<><dt className="text-gray-500"><Bi en="Supply date" ar="تاريخ التوريد" /></dt><dd>{fmtDay(inv.supply_date)}</dd></>)}
            <dt className="text-gray-500"><Bi en="Currency" ar="العملة" /></dt><dd>{inv.currency}</dd>
            {original && (<><dt className="text-gray-500"><Bi en="Original invoice" ar="الفاتورة الأصلية" /></dt><dd className="font-mono">{original.invoice_no} ({fmtDay(original.issued_at)})</dd></>)}
          </dl>
        </section>

        <table className="w-full mt-4 border-collapse">
          <thead>
            <tr className="bg-gray-50 text-xs text-gray-600">
              <th scope="col" className="p-2 text-start border"><Bi en="Description" ar="الوصف" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="Qty" ar="الكمية" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="Unit price" ar="سعر الوحدة" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="Net" ar="الصافي" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="VAT %" ar="نسبة الضريبة" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="VAT" ar="الضريبة" /></th>
              <th scope="col" className="p-2 text-end border"><Bi en="Total" ar="الإجمالي" /></th>
            </tr>
          </thead>
          <tbody>
            {(inv.lines ?? []).map((l) => (
              <tr key={l.id ?? l.position}>
                <td className="p-2 border">{l.description}</td>
                <td className="p-2 border text-end tabular-nums">{l.quantity}</td>
                <td className="p-2 border text-end tabular-nums">{money(l.unit_price)}</td>
                <td className="p-2 border text-end tabular-nums">{money(l.net_amount ?? 0)}</td>
                <td className="p-2 border text-end tabular-nums">{Math.round((l.vat_rate ?? 0) * 10000) / 100}%</td>
                <td className="p-2 border text-end tabular-nums">{money(l.vat_amount ?? 0)}</td>
                <td className="p-2 border text-end tabular-nums">{money((l.net_amount ?? 0) + (l.vat_amount ?? 0))}</td>
              </tr>
            ))}
          </tbody>
        </table>

        <section className="flex justify-between gap-6 mt-4">
          <table className="text-xs self-start">
            <thead>
              <tr className="text-gray-500">
                <th scope="col" className="pe-4 text-start font-normal"><Bi en="VAT category" ar="فئة الضريبة" /></th>
                <th scope="col" className="pe-4 text-end font-normal"><Bi en="Net" ar="الصافي" /></th>
                <th scope="col" className="text-end font-normal"><Bi en="VAT" ar="الضريبة" /></th>
              </tr>
            </thead>
            <tbody>
              {[...byCategory.entries()].map(([cat, v]) => {
                const label = VAT_CATEGORY_LABELS[cat as keyof typeof VAT_CATEGORY_LABELS]
                return (
                  <tr key={cat}>
                    <td className="pe-4"><Bi en={`${label?.en ?? cat} (${Math.round(v.rate * 10000) / 100}%)`} ar={label?.ar ?? cat} /></td>
                    <td className="pe-4 text-end tabular-nums">{money(v.net)}</td>
                    <td className="text-end tabular-nums">{money(v.vat)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          <dl className="grid grid-cols-2 gap-x-6 gap-y-1 min-w-[45%]">
            <dt><Bi en="Subtotal" ar="المجموع قبل الضريبة" /></dt><dd className="text-end tabular-nums">{money(inv.subtotal)}</dd>
            <dt><Bi en="VAT" ar="ضريبة القيمة المضافة" /></dt><dd className="text-end tabular-nums">{money(inv.vat_amount)}</dd>
            <dt className="font-bold border-t pt-1"><Bi en={`Total (${inv.currency})`} ar="الإجمالي" /></dt>
            <dd className="font-bold border-t pt-1 text-end tabular-nums">{money(inv.total)}</dd>
          </dl>
        </section>

        {(inv.reason || inv.notes) && (
          <section className="mt-6 text-xs text-gray-600 space-y-1">
            {inv.reason && <p><Bi en="Reason:" ar="السبب:" /> {inv.reason}</p>}
            {inv.notes && <p className="whitespace-pre-line">{inv.notes}</p>}
          </section>
        )}

        {(branding.bank_name || branding.bank_iban) && (
          <section className="mt-6 text-xs text-gray-600">
            <p className="font-medium"><Bi en="Bank details" ar="البيانات البنكية" /></p>
            <p>{branding.bank_name}</p>
            {branding.bank_iban && <p dir="ltr">IBAN {branding.bank_iban}</p>}
          </section>
        )}
      </div>
    </div>
  )
}
