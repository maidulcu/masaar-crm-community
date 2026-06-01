'use client'

import { useEffect, useState } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import Link from 'next/link'

interface TenantPerformance {
  tenant_id: string
  tenant_name: string
  rental_history: number
  average_stay: number
  payment_on_time_rate: number
  dispute_count: number
  risk_score: number
  status: string
}

export default function TenantsAnalyticsPage() {
  const { lang } = useLang()
  const [tenants, setTenants] = useState<TenantPerformance[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const pageSize = 10

  const isArabic = lang === 'ar'

  const labels = {
    en: {
      tenants: 'Tenants',
      tenant: 'Tenant',
      status: 'Status',
      rentals: 'Rentals',
      avgStay: 'Avg Stay',
      payment: 'Payment Rate',
      disputes: 'Disputes',
      risk: 'Risk Score',
      days: 'days',
      percent: '%',
      active: 'Active',
      inactive: 'Inactive',
      lowRisk: 'Low Risk',
      mediumRisk: 'Medium Risk',
      highRisk: 'High Risk',
      loading: 'Loading...',
      noResults: 'No tenants found',
    },
    ar: {
      tenants: 'المستأجرون',
      tenant: 'المستأجر',
      status: 'الحالة',
      rentals: 'العقود',
      avgStay: 'متوسط الإقامة',
      payment: 'معدل الدفع',
      disputes: 'النزاعات',
      risk: 'درجة المخاطرة',
      days: 'أيام',
      percent: '%',
      active: 'نشط',
      inactive: 'غير نشط',
      lowRisk: 'خطر منخفض',
      mediumRisk: 'خطر متوسط',
      highRisk: 'خطر عالي',
      loading: 'جاري التحميل...',
      noResults: 'لم يتم العثور على مستأجرين',
    },
  }

  const t = labels[isArabic ? 'ar' : 'en']

  const getRiskColor = (score: number) => {
    if (score < 30) return { bg: 'bg-green-100 dark:bg-green-900', text: 'text-green-800 dark:text-green-200', label: t.lowRisk }
    if (score < 60) return { bg: 'bg-yellow-100 dark:bg-yellow-900', text: 'text-yellow-800 dark:text-yellow-200', label: t.mediumRisk }
    return { bg: 'bg-red-100 dark:bg-red-900', text: 'text-red-800 dark:text-red-200', label: t.highRisk }
  }

  useEffect(() => {
    const fetchTenants = async () => {
      try {
        setLoading(true)
        const res = (await api.analytics.listTenants({
          limit: pageSize,
          offset: (page - 1) * pageSize,
        })) as any
        setTenants((res.data || []) as TenantPerformance[])
        setTotal((res.meta?.total as number) || 0)
      } catch (error) {
        console.error('Failed to fetch tenants:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchTenants()
  }, [page])

  const totalPages = Math.ceil(total / pageSize)

  return (
    <div className="space-y-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-100 dark:bg-gray-700">
              <tr>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.tenant}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.status}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.rentals}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.avgStay}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.payment}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.disputes}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.risk}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {loading ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    {t.loading}
                  </td>
                </tr>
              ) : tenants.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    {t.noResults}
                  </td>
                </tr>
              ) : (
                tenants.map((tenant) => {
                  const riskColor = getRiskColor(tenant.risk_score)
                  return (
                    <tr key={tenant.tenant_id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                      <td className="px-6 py-4 font-medium text-gray-900 dark:text-white">
                        {tenant.tenant_name}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`px-3 py-1 rounded-full text-xs font-semibold ${
                            tenant.status === 'active'
                              ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
                              : 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200'
                          }`}
                        >
                          {tenant.status === 'active' ? t.active : t.inactive}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-gray-900 dark:text-white">
                        {tenant.rental_history}
                      </td>
                      <td className="px-6 py-4 text-gray-600 dark:text-gray-400">
                        {tenant.average_stay.toFixed(0)} {t.days}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2">
                          <div className="w-16 h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden">
                            <div
                              className={`h-full ${
                                tenant.payment_on_time_rate >= 80 ? 'bg-green-500' : 'bg-yellow-500'
                              }`}
                              style={{ width: `${Math.min(tenant.payment_on_time_rate, 100)}%` }}
                            />
                          </div>
                          <span className="text-xs font-semibold">
                            {tenant.payment_on_time_rate.toFixed(0)}{t.percent}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-gray-900 dark:text-white">
                        {tenant.dispute_count}
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`px-3 py-1 rounded-full text-xs font-semibold ${riskColor.bg} ${riskColor.text}`}
                        >
                          {tenant.risk_score} ({riskColor.label})
                        </span>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-6 py-4 bg-gray-50 dark:bg-gray-700 flex justify-between items-center">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-4 py-2 bg-gray-200 dark:bg-gray-600 text-gray-900 dark:text-white rounded hover:bg-gray-300 dark:hover:bg-gray-500 disabled:opacity-50"
            >
              ← Previous
            </button>
            <span className="text-gray-600 dark:text-gray-400 text-sm">
              {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages, page + 1))}
              disabled={page === totalPages}
              className="px-4 py-2 bg-gray-200 dark:bg-gray-600 text-gray-900 dark:text-white rounded hover:bg-gray-300 dark:hover:bg-gray-500 disabled:opacity-50"
            >
              Next →
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
