'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'

interface PropertyDetail {
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

const AED = (n: number) =>
  n.toLocaleString('en-AE', { minimumFractionDigits: 0, maximumFractionDigits: 0 })

export default function PropertyDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { lang } = useLang()
  const ar = lang === 'ar'
  const [data, setData] = useState<PropertyDetail | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    ;(api.analytics.getProperty(id) as Promise<any>)
      .then((res: any) => setData(res.data ?? res))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [id])

  if (loading) {
    return (
      <div className="space-y-4 p-6">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-24 bg-gray-100 rounded-xl animate-pulse" />
        ))}
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-6 text-center text-gray-400 py-16">
        {ar ? 'لم يتم العثور على العقار' : 'Property not found'}
      </div>
    )
  }

  const occupancyColor =
    data.occupancy_rate >= 90 ? 'text-green-600' :
    data.occupancy_rate >= 70 ? 'text-yellow-600' : 'text-red-600'

  return (
    <div className="space-y-6 p-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <Link href="/analytics/properties" className="hover:text-brand-600">
          {ar ? 'العقارات' : 'Properties'}
        </Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">{data.property_name}</span>
      </div>

      {/* Header card */}
      <div className="bg-white border border-gray-100 rounded-xl p-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{data.property_name}</h1>
            <p className="text-gray-500 text-sm mt-1">
              {data.property_type} · {data.area}
            </p>
          </div>
          <span className={`text-3xl font-bold ${occupancyColor}`}>
            {data.occupancy_rate.toFixed(0)}%
            <span className="text-sm font-normal text-gray-400 ml-1">
              {ar ? 'إشغال' : 'occupancy'}
            </span>
          </span>
        </div>

        {/* Occupancy bar */}
        <div className="mt-4">
          <div className="flex justify-between text-xs text-gray-500 mb-1">
            <span>{data.occupied_units} {ar ? 'مشغول' : 'occupied'}</span>
            <span>{data.vacant_units} {ar ? 'شاغر' : 'vacant'}</span>
          </div>
          <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full ${
                data.occupancy_rate >= 90 ? 'bg-green-500' :
                data.occupancy_rate >= 70 ? 'bg-yellow-400' : 'bg-red-400'
              }`}
              style={{ width: `${data.occupancy_rate}%` }}
            />
          </div>
          <p className="text-xs text-gray-400 mt-1">
            {data.total_units} {ar ? 'إجمالي الوحدات' : 'total units'}
          </p>
        </div>
      </div>

      {/* KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          {
            label: ar ? 'الإيرادات الشهرية' : 'Monthly Revenue',
            value: `${AED(data.monthly_revenue)} AED`,
            color: 'bg-green-50 text-green-700',
          },
          {
            label: ar ? 'نفقات التشغيل' : 'Operating Expenses',
            value: `${AED(data.operating_expenses)} AED`,
            color: 'bg-red-50 text-red-700',
          },
          {
            label: ar ? 'صافي الدخل' : 'Net Operating Income',
            value: `${AED(data.net_operating_income)} AED`,
            color: data.net_operating_income >= 0 ? 'bg-blue-50 text-blue-700' : 'bg-red-50 text-red-700',
          },
          {
            label: ar ? 'تحتاج صيانة' : 'Maintenance Needed',
            value: String(data.maintenance_needed),
            color: data.maintenance_needed > 0 ? 'bg-orange-50 text-orange-700' : 'bg-gray-50 text-gray-700',
          },
        ].map((kpi) => (
          <div key={kpi.label} className={`rounded-xl p-4 ${kpi.color}`}>
            <p className="text-xs opacity-70 mb-1">{kpi.label}</p>
            <p className="text-lg font-bold">{kpi.value}</p>
          </div>
        ))}
      </div>

      {/* Lease status */}
      <div className="bg-white border border-gray-100 rounded-xl p-6">
        <h2 className="font-semibold text-gray-900 mb-4">
          {ar ? 'حالة عقود الإيجار' : 'Lease Status'}
        </h2>
        <div className="grid grid-cols-2 gap-6">
          <div className="text-center p-4 bg-blue-50 rounded-xl">
            <p className="text-3xl font-bold text-blue-700">{data.active_leases}</p>
            <p className="text-sm text-blue-600 mt-1">{ar ? 'عقود نشطة' : 'Active Leases'}</p>
          </div>
          <div className="text-center p-4 bg-amber-50 rounded-xl">
            <p className="text-3xl font-bold text-amber-700">{data.expiring_leases}</p>
            <p className="text-sm text-amber-600 mt-1">
              {ar ? 'تنتهي قريباً' : 'Expiring Soon'}
            </p>
          </div>
        </div>
        {data.expiring_leases > 0 && (
          <div className="mt-4 p-3 bg-amber-50 border border-amber-200 rounded-lg text-sm text-amber-800">
            ⚠ {data.expiring_leases} {ar
              ? 'عقد ينتهي خلال 60 يوماً — اتخذ إجراءات التجديد'
              : 'lease(s) expiring within 60 days — action needed'}
          </div>
        )}
      </div>
    </div>
  )
}
