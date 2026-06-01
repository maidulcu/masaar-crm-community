'use client'

import { useEffect, useState } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import Link from 'next/link'

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

export default function PropertiesAnalyticsPage() {
  const { lang } = useLang()
  const [properties, setProperties] = useState<PropertyAnalytics[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const pageSize = 10

  const isArabic = lang === 'ar'

  const labels = {
    en: {
      properties: 'Properties',
      property: 'Property',
      type: 'Type',
      area: 'Area',
      units: 'Units',
      occupancy: 'Occupancy',
      revenue: 'Monthly Revenue',
      expenses: 'Operating Expenses',
      noi: 'Net Operating Income',
      maintenance: 'Maintenance Tasks',
      leases: 'Active Leases',
      expiring: 'Expiring (60d)',
      aed: 'AED',
      percent: '%',
      loading: 'Loading...',
      noResults: 'No properties found',
    },
    ar: {
      properties: 'الممتلكات',
      property: 'الممتلكة',
      type: 'النوع',
      area: 'المنطقة',
      units: 'الوحدات',
      occupancy: 'الاشغال',
      revenue: 'الإيرادات الشهرية',
      expenses: 'نفقات التشغيل',
      noi: 'صافي الدخل التشغيلي',
      maintenance: 'مهام الصيانة',
      leases: 'العقود النشطة',
      expiring: 'المنتهية (60)',
      aed: 'د.إ',
      percent: '%',
      loading: 'جاري التحميل...',
      noResults: 'لم يتم العثور على ممتلكات',
    },
  }

  const t = labels[isArabic ? 'ar' : 'en']

  useEffect(() => {
    const fetchProperties = async () => {
      try {
        setLoading(true)
        const res = (await api.analytics.listProperties({
          limit: pageSize,
          offset: (page - 1) * pageSize,
        })) as any
        setProperties((res.data || []) as PropertyAnalytics[])
        setTotal((res.meta?.total as number) || 0)
      } catch (error) {
        console.error('Failed to fetch properties:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchProperties()
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
                  {t.property}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.area}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.units}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.occupancy}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.revenue}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.noi}
                </th>
                <th className="px-6 py-3 text-left text-sm font-semibold text-gray-900 dark:text-white">
                  {t.maintenance}
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
              ) : properties.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-gray-500">
                    {t.noResults}
                  </td>
                </tr>
              ) : (
                properties.map((prop) => (
                  <tr
                    key={prop.property_id}
                    className="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
                  >
                    <td className="px-6 py-4">
                      <Link href={`/analytics/properties/${prop.property_id}`} className="text-blue-600 hover:underline">
                        {prop.property_name}
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-gray-600 dark:text-gray-400 text-sm">
                      {prop.area}
                    </td>
                    <td className="px-6 py-4 text-gray-900 dark:text-white">
                      {prop.occupied_units}/{prop.total_units}
                    </td>
                    <td className="px-6 py-4">
                      <div className="w-12 h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-blue-500"
                          style={{ width: `${prop.occupancy_rate}%` }}
                        />
                      </div>
                      <span className="text-xs text-gray-600 dark:text-gray-400">
                        {prop.occupancy_rate.toFixed(1)}{t.percent}
                      </span>
                    </td>
                    <td className="px-6 py-4 font-semibold text-green-600 dark:text-green-400">
                      {prop.monthly_revenue.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                    </td>
                    <td className="px-6 py-4 font-semibold text-gray-900 dark:text-white">
                      {prop.net_operating_income.toLocaleString('en-AE', { maximumFractionDigits: 0 })} {t.aed}
                    </td>
                    <td className="px-6 py-4">
                      {prop.maintenance_needed > 0 ? (
                        <span className="px-3 py-1 bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200 rounded-full text-xs font-semibold">
                          {prop.maintenance_needed}
                        </span>
                      ) : (
                        <span className="px-3 py-1 bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 rounded-full text-xs font-semibold">
                          ✓
                        </span>
                      )}
                    </td>
                  </tr>
                ))
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
