type Translate = (ar: string, en: string) => string

const DOC_TYPES: Record<string, [string, string]> = {
  emirates_id: ['الهوية الإماراتية', 'Emirates ID'],
  passport: ['جواز السفر', 'Passport'],
  residence_visa: ['تأشيرة الإقامة', 'Residence visa'],
  trade_licence: ['الرخصة التجارية', 'Trade licence'],
  other: ['أخرى', 'Other'],
}
const MISSING: Record<string, [string, string]> = {
  id_missing: ['لا توجد هوية إماراتية أو جواز سفر', 'No Emirates ID or passport on file'],
  id_expired: ['وثائق الهوية منتهية الصلاحية', 'The identity documents have expired'],
  id_no_expiry: ['تاريخ انتهاء الهوية غير مسجل', 'The ID has no expiry date recorded'],
  licence_missing: ['لا توجد رخصة تجارية سارية', 'No valid trade licence on file'],
}
const SCREENING: Record<string, [string, string]> = {
  not_screened: ['لم يُفحص', 'Not screened'],
  clear: ['لا تطابق', 'Clear'],
  possible_match: ['تطابق محتمل', 'Possible match'],
  match: ['تطابق', 'Match'],
}
const REASONS: Record<string, [string, string]> = {
  cash_at_or_above_threshold: ['نقد يساوي الحد أو يتجاوزه', 'Cash at or above the threshold'],
  virtual_asset: ['أصول افتراضية', 'Virtual assets'],
}
const PAY: Record<string, [string, string]> = {
  '': ['غير محدد', 'Not recorded'], cash: ['نقداً', 'Cash'], cheque: ['شيك', 'Cheque'], transfer: ['تحويل بنكي', 'Bank transfer'],
  virtual_asset: ['أصول افتراضية', 'Virtual assets'], mixed: ['مختلط', 'Mixed'],
}

const pick = (m: Record<string, [string, string]>) => (k: string, t: Translate) => (m[k] ? t(m[k][0], m[k][1]) : k)
export const docTypeLabel = pick(DOC_TYPES)
export const missingLabel = pick(MISSING)
export const screeningLabel = pick(SCREENING)
export const reasonLabel = pick(REASONS)
export const paymentLabel = pick(PAY)
export const DOC_TYPE_KEYS = Object.keys(DOC_TYPES)
export const SCREENING_KEYS = Object.keys(SCREENING)
export const PAYMENT_KEYS = Object.keys(PAY)
