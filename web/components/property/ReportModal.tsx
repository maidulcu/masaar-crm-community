'use client'

import { useState } from 'react'
import { api } from '@/lib/api'

interface ReportModalProps {
  leadId?: string
  defaultClientName?: string
  onClose: () => void
}

export function ReportModal({ leadId, defaultClientName = '', onClose }: ReportModalProps) {
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

  function set(field: string, value: string | boolean) {
    setForm(f => ({ ...f, [field]: value }))
  }

  async function handleGenerate() {
    if (!form.area.trim()) {
      setError('Area is required')
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
      setError(e.message || 'Failed to generate report')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden">

        {/* Header */}
        <div className="bg-[#1a3a5c] px-6 py-4 flex items-center justify-between">
          <div>
            <h2 className="text-white font-semibold text-lg">Generate Client Report</h2>
            <p className="text-blue-200 text-xs mt-0.5">Branded PDF with DLD data, yield &amp; amenities</p>
          </div>
          <button onClick={onClose} className="text-blue-200 hover:text-white transition">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">

          {error && (
            <div className="rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">
              {error}
            </div>
          )}

          {/* Client & agent */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Client name</label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.client_name}
                onChange={e => set('client_name', e.target.value)}
                placeholder="Ahmed Al-Rashid"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Agent name</label>
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
              <label className="block text-xs font-medium text-gray-600 mb-1">Area <span className="text-red-500">*</span></label>
              <input
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.area}
                onChange={e => set('area', e.target.value)}
                placeholder="Dubai Marina"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Area slug (for market data)</label>
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
              <label className="block text-xs font-medium text-gray-600 mb-1">Type</label>
              <select
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.property_type}
                onChange={e => set('property_type', e.target.value)}
              >
                <option value="">Any</option>
                <option value="Unit">Apartment / Unit</option>
                <option value="Villa">Villa</option>
                <option value="Townhouse">Townhouse</option>
                <option value="Penthouse">Penthouse</option>
                <option value="Land">Land</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Bedrooms</label>
              <select
                className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={form.bedrooms}
                onChange={e => set('bedrooms', e.target.value)}
              >
                <option value="">Any</option>
                <option value="Studio">Studio</option>
                <option value="1BR">1BR</option>
                <option value="2BR">2BR</option>
                <option value="3BR">3BR</option>
                <option value="4BR+">4BR+</option>
              </select>
            </div>
            <div className="col-span-1">
              <label className="block text-xs font-medium text-gray-600 mb-1">Budget (AED)</label>
              <div className="flex gap-1">
                <input
                  className="w-full border border-gray-200 rounded-lg px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={form.budget_min}
                  onChange={e => set('budget_min', e.target.value)}
                  placeholder="Min"
                  type="number"
                />
                <input
                  className="w-full border border-gray-200 rounded-lg px-2 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={form.budget_max}
                  onChange={e => set('budget_max', e.target.value)}
                  placeholder="Max"
                  type="number"
                />
              </div>
            </div>
          </div>

          {/* Optional: coordinates for POIs */}
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">
              Property coordinates <span className="text-gray-400 font-normal">(optional — enables nearby amenities)</span>
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
            <p className="text-xs font-semibold text-gray-600 uppercase tracking-wide">Include in report</p>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="w-4 h-4 rounded accent-blue-600"
                checked={form.include_yield}
                onChange={e => set('include_yield', e.target.checked)}
              />
              <div>
                <div className="text-sm font-medium text-gray-800">Rental yield analysis</div>
                <div className="text-xs text-gray-500">Ejari data — estimated ROI for investor clients</div>
              </div>
            </label>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                className="w-4 h-4 rounded accent-blue-600"
                checked={form.include_pois}
                onChange={e => set('include_pois', e.target.checked)}
              />
              <div>
                <div className="text-sm font-medium text-gray-800">Nearby amenities</div>
                <div className="text-xs text-gray-500">Metro, schools, malls — requires coordinates above</div>
              </div>
            </label>
          </div>

        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-100 flex items-center justify-between bg-gray-50">
          <p className="text-xs text-gray-400">
            Data sourced from DLD via BuyOrSell24
          </p>
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-800 transition"
            >
              Cancel
            </button>
            <button
              onClick={handleGenerate}
              disabled={loading}
              className="px-5 py-2 bg-[#1a3a5c] text-white text-sm font-semibold rounded-lg hover:bg-[#22487a] transition disabled:opacity-50 flex items-center gap-2"
            >
              {loading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  Generating…
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                  </svg>
                  Download PDF
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
