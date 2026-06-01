'use client'

import { useEffect, useState } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import Link from 'next/link'

interface TenantAnalytics {
  total_tenants: number
  active_tenants: number
  inactive_tenants: number
  vacant_units: number
  occupied_units: number
  occupancy_rate: number
  average_rent_per_unit: number
  total_monthly_revenue: number
  collection_rate: number
  overdue_payments: number
  overdue_dues_amount: number
  upcoming_renewals: number
  tenant_churn_rate: number
}

interface PropertyAnalytics {
  property_id: string
  property_name: string
  property_type: string
  area: string
  total_units: number
  occupied_units: number
  vacant_units: number
  occupancy_rate: number
  monthly_revenue: number
  operating_expenses: number
  net_operating_income: number
  maintenance_needed: number
  active_leases: number
  expiring_leases: number
}

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

interface MaintenanceAnalytics {
  total_tasks: number
  completed_tasks: number
  pending_tasks: number
  avg_completion_days: number
  high_priority_tasks: number
  completion_rate: number
}

export default function AnalyticsPage() {
  const { lang } = useLang()
  const [tenantAnalytics, setTenantAnalytics] = useState<TenantAnalytics | null>(null)
  const [topProperties, setTopProperties] = useState<PropertyAnalytics[]>([])
  const [financialAnalytics, setFinancialAnalytics] = useState<FinancialAnalytics | null>(null)
  const [maintenanceAnalytics, setMaintenanceAnalytics] = useState<MaintenanceAnalytics | null>(null)
  const [loading, setLoading] = useState(true)

  const isArabic = lang === 'ar'

  const labels = {
    en: {
      overview: 'Analytics Overview',
      tenantMetrics: 'Tenant Metrics',
      propertyMetrics: 'Property Metrics',
      financialMetrics: 'Financial Metrics',
      maintenanceMetrics: 'Maintenance Metrics',
      totalTenants: 'Total Tenants',
      activeTenants: 'Active Tenants',
      occupancyRate: 'Occupancy Rate',
      avgRent: 'Avg Rent/Unit',
      totalRevenue: 'Total Revenue',
      collectionRate: 'Collection Rate',
      overduePayments: 'Overdue Payments',
      upcomingRenewals: 'Upcoming Renewals (60d)',
      churnRate: 'Churn Rate',
      properties: 'Properties',
      financial: 'Financial',
      maintenance: 'Maintenance',
      viewAll: 'View All',
      aed: 'AED',
      percent: '%',
      days: 'days',
      tasks: 'tasks',
      completionRate: 'Completion Rate',
      totalExpenses: 'Total Expenses',
      netProfit: 'Net Profit',
      rentCollected: 'Rent Collected',
      pendingPayments: 'Pending Payments',
    },
    ar: {
      overview: 'نظرة عامة على التحليلات',
      tenantMetrics: 'مقاييس المستأجرين',
      propertyMetrics: 'مقاييس الممتلكات',
      financialMetrics: 'المقاييس المالية',
      maintenanceMetrics: 'مقاييس الصيانة',
      totalTenants: 'إجمالي المستأجرين',
      activeTenants: 'المستأجرون النشطون',
      occupancyRate: 'معدل الاشغال',
      avgRent: 'متوسط الإيجار/وحدة',
      totalRevenue: 'إجمالي الإيرادات',
      collectionRate: 'معدل التحصيل',
      overduePayments: 'المدفوعات المتأخرة',
      upcomingRenewals: 'التجديدات القادمة (60)',
      churnRate: 'معدل الفقد',
      properties: 'الممتلكات',
      financial: 'المالي',
      maintenance: 'الصيانة',
      viewAll: 'عرض الكل',
      aed: 'د.إ',
      percent: '%',
      days: 'أيام',
      tasks: 'مهام',
      completionRate: 'معدل الإنجاز',
      totalExpenses: 'إجمالي النفقات',
      netProfit: 'الربح الصافي',
      rentCollected: 'الإيجار المحصل',
      pendingPayments: 'المدفوعات المعلقة',
    },
  }

  const t = labels[isArabic ? 'ar' : 'en']

  useEffect(() => {
    const fetchAnalytics = async () => {
      try {
        setLoading(true)
        const [tenantRes, propertiesRes, financialRes, maintenanceRes] = await Promise.all([
          api.analytics.getTenantOverview() as Promise<any>,
          api.analytics.listProperties({ limit: 5, offset: 0 }) as Promise<any>,
          api.analytics.getFinancial() as Promise<any>,
          api.analytics.getMaintenance() as Promise<any>,
        ])

        setTenantAnalytics(tenantRes.data as TenantAnalytics)
        setTopProperties((propertiesRes.data || []) as PropertyAnalytics[])
        setFinancialAnalytics(financialRes.data as FinancialAnalytics)
        setMaintenanceAnalytics(maintenanceRes.data as MaintenanceAnalytics)
      } catch (error) {
        console.error('Failed to fetch analytics:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchAnalytics()
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-32 bg-gray-200 dark:bg-gray-700 rounded-lg animate-pulse" />
        ))}
      </div>
    )
  }

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444']

  return (
    <div className="space-y-6">
      {/* Key Metrics Grid */}
      {tenantAnalytics && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Total Tenants */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-600 dark:text-gray-400 text-sm">{t.totalTenants}</p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
                  {tenantAnalytics.total_tenants}
                </p>
                <p className="text-xs text-green-600 dark:text-green-400 mt-2">
                  {tenantAnalytics.active_tenants} {t.activeTenants}
                </p>
              </div>
              <div className="text-4xl text-blue-500">👥</div>
            </div>
          </div>

          {/* Occupancy Rate */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-600 dark:text-gray-400 text-sm">{t.occupancyRate}</p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
                  {tenantAnalytics.occupancy_rate.toFixed(1)}{t.percent}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  {tenantAnalytics.occupied_units}/{tenantAnalytics.occupied_units + tenantAnalytics.vacant_units}
                </p>
              </div>
              <div className="text-4xl">🏢</div>
            </div>
          </div>

          {/* Total Revenue */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-600 dark:text-gray-400 text-sm">{t.totalRevenue}</p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
                  {tenantAnalytics.total_monthly_revenue.toLocaleString('en-AE', { maximumFractionDigits: 0 })}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">{t.aed}/month</p>
              </div>
              <div className="text-4xl">💰</div>
            </div>
          </div>

          {/* Collection Rate */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-600 dark:text-gray-400 text-sm">{t.collectionRate}</p>
                <p className="text-3xl font-bold text-gray-900 dark:text-white mt-2">
                  {tenantAnalytics.collection_rate.toFixed(1)}{t.percent}
                </p>
                <p className="text-xs text-red-600 dark:text-red-400 mt-2">
                  {tenantAnalytics.overdue_payments} {t.overduePayments}
                </p>
              </div>
              <div className="text-4xl">📊</div>
            </div>
          </div>
        </div>
      )}

      {/* Tenant & Maintenance Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Additional Tenant Metrics */}
        {tenantAnalytics && (
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t.tenantMetrics}</h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.avgRent}</span>
                <span className="font-semibold text-gray-900 dark:text-white">
                  {tenantAnalytics.average_rent_per_unit.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.upcomingRenewals}</span>
                <span className="font-semibold text-gray-900 dark:text-white">{tenantAnalytics.upcoming_renewals}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.churnRate}</span>
                <span className="font-semibold text-gray-900 dark:text-white">
                  {tenantAnalytics.tenant_churn_rate.toFixed(1)}{t.percent}
                </span>
              </div>
              <Link href="/analytics/tenants" className="text-blue-600 dark:text-blue-400 hover:underline text-sm mt-4">
                {t.viewAll} →
              </Link>
            </div>
          </div>
        )}

        {/* Maintenance Metrics */}
        {maintenanceAnalytics && (
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t.maintenanceMetrics}</h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">Total {t.tasks}</span>
                <span className="font-semibold text-gray-900 dark:text-white">{maintenanceAnalytics.total_tasks}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">Completed</span>
                <span className="font-semibold text-green-600 dark:text-green-400">
                  {maintenanceAnalytics.completed_tasks}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.completionRate}</span>
                <span className="font-semibold text-gray-900 dark:text-white">
                  {maintenanceAnalytics.completion_rate.toFixed(1)}{t.percent}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">Avg Completion</span>
                <span className="font-semibold text-gray-900 dark:text-white">
                  {maintenanceAnalytics.avg_completion_days.toFixed(1)} {t.days}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Financial & Properties Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Financial Metrics */}
        {financialAnalytics && (
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t.financialMetrics}</h3>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.rentCollected}</span>
                <span className="font-semibold text-green-600 dark:text-green-400">
                  {financialAnalytics.rent_collected.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.pendingPayments}</span>
                <span className="font-semibold text-yellow-600 dark:text-yellow-400">
                  {financialAnalytics.rent_pending.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.totalExpenses}</span>
                <span className="font-semibold text-red-600 dark:text-red-400">
                  {financialAnalytics.total_expenses.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-gray-600 dark:text-gray-400">{t.netProfit}</span>
                <span className="font-semibold text-gray-900 dark:text-white">
                  {financialAnalytics.net_profit.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                </span>
              </div>
              <Link href="/analytics/financial" className="text-blue-600 dark:text-blue-400 hover:underline text-sm mt-4">
                {t.viewAll} →
              </Link>
            </div>
          </div>
        )}

        {/* Top Properties */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">{t.properties}</h3>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {topProperties.length > 0 ? (
              topProperties.map((prop) => (
                <Link
                  key={prop.property_id}
                  href={`/analytics/properties/${prop.property_id}`}
                  className="block p-3 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition"
                >
                  <p className="font-medium text-gray-900 dark:text-white">{prop.property_name}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {prop.occupied_units}/{prop.total_units} {t.occupancyRate}: {prop.occupancy_rate.toFixed(1)}{t.percent}
                  </p>
                </Link>
              ))
            ) : (
              <p className="text-gray-500 dark:text-gray-400 text-sm">{t.properties}</p>
            )}
          </div>
          <Link href="/analytics/properties" className="text-blue-600 dark:text-blue-400 hover:underline text-sm mt-4">
            {t.viewAll} →
          </Link>
        </div>
      </div>
    </div>
  )
}
