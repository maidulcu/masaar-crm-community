import { currentLocale } from '@/store/locale'

// Minor-unit digits per ISO 4217 code (mirrors internal/jurisdiction on the server).
const DECIMALS: Record<string, number> = {
  AED: 2, SAR: 2, QAR: 2, USD: 2, EUR: 2, GBP: 2, INR: 2, PKR: 2,
  BHD: 3, KWD: 3, OMR: 3, JOD: 3,
}

export function currencyDecimals(currency: string): number {
  return DECIMALS[currency?.toUpperCase()] ?? 2
}

/** The company's default currency (AED unless the company chose another). */
export function companyCurrency(): string {
  return currentLocale().currency || 'AED'
}

interface MoneyOptions {
  /** Drop the minor units (e.g. dashboards). Defaults to false. */
  whole?: boolean
  /** Compact notation, e.g. "AED 1.2M". */
  compact?: boolean
}

/**
 * Formats an amount with its currency code and the currency's own number of decimals
 * (2 for AED, 3 for BHD/KWD/OMR). Null and undefined render as "—".
 */
export function formatMoney(
  amount: number | string | null | undefined,
  currency?: string | null,
  opts: MoneyOptions = {},
): string {
  if (amount === null || amount === undefined || amount === '') return '—'
  const n = typeof amount === 'string' ? Number(amount) : amount
  if (!Number.isFinite(n)) return '—'
  const cur = (currency || companyCurrency()).toUpperCase()
  const digits = opts.whole || opts.compact ? 0 : currencyDecimals(cur)
  const num = new Intl.NumberFormat('en-AE', {
    minimumFractionDigits: digits,
    maximumFractionDigits: opts.compact ? 1 : digits,
    notation: opts.compact ? 'compact' : 'standard',
  }).format(n)
  return `${cur} ${num}`
}

/** Currencies offered in forms. The company's own currency is always listed first. */
export const CURRENCIES = ['AED', 'SAR', 'QAR', 'BHD', 'KWD', 'OMR', 'USD', 'EUR', 'GBP'] as const

export function currencyChoices(extra?: string | null): string[] {
  const first = companyCurrency()
  const list = [first, ...CURRENCIES.filter((c) => c !== first)]
  if (extra && !list.includes(extra)) list.push(extra)
  return list
}
