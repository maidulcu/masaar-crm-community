'use client'

import { useEffect, useState } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'

interface FinancialAnalytics {
  period: string
  total_revenue: number
  total_expenses: number
  net_profit: number
  profit_margin: number
  rent_collected: number
  rent_pending: number
  utilities_expense: number
  maintenance_expense: number
  other_expenses: number
}

export default function FinancialAnalyticsPage() {
  const { lang } = useLang()
  const [financial, setFinancial] = useState<FinancialAnalytics | null>(null)
  const [chartData, setChartData] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  const isArabic = lang === 'ar'

  const labels = {
    en: {
      financial: 'Financial Analysis',
      period: 'Period',
      revenue: 'Revenue',
      expenses: 'Expenses',
      profit: 'Net Profit',
      margin: 'Profit Margin',
      collected: 'Rent Collected',
      pending: 'Pending Payments',
      utilities: 'Utilities',
      maintenance: 'Maintenance',
      other: 'Other Expenses',
      aed: 'AED',
      percent: '%',
      filterLabel: 'Select Date Range',
      from: 'From',
      to: 'To',
      apply: 'Apply',
      loading: 'Loading...',
    },
    ar: {
      financial: 'التحليل المالي',
      period: 'الفترة',
      revenue: 'الإيرادات',
      expenses: 'المصاريف',
      profit: 'الربح الصافي',
      margin: 'هامش الربح',
      collected: 'الإيجار المحصل',
      pending: 'المدفوعات المعلقة',
      utilities: 'الخدمات',
      maintenance: 'الصيانة',
      other: 'مصاريف أخرى',
      aed: 'د.إ',
      percent: '%',
      filterLabel: 'اختر نطاق التاريخ',
      from: 'من',
      to: 'إلى',
      apply: 'تطبيق',
      loading: 'جاري التحميل...',
    },
  }

  const t = labels[isArabic ? 'ar' : 'en']

  useEffect(() => {
    const fetchFinancial = async () => {
      try {
        setLoading(true)
        const res = (await api.analytics.getFinancial(startDate, endDate)) as any
        setFinancial(res.data as FinancialAnalytics)

        // Prepare chart data
        const data: any[] = [
          {
            name: t.revenue,
            value: res.data.total_revenue,
            fill: '#10b981',
          },
          {
            name: t.expenses,
            value: res.data.total_expenses,
            fill: '#ef4444',
          },
          {
            name: t.profit,
            value: res.data.net_profit,
            fill: '#3b82f6',
          },
        ]

        setChartData(data)
      } catch (error) {
        console.error('Failed to fetch financial analytics:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchFinancial()
  }, [startDate, endDate])

  if (loading) {
    return <div className="text-center py-8">{t.loading}</div>
  }

  if (!financial) {
    return <div className="text-center py-8">No data available</div>
  }

  return (
    <div className="space-y-6">
      {/* Date Filter */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t.filterLabel}</h3>
        <div className="flex gap-4 flex-wrap">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t.from}
            </label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t.to}
            </label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white"
            />
          </div>
        </div>
      </div>

      {/* Main Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Revenue vs Expenses Summary */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Revenue vs Expenses</h3>
          <div className="space-y-3">
            {chartData.map((item: any) => (
              <div key={item.name}>
                <div className="flex justify-between items-center mb-1">
                  <span className="text-sm font-medium text-gray-600 dark:text-gray-400">{item.name}</span>
                  <span className="font-semibold text-gray-900 dark:text-white">
                    {item.value.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                  </span>
                </div>
                <div className="w-full h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                  <div
                    className="h-full transition-all duration-300"
                    style={{
                      backgroundColor: item.fill,
                      width: chartData.length > 0 ? `${(item.value / Math.max(...chartData.map((d: any) => d.value))) * 100}%` : '0%',
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Key Metrics Cards */}
        <div className="space-y-4">
          <div className="bg-gradient-to-br from-green-100 dark:from-green-900 to-green-50 dark:to-green-800 rounded-lg shadow p-6">
            <p className="text-gray-600 dark:text-gray-300 text-sm">{t.revenue}</p>
            <p className="text-3xl font-bold text-green-600 dark:text-green-400 mt-2">
              {financial.total_revenue.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
          </div>

          <div className="bg-gradient-to-br from-red-100 dark:from-red-900 to-red-50 dark:to-red-800 rounded-lg shadow p-6">
            <p className="text-gray-600 dark:text-gray-300 text-sm">{t.expenses}</p>
            <p className="text-3xl font-bold text-red-600 dark:text-red-400 mt-2">
              {financial.total_expenses.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
          </div>

          <div className="bg-gradient-to-br from-blue-100 dark:from-blue-900 to-blue-50 dark:to-blue-800 rounded-lg shadow p-6">
            <p className="text-gray-600 dark:text-gray-300 text-sm">{t.profit}</p>
            <p className="text-3xl font-bold text-blue-600 dark:text-blue-400 mt-2">
              {financial.net_profit.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">
              {t.margin}: {financial.profit_margin.toFixed(1)}{t.percent}
            </p>
          </div>
        </div>
      </div>

      {/* Expense Breakdown */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-6">Expense Breakdown</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border-l-4 border-blue-500 pl-4">
            <p className="text-gray-600 dark:text-gray-400 text-sm">{t.utilities}</p>
            <p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
              {financial.utilities_expense.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {((financial.utilities_expense / financial.total_expenses) * 100).toFixed(1)}{t.percent} of total
            </p>
          </div>

          <div className="border-l-4 border-orange-500 pl-4">
            <p className="text-gray-600 dark:text-gray-400 text-sm">{t.maintenance}</p>
            <p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
              {financial.maintenance_expense.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {((financial.maintenance_expense / financial.total_expenses) * 100).toFixed(1)}{t.percent} of total
            </p>
          </div>

          <div className="border-l-4 border-purple-500 pl-4">
            <p className="text-gray-600 dark:text-gray-400 text-sm">{t.other}</p>
            <p className="text-2xl font-bold text-gray-900 dark:text-white mt-1">
              {financial.other_expenses.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {((financial.other_expenses / financial.total_expenses) * 100).toFixed(1)}{t.percent} of total
            </p>
          </div>
        </div>
      </div>

      {/* Payment Status */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-6">Payment Status</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <p className="text-gray-600 dark:text-gray-400 text-sm mb-2">{t.collected}</p>
            <div className="w-full h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className="h-full bg-green-500"
                style={{
                  width: `${Math.min((financial.rent_collected / (financial.rent_collected + financial.rent_pending)) * 100, 100)}%`,
                }}
              />
            </div>
            <p className="text-sm font-semibold text-gray-900 dark:text-white mt-2">
              {financial.rent_collected.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
          </div>

          <div>
            <p className="text-gray-600 dark:text-gray-400 text-sm mb-2">{t.pending}</p>
            <div className="w-full h-4 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className="h-full bg-yellow-500"
                style={{
                  width: `${Math.min((financial.rent_pending / (financial.rent_collected + financial.rent_pending)) * 100, 100)}%`,
                }}
              />
            </div>
            <p className="text-sm font-semibold text-gray-900 dark:text-white mt-2">
              {financial.rent_pending.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
