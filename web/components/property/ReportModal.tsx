'use client'

import { useState, useEffect } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface ReportModalProps {
  leadId?: string
  defaultClientName?: string
  onClose: () => void
}

export function ReportModal({ leadId, defaultClientName = '', onClose }: ReportModalProps) {
  const { t } = useLang()
  const [form, setForm] = useState({
    client_name:   defaultClientName,
    agent_name:    '',
    area:          '',
    area_slug:     '',
    property_type: '',
    bedrooms:      '',
    budget_min:    '',
    budget_max:    '',
    lat:           '',
    lng:           '',
    include_yield: true,
    include_pois:  false,
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  function set(field: string, value: string | boolean) {
    setForm(f => ({ ...f, [field]: value }))
  }

  async function handleGenerate() {
    if (!form.area.trim()) {
      setError(t('المنطقة مطلوبة', 'Area is required'))
      return
    }
    setLoading(true)
    setError('')

    try {
      const payload: Record<string, unknown> = {
        client_name:   form.client_name,
        agent_name:    form.agent_name,
        area:          form.area,
        area_slug:     form.area_slug || form.area.toLowerCase().replace(/\s+/g, '-'),
        property_type: form.property_type,
        bedrooms:      form.bedrooms,
        budget_min:    form.budget_min ? parseFloat(form.budget_min) : 0,
        budget_max:    form.budget_max ? parseFloat(form.budget_max) : 0,
        include_yield: form.include_yield,
        include_pois:  form.include_pois,
      }
      if (leadId) {
        payload.lead_id = leadId
      }
      if (form.lat && form.lng) {
        payload.lat = parseFloat(form.lat)
        payload.lng = parseFloat(form.lng)
      }

      const blob = await api.bos24.reportPdf(payload)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      const area = form.area.replace(/\s+/g, '-')
      a.download = `property-report-${area}-${new Date().toISOString().slice(0, 10)}.pdf`
      a.click()
      URL.revokeObjectURL(url)
      onClose()
    } catch (e: any) {
      setError(e.message || t('فشل إنشاء التقرير', 'Failed to generate report'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="report-modal-title"
        className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden"
      >

        {/* Header */}
        <div className="bg-[#1a3a5c] px-6 py-4 flex items-center justify-between">
          <div>
            <h2 id="report-modal-title" className="text-white font-semibold text-lg">
              {t('إنشاء تقرير العميل', 'Generate Client Report')}
            </h2>
            <p className="text-blue-200 text-xs mt-0.5">
              {t('تقرير بصيغة PDF مع بيانات دائرة الأراضي والأملاك', 'Branded PDF with DLD data, yield & amenities')}
            </p>
          </div>
          <button
            onClick={onClose}
            aria-label={t('إغلاق النافذة', 'Close modal')}
            className="text-blue-200 hover:text-white transition p-1 rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">

          {error && (
            <div role="alert" className="rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">
              {error}
            </div>
          )}

          {/* Client & agent */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('اسم العميل', 'Client name')}</label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.client_name}
                onChange={e => set('client_name', e.target.value)}
                placeholder="Ahmed Al-Rashid"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('اسم الوكيل', 'Agent name')}</label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.agent_name}
                onChange={e => set('agent_name', e.target.value)}
                placeholder="Your name"
              />
            </div>
          </div>

          {/* Area */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('المنطقة', 'Area')} <span className="text-red-500">*</span></label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.area}
                onChange={e => set('area', e.target.value)}
                placeholder="Dubai Marina"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('اسم المنطقة المبسط (لبيانات السوق)', 'Area slug (for market data)')}</label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.area_slug}
                onChange={e => set('area_slug', e.target.value)}
                placeholder="dubai-marina (auto if blank)"
              />
            </div>
          </div>

          {/* Property details */}
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('النوع', 'Type')}</label>
              <select
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.property_type}
                onChange={e => set('property_type', e.target.value)}
              >
                <option value="">{t('الكل', 'Any')}</option>
                <option value="Unit">{t('شقة / وحدة', 'Apartment / Unit')}</option>
                <option value="Villa">{t('فيلا', 'Villa')}</option>
                <option value="Townhouse">{t('تاون هاوس', 'Townhouse')}</option>
                <option value="Penthouse">{t('بنتهاوس', 'Penthouse')}</option>
                <option value="Land">{t('أرض', 'Land')}</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('غرف النوم', 'Bedrooms')}</label>
              <select
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.bedrooms}
                onChange={e => set('bedrooms', e.target.value)}
              >
                <option value="">{t('الكل', 'Any')}</option>
                <option value="Studio">{t('استوديو', 'Studio')}</option>
                <option value="1BR">1BR</option>
                <option value="2BR">2BR</option>
                <option value="3BR">3BR</option>
                <option value="4BR+">4BR+</option>
              </select>
            </div>
            <div className="col-span-1">
              <label className="block text-xs font-medium text-gray-600 mb-1">{t('الميزانية (درهم)', 'Budget (AED)')}</label>
              <div className="flex gap-1">
                <input
                  className="w-full border border-gray-200 rounded-lg px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={form.budget_min}
                  onChange={e => set('budget_min', e.target.value)}
                  placeholder={t('الأدنى', 'Min')}
                  type="number"
                />
                <input
                  className="w-full border border-gray-200 rounded-lg px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={form.budget_max}
                  onChange={e => set('budget_max', e.target.value)}
                  placeholder={t('الأقصى', 'Max')}
                  type="number"
                />
              </div>
            </div>
          </div>

          {/* Optional: coordinates for POIs */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">
              {t('إحداثيات العقار', 'Property coordinates')} <span className="text-gray-400 font-normal">({t('اختياري — لتحديد المرافق القريبة', 'optional — enables nearby amenities')})</span>
            </label>
            <div className="grid grid-cols-2 gap-3">
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.lat}
                onChange={e => set('lat', e.target.value)}
                placeholder="Latitude e.g. 25.0819"
                type="number"
                step="any"
              />
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.lng}
                onChange={e => set('lng', e.target.value)}
                placeholder="Longitude e.g. 55.1367"
                type="number"
                step="any"
              />
            </div>
          </div>

          {/* Sections toggle */}
          <div className="border border-gray-200 rounded-xl p-4 space-y-3">
            <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">{t('تضمين في التقرير', 'Include in report')}</p>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="w-4 h-4 rounded accent-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                checked={form.include_yield}
                onChange={e => set('include_yield', e.target.checked)}
              />
              <div>
                <div className="text-sm font-medium text-gray-800">{t('تحليل العائد الإيجاري', 'Rental yield analysis')}</div>
                <div className="text-xs text-gray-500">{t('بيانات إيجاري — العائد المتوقع للمستثمرين', 'Ejari data — estimated ROI for investor clients')}</div>
              </div>
            </label>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="w-4 h-4 rounded accent-blue-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
                checked={form.include_pois}
                onChange={e => set('include_pois', e.target.checked)}
              />
              <div>
                <div className="text-sm font-medium text-gray-800">{t('المرافق القريبة', 'Nearby amenities')}</div>
                <div className="text-xs text-gray-500">{t('المترو، المدارس، الموالات — يتطلب إحداثيات أعلاه', 'Metro, schools, malls — requires coordinates above')}</div>
              </div>
            </label>
          </div>

        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-100 flex items-center justify-between bg-gray-50">
          <p className="text-xs text-gray-400">
            {t('مصدر البيانات: دائرة الأراضي والأملاك عبر وقوف', 'Data sourced from DLD via Waqov')}
          </p>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              {t('إلغاء', 'Cancel')}
            </button>
            <button
              onClick={handleGenerate}
              disabled={loading}
              className="px-5 py-2 bg-[#1a3a5c] text-white text-sm font-semibold rounded-lg hover:bg-[#22487a] transition disabled:opacity-50 flex items-center gap-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  {t('جاري الإنشاء...', 'Generating…')}
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                  </svg>
                  {t('تنزيل PDF', 'Download PDF')}
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
