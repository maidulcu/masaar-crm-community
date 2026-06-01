'use client'
import { useState, useRef } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

type Entity = 'contacts' | 'leads' | 'listings'
type ImportResult = { imported: number; skipped: number; errors: { row: number; error: string }[] }

const BASE = process.env.NEXT_PUBLIC_API_URL || ''

function getToken() {
  if (typeof window === 'undefined') return ''
  return localStorage.getItem('access_token') || ''
}

function downloadUrl(url: string) {
  const token = getToken()
  // Append token as query param for direct download links
  const sep = url.includes('?') ? '&' : '?'
  window.open(`${url}${sep}token=${token}`, '_blank')
}

export default function ImportExportPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [importEntity, setImportEntity] = useState<Entity>('contacts')
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [importError, setImportError] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const handleFile = async (file: File) => {
    if (!file.name.endsWith('.csv')) {
      setImportError('Please upload a .csv file')
      return
    }
    setImporting(true); setImportResult(null); setImportError('')
    try {
      let result: any
      if (importEntity === 'contacts') {
        result = await api.importExport.importContacts(file)
      } else {
        result = await api.importExport.importLeads(file)
      }
      setImportResult(result)
    } catch (err: any) {
      setImportError(err.message || 'Import failed')
    } finally { setImporting(false) }
  }

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault(); setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('الاستيراد والتصدير', 'Import & Export')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-3xl space-y-6">

          {/* ── Export section ──────────────────────────────────────── */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-1">{t('تصدير البيانات', 'Export Data')}</h2>
            <p className="text-xs text-gray-500 mb-4">Download your CRM data as a CSV file.</p>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              {[
                { label: 'Contacts', desc: 'All contacts with phone, email, language', action: () => downloadUrl(api.importExport.exportContacts()) },
                { label: 'Leads', desc: 'All leads with stage, value, contact info', action: () => downloadUrl(api.importExport.exportLeads()) },
                { label: 'Listings', desc: 'All property listings with specs & prices', action: () => downloadUrl(api.importExport.exportListings()) },
              ].map(item => (
                <div key={item.label} className="border border-gray-200 rounded-xl p-4">
                  <div className="flex items-start gap-3 mb-3">
                    <div className="w-9 h-9 rounded-lg bg-brand-50 flex items-center justify-center shrink-0">
                      <svg className="w-5 h-5 text-brand-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-gray-800">{item.label}</p>
                      <p className="text-xs text-gray-500">{item.desc}</p>
                    </div>
                  </div>
                  <button onClick={item.action}
                    className="w-full py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 transition-colors">
                    Download CSV
                  </button>
                </div>
              ))}
            </div>
          </div>

          {/* ── Import section ──────────────────────────────────────── */}
          {isAgent && (
            <div className="bg-white rounded-xl border border-gray-200 p-6">
              <h2 className="text-sm font-semibold text-gray-700 mb-1">{t('استيراد البيانات', 'Import Data')}</h2>
              <p className="text-xs text-gray-500 mb-4">
                Upload a CSV to bulk-create records. Max 500 rows. Download a template to see the expected columns.
              </p>

              {/* Entity selector */}
              <div className="flex gap-2 mb-4">
                {(['contacts', 'leads'] as const).map(e => (
                  <button key={e} onClick={() => { setImportEntity(e); setImportResult(null); setImportError('') }}
                    className={`px-4 py-2 rounded-lg text-xs font-semibold transition-colors capitalize ${
                      importEntity === e ? 'bg-brand-600 text-white' : 'border border-gray-200 text-gray-600 hover:bg-gray-50'
                    }`}>
                    {e}
                  </button>
                ))}
                <a href={`${BASE}/api/v1/import/template/${importEntity}?authorization=Bearer ${getToken()}`}
                  onClick={e => { e.preventDefault(); downloadUrl(api.importExport.template(importEntity)) }}
                  className="ml-auto text-xs text-brand-600 hover:underline flex items-center gap-1 font-medium">
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                  Download template
                </a>
              </div>

              {/* Drop zone */}
              <div
                onDragOver={e => { e.preventDefault(); setDragOver(true) }}
                onDragLeave={() => setDragOver(false)}
                onDrop={onDrop}
                onClick={() => fileRef.current?.click()}
                className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors ${
                  dragOver ? 'border-brand-400 bg-brand-50' : 'border-gray-200 hover:border-brand-300 hover:bg-gray-50'
                }`}>
                <input ref={fileRef} type="file" accept=".csv" className="hidden"
                  onChange={e => { const f = e.target.files?.[0]; if (f) handleFile(f); e.target.value = '' }} />
                {importing ? (
                  <div className="flex flex-col items-center gap-2">
                    <div className="w-8 h-8 border-2 border-brand-600 border-t-transparent rounded-full animate-spin" />
                    <p className="text-sm text-gray-500">Processing...</p>
                  </div>
                ) : (
                  <>
                    <svg className="w-10 h-10 text-gray-300 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                    </svg>
                    <p className="text-sm font-medium text-gray-700 mb-1">Drop CSV here or click to browse</p>
                    <p className="text-xs text-gray-400">Max 500 rows · .csv files only</p>
                  </>
                )}
              </div>

              {/* Error */}
              {importError && (
                <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                  {importError}
                </div>
              )}

              {/* Result */}
              {importResult && (
                <div className="mt-4 space-y-3">
                  <div className="grid grid-cols-3 gap-3">
                    {[
                      { label: 'Imported', value: importResult.imported, color: 'text-green-600 bg-green-50 border-green-200' },
                      { label: 'Skipped', value: importResult.skipped, color: 'text-amber-600 bg-amber-50 border-amber-200' },
                      { label: 'Errors', value: importResult.errors?.length ?? 0, color: 'text-red-600 bg-red-50 border-red-200' },
                    ].map(stat => (
                      <div key={stat.label} className={`rounded-xl border p-3 text-center ${stat.color}`}>
                        <p className="text-2xl font-bold">{stat.value}</p>
                        <p className="text-xs font-medium mt-0.5">{stat.label}</p>
                      </div>
                    ))}
                  </div>

                  {importResult.errors?.length > 0 && (
                    <div className="border border-red-200 rounded-xl overflow-hidden">
                      <div className="bg-red-50 px-4 py-2 border-b border-red-200">
                        <p className="text-xs font-semibold text-red-700">Row Errors</p>
                      </div>
                      <div className="max-h-48 overflow-y-auto divide-y divide-red-100">
                        {importResult.errors.slice(0, 50).map((err, i) => (
                          <div key={i} className="px-4 py-2 flex items-center gap-3">
                            <span className="text-[10px] bg-red-100 text-red-700 font-bold px-1.5 py-0.5 rounded shrink-0">Row {err.row}</span>
                            <span className="text-xs text-red-600">{err.error}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {importResult.imported > 0 && (
                    <p className="text-sm text-green-700 font-medium text-center">
                      ✅ {importResult.imported} {importEntity} imported successfully
                    </p>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Column reference */}
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('أعمدة CSV المطلوبة', 'Required CSV Columns')}</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
              <div>
                <p className="font-semibold text-gray-600 mb-1">Contacts</p>
                <table className="w-full">
                  <tbody>
                    {[['phone_wa', 'required', 'WhatsApp number (+971...)'], ['full_name', 'required', 'Contact name'], ['email', 'optional', 'Email address'], ['language', 'optional', 'en or ar']].map(([col, req, desc]) => (
                      <tr key={col} className="border-b border-gray-100">
                        <td className="py-1 font-mono text-gray-700">{col}</td>
                        <td className={`py-1 px-2 ${req === 'required' ? 'text-red-500' : 'text-gray-400'}`}>{req}</td>
                        <td className="py-1 text-gray-500">{desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div>
                <p className="font-semibold text-gray-600 mb-1">Leads</p>
                <table className="w-full">
                  <tbody>
                    {[['contact_phone', 'required', 'Buyer phone number'], ['stage', 'optional', 'new/contacted/qualified'], ['source', 'optional', 'web/referral/event'], ['deal_value', 'optional', 'Numeric amount'], ['currency', 'optional', 'AED (default)'], ['notes', 'optional', 'Free text']].map(([col, req, desc]) => (
                      <tr key={col} className="border-b border-gray-100">
                        <td className="py-1 font-mono text-gray-700">{col}</td>
                        <td className={`py-1 px-2 ${req === 'required' ? 'text-red-500' : 'text-gray-400'}`}>{req}</td>
                        <td className="py-1 text-gray-500">{desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

        </div>
      </main>
    </div>
  )
}
