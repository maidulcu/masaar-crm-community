'use client'
import { Suspense, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { DEFAULT_RATES, commission, mortgage, purchaseCosts, rentalYield, type CalcRates } from '@/lib/calculators'
import { companyCurrency, formatMoney } from '@/lib/money'

const RATES_KEY = 'masaar.calculators.rates'

function loadRates(): CalcRates {
  try {
    const raw = localStorage.getItem(RATES_KEY)
    if (raw) return { ...DEFAULT_RATES, ...JSON.parse(raw) }
  } catch { /* storage unavailable */ }
  return DEFAULT_RATES
}

function num(v: string | null, fallback: number) {
  const n = Number(v)
  return v !== null && v !== '' && Number.isFinite(n) && n >= 0 ? n : fallback
}

const input = 'w-full mt-1 px-2.5 py-1.5 border border-gray-300 rounded-lg text-sm tabular-nums'

function Field({ label, value, onChange, step = 'any' }: { label: string; value: number; onChange: (n: number) => void; step?: string }) {
  return (
    <label className="block text-xs text-gray-600">{label}
      <input type="number" min={0} step={step} value={Number.isFinite(value) ? value : ''} className={input}
        onChange={(e) => onChange(e.target.value === '' ? 0 : Math.max(0, Number(e.target.value)))} />
    </label>
  )
}

function Calculators() {
  const { t } = useLang()
  const params = useSearchParams()
  const currency = (params.get('currency') || companyCurrency()).toUpperCase()
  const title = params.get('title') || ''
  const money = (n: number) => formatMoney(n, currency)

  const [rates, setRates] = useState<CalcRates>(DEFAULT_RATES)
  const [showRates, setShowRates] = useState(false)
  const [price, setPrice] = useState(() => num(params.get('price'), 2_000_000))
  const [downPct, setDownPct] = useState(25)
  const [interest, setInterest] = useState(4.5)
  const [years, setYears] = useState(25)
  const [financed, setFinanced] = useState(true)
  const [rent, setRent] = useState(() => num(params.get('rent'), Math.round(num(params.get('price'), 2_000_000) * 0.07)))
  const [runningCosts, setRunningCosts] = useState(20_000)
  const [commAmount, setCommAmount] = useState(() => num(params.get('amount') ?? params.get('price'), 2_000_000))
  const [commKind, setCommKind] = useState<'sale' | 'rent'>(params.get('type') === 'rent' ? 'rent' : 'sale')

  useEffect(() => { setRates(loadRates()) }, [])
  const setRate = (k: keyof CalcRates) => (n: number) => {
    const next = { ...rates, [k]: n }
    setRates(next)
    try { localStorage.setItem(RATES_KEY, JSON.stringify(next)) } catch { /* storage unavailable */ }
  }
  const resetRates = () => {
    setRates(DEFAULT_RATES)
    try { localStorage.removeItem(RATES_KEY) } catch { /* storage unavailable */ }
  }

  const loan = useMemo(() => mortgage(price, financed ? downPct : 100, interest, years), [price, financed, downPct, interest, years])
  const costs = useMemo(() => purchaseCosts(price, loan.loan, rates), [price, loan.loan, rates])
  const yld = useMemo(() => rentalYield(price, rent, runningCosts, costs.total), [price, rent, runningCosts, costs.total])
  const commRate = commKind === 'sale' ? rates.salesCommissionPct : rates.rentCommissionPct
  const comm = commission(commAmount, commRate, rates.vatPct)

  const summary = [
    title && title,
    `${t('السعر', 'Price')}: ${money(price)}`,
    ...costs.lines.map((l) => `${t(l.ar, l.en)}: ${money(l.amount)}`),
    `${t('إجمالي التكاليف', 'Total costs')}: ${money(costs.total)}`,
    `${t('المبلغ المطلوب نقداً', 'Cash needed')}: ${money(price - loan.loan + costs.total)}`,
    financed && loan.loan > 0 && `${t('القسط الشهري', 'Monthly payment')}: ${money(loan.monthly)} (${t('قرض', 'loan')} ${money(loan.loan)}, ${interest}% / ${years} ${t('سنة', 'years')})`,
    `${t('العائد الإجمالي / الصافي', 'Gross / net yield')}: ${yld.gross}% / ${yld.net}%`,
    t('أرقام تقديرية، يرجى التحقق منها.', 'Indicative figures; please verify.'),
  ].filter(Boolean).join('\n')

  const section = 'bg-white rounded-xl border border-gray-200 p-4 space-y-3 break-inside-avoid'
  const row = (label: string, value: string, strong = false) => (
    <div className={`flex justify-between gap-3 text-sm ${strong ? 'font-semibold text-gray-900 border-t border-gray-100 pt-2' : 'text-gray-700'}`}>
      <span>{label}</span><span className="tabular-nums whitespace-nowrap">{value}</span>
    </div>
  )

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div className="print:hidden"><Header title={t('الحاسبات', 'Calculators')} /></div>
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50 print:bg-white print:p-0">
        <div className="max-w-5xl space-y-4">
          {title && <h1 className="text-lg font-semibold text-gray-900">{title}</h1>}
          <p className="text-xs text-gray-500">
            {t('الرسوم الافتراضية تقديرية لدبي وقد تتغير. تحقق منها لدى دائرة الأراضي والأملاك والبنك، وعدّلها من "الرسوم والنسب". ليست استشارة مالية أو قانونية.',
              'Default fees are indicative for Dubai and change from time to time. Check them with the Dubai Land Department and the bank, and adjust them under "Fees and rates". Not financial or legal advice.')}
          </p>
          <div className="flex flex-wrap gap-2 print:hidden">
            <button type="button" onClick={() => window.print()} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('طباعة', 'Print')}</button>
            <a href={`https://wa.me/?text=${encodeURIComponent(summary)}`} target="_blank" rel="noopener noreferrer"
              className="px-3 py-1.5 text-xs rounded-lg border border-emerald-300 text-emerald-700 bg-white">{t('مشاركة عبر واتساب', 'Share on WhatsApp')}</a>
            <button type="button" onClick={() => navigator.clipboard?.writeText(summary)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('نسخ الملخص', 'Copy summary')}</button>
            <button type="button" onClick={() => setShowRates((s) => !s)} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300 bg-white">{t('الرسوم والنسب', 'Fees and rates')}</button>
          </div>

          {showRates && (
            <div className={`${section} print:hidden`}>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <Field label={t('رسوم النقل %', 'Transfer fee %')} value={rates.transferFeePct} onChange={setRate('transferFeePct')} />
                <Field label={t('رسوم الإدارة', 'Admin fee')} value={rates.adminFee} onChange={setRate('adminFee')} />
                <Field label={t('رسوم الأمين (مرتفعة)', 'Trustee fee (high)')} value={rates.trusteeFeeHigh} onChange={setRate('trusteeFeeHigh')} />
                <Field label={t('رسوم الأمين (منخفضة)', 'Trustee fee (low)')} value={rates.trusteeFeeLow} onChange={setRate('trusteeFeeLow')} />
                <Field label={t('حد رسوم الأمين', 'Trustee fee threshold')} value={rates.trusteeThreshold} onChange={setRate('trusteeThreshold')} />
                <Field label={t('تسجيل الرهن %', 'Mortgage registration %')} value={rates.mortgageRegPct} onChange={setRate('mortgageRegPct')} />
                <Field label={t('رسوم تسجيل الرهن', 'Mortgage registration fee')} value={rates.mortgageRegFee} onChange={setRate('mortgageRegFee')} />
                <Field label={t('رسوم ترتيب التمويل %', 'Bank arrangement %')} value={rates.bankArrangementPct} onChange={setRate('bankArrangementPct')} />
                <Field label={t('رسوم التقييم', 'Valuation fee')} value={rates.valuationFee} onChange={setRate('valuationFee')} />
                <Field label={t('عمولة البيع %', 'Sales commission %')} value={rates.salesCommissionPct} onChange={setRate('salesCommissionPct')} />
                <Field label={t('عمولة الإيجار %', 'Rental commission %')} value={rates.rentCommissionPct} onChange={setRate('rentCommissionPct')} />
                <Field label={t('ضريبة القيمة المضافة %', 'VAT %')} value={rates.vatPct} onChange={setRate('vatPct')} />
              </div>
              <button type="button" onClick={resetRates} className="text-xs text-gray-600 underline">{t('استعادة القيم الافتراضية', 'Reset to defaults')}</button>
            </div>
          )}

          <div className="grid md:grid-cols-2 gap-4">
            <div className={section}>
              <h2 className="text-sm font-semibold text-gray-900">{t('تكاليف الشراء', 'Purchase costs')}</h2>
              <div className="grid grid-cols-2 gap-3 print:hidden">
                <Field label={`${t('السعر', 'Price')} (${currency})`} value={price} onChange={setPrice} step="1000" />
                <label className="flex items-center gap-2 text-xs text-gray-600 mt-5">
                  <input type="checkbox" checked={financed} onChange={(e) => setFinanced(e.target.checked)} />
                  {t('بتمويل عقاري', 'With a mortgage')}
                </label>
              </div>
              {costs.lines.map((l) => row(t(l.ar, l.en), money(l.amount)))}
              {row(t('إجمالي التكاليف', 'Total costs'), money(costs.total), true)}
              {row(t('المبلغ المطلوب نقداً (الدفعة الأولى + التكاليف)', 'Cash needed (down payment + costs)'), money(price - loan.loan + costs.total), true)}
            </div>

            <div className={section}>
              <h2 className="text-sm font-semibold text-gray-900">{t('التمويل العقاري', 'Mortgage')}</h2>
              <div className="grid grid-cols-3 gap-3 print:hidden">
                <Field label={t('الدفعة الأولى %', 'Down payment %')} value={financed ? downPct : 100} onChange={(n) => setDownPct(Math.min(n, 100))} />
                <Field label={t('الفائدة السنوية %', 'Interest % / year')} value={interest} onChange={setInterest} step="0.05" />
                <Field label={t('المدة (سنوات)', 'Term (years)')} value={years} onChange={(n) => setYears(Math.min(n, 30))} step="1" />
              </div>
              {row(t('مبلغ القرض', 'Loan'), money(loan.loan))}
              {row(t('القسط الشهري', 'Monthly payment'), money(loan.monthly), true)}
              {row(t('إجمالي الفوائد', 'Total interest'), money(loan.totalInterest))}
              {row(t('إجمالي المدفوع', 'Total repaid'), money(loan.totalPaid))}
              <p className="text-[11px] text-gray-500">
                {t('يحدد المصرف المركزي حدوداً لنسبة التمويل حسب الإقامة وقيمة العقار؛ تحقق من الحد المطبق.',
                  'The Central Bank sets loan-to-value limits by residency and property value; check which limit applies.')}
              </p>
            </div>

            <div className={section}>
              <h2 className="text-sm font-semibold text-gray-900">{t('العائد الإيجاري', 'Rental yield')}</h2>
              <div className="grid grid-cols-2 gap-3 print:hidden">
                <Field label={`${t('الإيجار السنوي', 'Annual rent')} (${currency})`} value={rent} onChange={setRent} step="1000" />
                <Field label={t('رسوم الخدمات والصيانة سنوياً', 'Service charges + upkeep / year')} value={runningCosts} onChange={setRunningCosts} step="500" />
              </div>
              {row(t('العائد الإجمالي', 'Gross yield'), `${yld.gross}%`)}
              {row(t('صافي الدخل السنوي', 'Net income / year'), money(yld.netIncome))}
              {row(t('العائد الصافي (على السعر + التكاليف)', 'Net yield (on price + costs)'), `${yld.net}%`, true)}
            </div>

            <div className={section}>
              <h2 className="text-sm font-semibold text-gray-900">{t('العمولة', 'Commission')}</h2>
              <div className="grid grid-cols-2 gap-3 print:hidden">
                <label className="block text-xs text-gray-600">{t('النوع', 'Type')}
                  <select value={commKind} onChange={(e) => setCommKind(e.target.value as 'sale' | 'rent')} className={input}>
                    <option value="sale">{t('بيع (من السعر)', 'Sale (of the price)')}</option>
                    <option value="rent">{t('إيجار (من الإيجار السنوي)', 'Lease (of the annual rent)')}</option>
                  </select>
                </label>
                <Field label={`${t('المبلغ', 'Amount')} (${currency})`} value={commAmount} onChange={setCommAmount} step="1000" />
              </div>
              {row(`${t('العمولة', 'Commission')} (${commRate}%)`, money(comm.fee))}
              {row(`${t('ضريبة القيمة المضافة', 'VAT')} (${rates.vatPct}%)`, money(comm.vat))}
              {row(t('الإجمالي', 'Total'), money(comm.total), true)}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

export default function CalculatorsPage() {
  return <Suspense><Calculators /></Suspense>
}
