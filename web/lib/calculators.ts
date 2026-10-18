// Property calculators. The default rates are indicative figures for Dubai and change from
// time to time: check them against the Dubai Land Department's published fees and your
// bank's offer, and adjust them in the calculator (they are saved in this browser).
// Nothing here is financial or legal advice.

export interface CalcRates {
  transferFeePct: number      // DLD transfer fee, % of the price
  adminFee: number            // DLD title deed / admin fee, flat
  trusteeFeeHigh: number      // registration trustee fee when the price is at or above trusteeThreshold
  trusteeFeeLow: number       // … below it
  trusteeThreshold: number
  mortgageRegPct: number      // mortgage registration, % of the loan
  mortgageRegFee: number      // … plus a flat fee
  bankArrangementPct: number  // bank arrangement fee, % of the loan
  valuationFee: number        // bank valuation, flat
  salesCommissionPct: number  // agency commission on a sale, % of the price
  rentCommissionPct: number   // agency commission on a lease, % of the annual rent
  vatPct: number              // VAT on fees and commission
}

export const DEFAULT_RATES: CalcRates = {
  transferFeePct: 4,
  adminFee: 580,
  trusteeFeeHigh: 4000,
  trusteeFeeLow: 2000,
  trusteeThreshold: 500_000,
  mortgageRegPct: 0.25,
  mortgageRegFee: 290,
  bankArrangementPct: 1,
  valuationFee: 3000,
  salesCommissionPct: 2,
  rentCommissionPct: 5,
  vatPct: 5,
}

const round2 = (n: number) => Math.round(n * 100) / 100

export interface CostLine { key: string; en: string; ar: string; amount: number }

/** Buyer's one-off costs for a purchase, with an optional mortgage. */
export function purchaseCosts(price: number, loan: number, r: CalcRates): { lines: CostLine[]; total: number } {
  const vat = r.vatPct / 100
  const lines: CostLine[] = [
    { key: 'transfer', en: `DLD transfer fee (${r.transferFeePct}%)`, ar: `رسوم نقل الملكية (${r.transferFeePct}%)`, amount: price * r.transferFeePct / 100 },
    { key: 'admin', en: 'Title deed / admin fee', ar: 'رسوم سند الملكية والإدارة', amount: r.adminFee },
  ]
  const trustee = price >= r.trusteeThreshold ? r.trusteeFeeHigh : r.trusteeFeeLow
  lines.push({ key: 'trustee', en: 'Registration trustee fee + VAT', ar: 'رسوم أمين التسجيل + الضريبة', amount: trustee * (1 + vat) })
  const commission = price * r.salesCommissionPct / 100
  lines.push({ key: 'commission', en: `Agency commission (${r.salesCommissionPct}%) + VAT`, ar: `عمولة الوساطة (${r.salesCommissionPct}%) + الضريبة`, amount: commission * (1 + vat) })
  if (loan > 0) {
    lines.push(
      { key: 'mortgage_reg', en: `Mortgage registration (${r.mortgageRegPct}% + fee)`, ar: `تسجيل الرهن (${r.mortgageRegPct}% + رسوم)`, amount: loan * r.mortgageRegPct / 100 + r.mortgageRegFee },
      { key: 'arrangement', en: `Bank arrangement fee (${r.bankArrangementPct}%) + VAT`, ar: `رسوم ترتيب التمويل (${r.bankArrangementPct}%) + الضريبة`, amount: loan * r.bankArrangementPct / 100 * (1 + vat) },
      { key: 'valuation', en: 'Valuation fee', ar: 'رسوم التقييم', amount: r.valuationFee },
    )
  }
  for (const l of lines) l.amount = round2(l.amount)
  return { lines, total: round2(lines.reduce((n, l) => n + l.amount, 0)) }
}

/** Commission on an amount, with VAT on top. */
export function commission(amount: number, ratePct: number, vatPct: number) {
  const fee = round2(amount * ratePct / 100)
  const vat = round2(fee * vatPct / 100)
  return { fee, vat, total: round2(fee + vat) }
}

/** Monthly repayment of an annuity loan. */
export function mortgage(price: number, downPct: number, ratePct: number, years: number) {
  const loan = round2(price * (1 - downPct / 100))
  const n = Math.round(years * 12)
  if (loan <= 0 || n <= 0) return { loan: Math.max(loan, 0), monthly: 0, totalInterest: 0, totalPaid: 0 }
  const i = ratePct / 100 / 12
  const monthly = i === 0 ? loan / n : (loan * i) / (1 - Math.pow(1 + i, -n))
  const totalPaid = monthly * n
  return { loan, monthly: round2(monthly), totalInterest: round2(totalPaid - loan), totalPaid: round2(totalPaid) }
}

/** Gross yield on the price; net yield after running costs, on the price plus purchase costs. */
export function rentalYield(price: number, annualRent: number, annualCosts: number, purchaseCostsTotal: number) {
  if (price <= 0) return { gross: 0, net: 0, netIncome: 0 }
  const netIncome = annualRent - annualCosts
  return {
    gross: round2(annualRent / price * 100),
    net: round2(netIncome / (price + purchaseCostsTotal) * 100),
    netIncome: round2(netIncome),
  }
}
