'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import type { Deal, PaginatedResult } from '@/types'
import clsx from 'clsx'

const stageColor: Record<string, string> = {
  open:   'bg-blue-100 text-blue-700',
  won:    'bg-green-100 text-green-700',
  lost:   'bg-red-100 text-red-700',
}

const stageLabel: Record<string, { en: string; ar: string }> = {
  open:   { en: 'Open',   ar: 'مفتوحة' },
  won:    { en: 'Won',    ar: 'مكسوبة' },
  lost:   { en: 'Lost',   ar: 'خاسرة' },
}

export default function DealsPage() {
  const [deals, setDeals] = useState<Deal[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [showForm, setShowForm] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newLeadId, setNewLeadId] = useState('')
  const [newTitle, setNewTitle] = useState('')
  const [newAmount, setNewAmount] = useState('')
  const [newCurrency, setNewCurrency] = useState('AED')
  const [newProbability, setNewProbability] = useState('50')
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const limit = 20

  useEffect(() => {
    setLoading(true)
    api.deals.list({ page, limit }).then((res: any) => {
      const data: Deal[] = res?.data ?? res
      setDeals(Array.isArray(data) ? data : [])
      setTotal(res?.total ?? 0)
    }).catch(() => setDeals([])).finally(() => setLoading(false))
  }, [page])

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newLeadId.trim() || !newTitle.trim()) return
    setCreating(true)
    try {
      await api.deals.create({
        lead_id: newLeadId.trim(),
        title: newTitle.trim(),
        amount: parseFloat(newAmount) || 0,
        currency: newCurrency,
        probability: parseInt(newProbability) || 50,
      })
      setShowForm(false)
      setNewLeadId('')
      setNewTitle('')
      setNewAmount('')
      setNewCurrency('AED')
      setNewProbability('50')
      setPage(1)
    } catch {
      alert(t('فشل إنشاء الصفقة', 'Failed to create deal'))
    } finally { setCreating(false) }
  }

  const formatAmount = (amount: number, currency: string) =>
    new Intl.NumberFormat(lang === 'ar' ? 'ar-AE' : 'en-AE', {
      style: 'currency', currency: currency || 'AED', maximumFractionDigits: 0,
    }).format(amount)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الصفقات', 'Deals')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : deals.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد صفقات بعد', 'No deals yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('أنشئ صفقة من صفحة خط الأنابيب', 'Create a deal from the Pipeline page')}</p>
            </div>
            <Link
              href="/pipeline"
              className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors"
            >
              {t('اذهب إلى خط الأنابيب', 'Go to Pipeline')}
            </Link>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{total} {t('صفقة', 'deals')}</p>
              {isAgent && (
                <button
                  onClick={() => setShowForm(true)}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors"
                >
                  {t('+ صفقة جديدة', '+ New Deal')}
                </button>
              )}
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('العنوان', 'Title')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('المرحلة', 'Stage')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('القيمة', 'Amount')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الاحتمالية', 'Probability')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('تاريخ الإغلاق', 'Close Date')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {deals.map((deal) => (
                    <tr key={deal.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-5 py-3.5 font-medium text-gray-900">{deal.title}</td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', stageColor[deal.stage])}>
                          {lang === 'ar' ? stageLabel[deal.stage]?.ar : stageLabel[deal.stage]?.en}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-gray-700 font-medium">
                        {deal.amount ? formatAmount(deal.amount, deal.currency) : '—'}
                      </td>
                      <td className="px-5 py-3.5">
                        <div className="flex items-center gap-2">
                          <div className="w-16 h-1.5 rounded-full bg-gray-100 overflow-hidden">
                            <div
                              className="h-full bg-brand-500 rounded-full"
                              style={{ width: `${deal.probability}%` }}
                            />
                          </div>
                          <span className="text-xs text-gray-400">{deal.probability}%</span>
                        </div>
                      </td>
                      <td className="px-5 py-3.5 text-gray-400 text-xs">
                        {deal.close_date
                          ? new Date(deal.close_date).toLocaleDateString(
                              lang === 'ar' ? 'ar-AE' : 'en-AE',
                              { year: 'numeric', month: 'short', day: 'numeric' }
                            )
                          : '—'}
                      </td>
                      <td className="px-5 py-3.5 text-end">
                        <Link
                          href={`/deals/${deal.id}`}
                          className="text-xs text-brand-600 hover:underline font-medium"
                        >
                          {t('عرض', 'View')}
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-6">
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50"
                >
                  {t('السابق', 'Prev')}
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(p => (
                  <button
                    key={p}
                    onClick={() => setPage(p)}
                    className={clsx(
                      'px-3 py-1.5 text-sm rounded-lg',
                      p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50'
                    )}
                  >
                    {p}
                  </button>
                ))}
                <button
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                  disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50"
                >
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Create Deal Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowForm(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('صفقة جديدة', 'New Deal')}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('معرف الفرصة', 'Lead ID')}</label>
                <input value={newLeadId} onChange={e => setNewLeadId(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                  placeholder="uuid" required />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('العنوان', 'Title')}</label>
                <input value={newTitle} onChange={e => setNewTitle(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                  placeholder={t('عنوان الصفقة', 'Deal title')} required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('المبلغ', 'Amount')}</label>
                  <input type="number" min="0" step="0.01" value={newAmount} onChange={e => setNewAmount(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('العملة', 'Currency')}</label>
                  <select value={newCurrency} onChange={e => setNewCurrency(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                    <option value="AED">AED</option>
                    <option value="USD">USD</option>
                    <option value="SAR">SAR</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('الاحتمالية %', 'Probability %')}</label>
                <input type="number" min="0" max="100" value={newProbability} onChange={e => setNewProbability(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={creating}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {creating ? '...' : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
