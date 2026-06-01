'use client'
import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'

interface PropertyResult {
  id: number
  building: string
  area: string
  property_type: string
  price: number
  price_per_sqm: number
  transaction_count?: number
  date?: string
}

interface TransactionFilters {
  area?: string
  property_type?: string
  trans_type?: string
  min_price?: number
  max_price?: number
  limit?: number
}

export default function PropertiesPage() {
  const { t } = useLang()

  // Search state
  const [searchQuery, setSearchQuery] = useState('')
  const [searchTab, setSearchTab] = useState<'search' | 'transactions'>('search')

  // Transaction filters
  const [filters, setFilters] = useState<TransactionFilters>({
    limit: 20
  })
  const [filterArea, setFilterArea] = useState('')
  const [filterType, setFilterType] = useState('')
  const [filterTransType, setFilterTransType] = useState('')
  const [filterMinPrice, setFilterMinPrice] = useState('')
  const [filterMaxPrice, setFilterMaxPrice] = useState('')

  // Results
  const [results, setResults] = useState<PropertyResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setResults([])

    if (!searchQuery.trim()) {
      setError(t('الرجاء إدخال بحث', 'Please enter a search query'))
      return
    }

    setLoading(true)
    try {
      const data = await api.bos24.search(searchQuery)
      setResults((data as any).results || [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setLoading(false)
    }
  }

  const handleTransactionSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setResults([])

    const searchFilters: TransactionFilters = {}
    if (filterArea) searchFilters.area = filterArea
    if (filterType) searchFilters.property_type = filterType
    if (filterTransType) searchFilters.trans_type = filterTransType
    if (filterMinPrice) searchFilters.min_price = parseFloat(filterMinPrice)
    if (filterMaxPrice) searchFilters.max_price = parseFloat(filterMaxPrice)

    setLoading(true)
    try {
      const data = await api.bos24.transactions(searchFilters as Record<string, string | number | undefined>)
      setResults((data as any).transactions || [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('بحث العقارات', 'Property Search')} />

      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-4xl space-y-6">

          {/* Tabs */}
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <div className="flex border-b border-gray-200">
              <button
                onClick={() => { setSearchTab('search'); setError(''); setResults([]); }}
                className={`flex-1 py-3 text-sm font-medium transition-colors ${
                  searchTab === 'search'
                    ? 'text-brand-600 border-b-2 border-brand-600 bg-brand-50'
                    : 'text-gray-600 hover:bg-gray-50'
                }`}
              >
                {t('البحث بالذكاء الاصطناعي', 'AI Search')}
              </button>
              <button
                onClick={() => { setSearchTab('transactions'); setError(''); setResults([]); }}
                className={`flex-1 py-3 text-sm font-medium transition-colors ${
                  searchTab === 'transactions'
                    ? 'text-brand-600 border-b-2 border-brand-600 bg-brand-50'
                    : 'text-gray-600 hover:bg-gray-50'
                }`}
              >
                {t('السجلات والمبيعات', 'Transactions')}
              </button>
            </div>

            {/* AI Search Tab */}
            {searchTab === 'search' && (
              <div className="p-6 space-y-4">
                <form onSubmit={handleSearch} className="space-y-3">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-2">
                      {t('اكتب ما تبحث عنه', 'Describe what you\'re looking for')}
                    </label>
                    <textarea
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder={t('مثال: شقة بغرفتي نوم في مارينا بسعر أقل من مليون درهم', 'Example: 2BR apartment in Marina under 1M AED')}
                      className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      rows={3}
                    />
                  </div>
                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
                  >
                    {loading ? t('جارٍ البحث...', 'Searching...') : t('بحث', 'Search')}
                  </button>
                </form>
              </div>
            )}

            {/* Transactions Tab */}
            {searchTab === 'transactions' && (
              <div className="p-6 space-y-4">
                <form onSubmit={handleTransactionSearch} className="space-y-4">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">
                        {t('المنطقة', 'Area')}
                      </label>
                      <input
                        type="text"
                        value={filterArea}
                        onChange={(e) => setFilterArea(e.target.value)}
                        placeholder={t('مثال: مارينا', 'e.g. Marina')}
                        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">
                        {t('نوع العقار', 'Property Type')}
                      </label>
                      <select
                        value={filterType}
                        onChange={(e) => setFilterType(e.target.value)}
                        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      >
                        <option value="">{t('الكل', 'All')}</option>
                        <option value="Unit">{t('شقة', 'Unit')}</option>
                        <option value="Villa">{t('فيلا', 'Villa')}</option>
                        <option value="Townhouse">{t('تاون هاوس', 'Townhouse')}</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">
                        {t('نوع المعاملة', 'Transaction Type')}
                      </label>
                      <select
                        value={filterTransType}
                        onChange={(e) => setFilterTransType(e.target.value)}
                        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      >
                        <option value="">{t('الكل', 'All')}</option>
                        <option value="Sell">{t('بيع', 'Sell')}</option>
                        <option value="Rent">{t('إيجار', 'Rent')}</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">
                        {t('السعر من', 'Min Price')}
                      </label>
                      <input
                        type="number"
                        value={filterMinPrice}
                        onChange={(e) => setFilterMinPrice(e.target.value)}
                        placeholder={t('مثال: 500000', 'e.g. 500000')}
                        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">
                        {t('السعر إلى', 'Max Price')}
                      </label>
                      <input
                        type="number"
                        value={filterMaxPrice}
                        onChange={(e) => setFilterMaxPrice(e.target.value)}
                        placeholder={t('مثال: 2000000', 'e.g. 2000000')}
                        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                      />
                    </div>
                  </div>
                  <button
                    type="submit"
                    disabled={loading}
                    className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
                  >
                    {loading ? t('جارٍ البحث...', 'Searching...') : t('بحث', 'Search')}
                  </button>
                </form>
              </div>
            )}
          </div>

          {/* Error Message */}
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-xl p-4">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          {/* Results */}
          {results.length > 0 && (
            <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
              <div className="p-4 border-b border-gray-200">
                <h3 className="text-sm font-semibold text-gray-800">
                  {t('النتائج', 'Results')} ({results.length})
                </h3>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-4 py-3 text-left font-semibold text-gray-600">{t('المبنى', 'Building')}</th>
                      <th className="px-4 py-3 text-left font-semibold text-gray-600">{t('المنطقة', 'Area')}</th>
                      <th className="px-4 py-3 text-left font-semibold text-gray-600">{t('النوع', 'Type')}</th>
                      <th className="px-4 py-3 text-right font-semibold text-gray-600">{t('السعر', 'Price')}</th>
                      <th className="px-4 py-3 text-right font-semibold text-gray-600">{t('السعر/م²', 'Price/m²')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {results.map((prop, idx) => (
                      <tr key={idx} className="border-b border-gray-100 hover:bg-gray-50">
                        <td className="px-4 py-3 text-gray-800 font-medium">{prop.building}</td>
                        <td className="px-4 py-3 text-gray-600">{prop.area}</td>
                        <td className="px-4 py-3 text-gray-600">{prop.property_type}</td>
                        <td className="px-4 py-3 text-right text-gray-800 font-medium">
                          {new Intl.NumberFormat('en-AE', {
                            style: 'currency',
                            currency: 'AED',
                            maximumFractionDigits: 0
                          }).format(prop.price)}
                        </td>
                        <td className="px-4 py-3 text-right text-gray-600">
                          {new Intl.NumberFormat('en-AE', {
                            maximumFractionDigits: 0
                          }).format(prop.price_per_sqm)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Empty State */}
          {!loading && results.length === 0 && !error && (
            <div className="bg-white rounded-xl border border-gray-200 p-12 text-center">
              <p className="text-gray-500">
                {searchTab === 'search'
                  ? t('ابدأ البحث عن العقارات', 'Start searching for properties')
                  : t('استخدم الفلاتر للبحث عن المعاملات', 'Use filters to search transactions')}
              </p>
            </div>
          )}

        </div>
      </main>
    </div>
  )
}
