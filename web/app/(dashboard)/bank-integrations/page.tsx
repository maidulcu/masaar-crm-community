'use client'
import { useEffect, useState } from 'react'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { BankIntegration, PaginatedResult } from '@/types'

export default function BankIntegrationsPage() {
  const { t } = useLang()
  const [integrations, setIntegrations] = useState<BankIntegration[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const limit = 20

  useEffect(() => {
    loadIntegrations()
  }, [page])

  const loadIntegrations = async () => {
    setLoading(true)
    try {
      const result = (await api.bankIntegrations.list({ page, limit })) as PaginatedResult<BankIntegration>
      setIntegrations(result.data ?? [])
      setTotal(result.total ?? 0)
    } catch (err) {
      console.error('Failed to load bank integrations:', err)
      setIntegrations([])
    } finally {
      setLoading(false)
    }
  }

  const formatDate = (date: string | null) => {
    if (!date) return '-'
    return new Date(date).toLocaleDateString()
  }

  const getStatusColor = (connected: boolean) => {
    return connected ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'
  }

  const getStatusText = (connected: boolean) => {
    return t(connected ? 'متصل' : 'غير متصل', connected ? 'Connected' : 'Disconnected')
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">
        {t('جاري التحميل...', 'Loading...')}
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تكاملات البنك', 'Bank Integrations')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('تكاملات البنك', 'Bank Integrations')} ({total})
          </h2>
          <button
            onClick={() => alert(t('سيتم إضافة هذه الميزة قريباً', 'This feature will be available soon'))}
            className="px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 transition-colors"
          >
            + {t('إضافة بنك', 'Add Bank')}
          </button>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          {integrations.length === 0 ? (
            <div className="p-6 text-center text-gray-400">
              {t('لا توجد تكاملات بنكية حتى الآن', 'No bank integrations yet')}
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('اسم البنك', 'Bank Name')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('رقم الحساب', 'Account Number')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('نوع التكامل', 'Integration Type')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('المزامنة التلقائية', 'Auto Sync')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('آخر مزامنة', 'Last Sync')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('الحالة', 'Status')}</th>
                  <th className="px-4 py-3 text-left font-medium text-gray-700">{t('الإجراءات', 'Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {integrations.map((integration) => (
                  <tr key={integration.id} className="border-b border-gray-200 hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-3 text-gray-800 font-medium">{integration.bank_name}</td>
                    <td className="px-4 py-3 text-gray-600 text-xs font-mono">
                      {integration.account_number.slice(-4).padStart(integration.account_number.length, '*')}
                    </td>
                    <td className="px-4 py-3 text-gray-600 text-xs capitalize">{integration.integration_type}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${
                        integration.auto_sync ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-700'
                      }`}>
                        {t(integration.auto_sync ? 'مفعلة' : 'معطلة', integration.auto_sync ? 'Enabled' : 'Disabled')}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-600 text-xs">{formatDate(integration.last_sync_date)}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(integration.is_connected)}`}>
                        {getStatusText(integration.is_connected)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-gray-500">
                      <button
                        onClick={() => alert(t('سيتم إضافة هذه الميزة قريباً', 'This feature will be available soon'))}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        {t('تعديل', 'Edit')}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Pagination */}
        {total > limit && (
          <div className="flex items-center justify-between mt-4 text-sm text-gray-600">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="px-3 py-1 disabled:opacity-50 hover:bg-gray-100 rounded"
            >
              {t('السابق', 'Previous')}
            </button>
            <span>{t(`صفحة ${page}`, `Page ${page}`)}</span>
            <button
              onClick={() => setPage(p => p + 1)}
              disabled={page * limit >= total}
              className="px-3 py-1 disabled:opacity-50 hover:bg-gray-100 rounded"
            >
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
