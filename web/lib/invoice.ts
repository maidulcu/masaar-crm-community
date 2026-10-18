import type { InvoiceStatus, VATCategory, InvoiceKind } from '@/types'

export const INVOICE_STATUS_COLORS: Record<InvoiceStatus, string> = {
  draft: 'bg-gray-100 text-gray-600',
  sent: 'bg-blue-100 text-blue-700',
  paid: 'bg-green-100 text-green-700',
  void: 'bg-red-50 text-red-600 line-through',
}

export const INVOICE_STATUS_LABELS: Record<InvoiceStatus, { en: string; ar: string }> = {
  draft: { en: 'Draft', ar: 'مسودة' },
  sent: { en: 'Sent', ar: 'مُرسلة' },
  paid: { en: 'Paid', ar: 'مدفوعة' },
  void: { en: 'Void', ar: 'ملغاة' },
}

export const VAT_CATEGORY_LABELS: Record<VATCategory, { en: string; ar: string }> = {
  standard: { en: 'Standard rated', ar: 'خاضع للنسبة الأساسية' },
  zero_rated: { en: 'Zero rated', ar: 'خاضع لنسبة الصفر' },
  exempt: { en: 'Exempt', ar: 'معفى' },
  out_of_scope: { en: 'Out of scope', ar: 'خارج النطاق' },
}

export const VAT_CATEGORIES: VATCategory[] = ['standard', 'zero_rated', 'exempt', 'out_of_scope']

export const KIND_LABELS: Record<InvoiceKind, { en: string; ar: string }> = {
  invoice: { en: 'Invoice', ar: 'فاتورة' },
  credit_note: { en: 'Credit note', ar: 'إشعار دائن' },
}

/** Document title as printed: "Tax Invoice" only when the supplier had a TRN. */
export function documentTitle(kind: InvoiceKind, isTaxInvoice: boolean): { en: string; ar: string } {
  if (kind === 'credit_note') {
    return isTaxInvoice ? { en: 'Tax Credit Note', ar: 'إشعار دائن ضريبي' } : { en: 'Credit Note', ar: 'إشعار دائن' }
  }
  return isTaxInvoice ? { en: 'Tax Invoice', ar: 'فاتورة ضريبية' } : { en: 'Invoice', ar: 'فاتورة' }
}
