'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { BankStatement, BankIntegration, FileFormat, ProcessingStatus } from '@/types'

const formatLabel: Record<FileFormat, string> = {
  csv: 'CSV',
  pdf: 'PDF',
  xlsx: 'XLSX',
}

const statusColor: Record<ProcessingStatus, string> = {
  pending:    'bg-yellow-100 text-yellow-700',
  processing: 'bg-blue-100 text-blue-700',
  completed:  'bg-green-100 text-green-700',
  failed:     'bg-red-100 text-red-700',
}

const statusLabel = {
  en: {
    pending:    'Pending',
    processing: 'Processing',
    completed:  'Completed',
    failed:     'Failed',
  },
  ar: {
    pending:    'قيد الانتظار',
    processing: 'قيد المعالجة',
    completed:  'مكتمل',
    failed:     'فشل',
  },
}

function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

export default function BankStatementsPage() {
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [statements, setStatements] = useState<BankStatement[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [showUpload, setShowUpload] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [selectedBankId, setSelectedBankId] = useState('')
  const [bankIntegrations, setBankIntegrations] = useState<BankIntegration[]>([])

  const [detailItem, setDetailItem] = useState<BankStatement | null>(null)

  const [deleting, setDeleting] = useState<string | null>(null)

  const sl = statusLabel[lang as 'en' | 'ar']

  useEffect(() => {
    loadStatements()
  }, [page])

  const loadStatements = async () => {
    setLoading(true)
    try {
      const res: any = await api.bankStatements.list({ page, limit })
      setStatements(res?.data ?? res ?? [])
      setTotal(res?.total ?? 0)
    } catch {
      setStatements([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const openUpload = async () => {
    setSelectedFile(null)
    setSelectedBankId('')
    setUploading(false)
    try {
      const res: any = await api.bankIntegrations.list({ page: 1, limit: 100 })
      const list: BankIntegration[] = res?.data ?? res ?? []
      setBankIntegrations(list)
    } catch {
      setBankIntegrations([])
    }
    setShowUpload(true)
  }

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedFile || !selectedBankId) return
    setUploading(true)
    try {
      await api.bankStatements.upload(selectedFile, selectedBankId)
      setShowUpload(false)
      setPage(1)
      loadStatements()
    } catch (err: any) {
      alert(err?.message || t('فشل الرفع', 'Upload failed'))
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('هل أنت متأكد من حذف كشف الحساب هذا؟', 'Delete this statement?'))) return
    setDeleting(id)
    try {
      await api.bankStatements.delete(id)
      loadStatements()
    } catch {
      alert(t('فشل الحذف', 'Failed to delete'))
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('كشوفات الحساب', 'Bank Statements')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : statements.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد كشوفات بعد', 'No statements yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('ارفع كشف حساب لبدء استيراد المعاملات', 'Upload a bank statement to start importing transactions')}</p>
            </div>
            <button onClick={openUpload}
              className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
              {t('+ رفع كشف', '+ Upload Statement')}
            </button>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{total} {t('كشف', 'statements')}</p>
              <button onClick={openUpload}
                className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                {t('+ رفع كشف', '+ Upload Statement')}
              </button>
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('اسم الملف', 'File Name')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('النوع', 'Format')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحجم', 'Size')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('تاريخ الرفع', 'Upload Date')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('المعاملات', 'Transactions')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {statements.map(st => (
                    <tr key={st.id} className="hover:bg-gray-50 transition-colors cursor-pointer" onClick={() => setDetailItem(st)}>
                      <td className="px-5 py-3.5 font-medium text-gray-900 max-w-[200px] truncate">{st.file_name}</td>
                      <td className="px-5 py-3.5 text-gray-600">
                        <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-700">
                          {formatLabel[st.file_format] || st.file_format}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-gray-600 text-xs">{formatBytes(st.file_size_bytes)}</td>
                      <td className="px-5 py-3.5 text-gray-400 text-xs">
                        {new Date(st.upload_date).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE')}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', statusColor[st.processing_status])}>
                          {sl[st.processing_status]}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-gray-600 text-xs">
                        {st.processing_status === 'completed' ? st.transactions_imported : '—'}
                      </td>
                      <td className="px-5 py-3.5 text-end" onClick={e => e.stopPropagation()}>
                        {isAdmin && (
                          <button
                            onClick={() => handleDelete(st.id)}
                            disabled={deleting === st.id}
                            className="text-xs text-red-500 hover:text-red-700 font-medium disabled:opacity-30"
                          >
                            {t('حذف', 'Delete')}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-center gap-2 mt-6">
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('السابق', 'Prev')}
                </button>
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(p => (
                  <button key={p} onClick={() => setPage(p)}
                    className={clsx('px-3 py-1.5 text-sm rounded-lg', p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50')}>
                    {p}
                  </button>
                ))}
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Upload Modal */}
      {showUpload && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowUpload(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('رفع كشف حساب', 'Upload Statement')}</h2>
            <form onSubmit={handleUpload} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('حساب بنكي', 'Bank Account')}</label>
                <select value={selectedBankId} onChange={e => setSelectedBankId(e.target.value)} required
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="">{t('اختر حساباً', 'Select an account')}</option>
                  {bankIntegrations.map(bi => (
                    <option key={bi.id} value={bi.id}>{bi.bank_name} — {bi.account_name} ({bi.account_number})</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('الملف', 'File')} (CSV, PDF, XLSX)</label>
                <input type="file" accept=".csv,.pdf,.xlsx" onChange={e => setSelectedFile(e.target.files?.[0] || null)}
                  className="w-full text-sm text-gray-600 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-brand-50 file:text-brand-700 hover:file:bg-brand-100" required />
              </div>
              {selectedFile && (
                <p className="text-xs text-gray-400">
                  {selectedFile.name} — {formatBytes(selectedFile.size)}
                </p>
              )}
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowUpload(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={uploading || !selectedFile || !selectedBankId}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {uploading ? '...' : t('رفع', 'Upload')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Detail Modal */}
      {detailItem && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setDetailItem(null)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('تفاصيل كشف الحساب', 'Statement Details')}</h2>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('اسم الملف', 'File Name')}</span>
                <span className="font-medium text-gray-900 text-end max-w-[60%] break-all">{detailItem.file_name}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('النوع', 'Format')}</span>
                <span className="font-medium">{formatLabel[detailItem.file_format]}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('الحجم', 'Size')}</span>
                <span className="font-medium">{formatBytes(detailItem.file_size_bytes)}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('تاريخ الرفع', 'Upload Date')}</span>
                <span className="font-medium">{new Date(detailItem.upload_date).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE')}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('الحالة', 'Status')}</span>
                <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', statusColor[detailItem.processing_status])}>
                  {sl[detailItem.processing_status]}
                </span>
              </div>
              {detailItem.processing_status === 'completed' && (
                <div className="flex justify-between py-2 border-b border-gray-50">
                  <span className="text-gray-500">{t('المعاملات المستوردة', 'Transactions Imported')}</span>
                  <span className="font-medium">{detailItem.transactions_imported}</span>
                </div>
              )}
              {detailItem.import_error && (
                <div className="py-2">
                  <span className="text-gray-500 block mb-1">{t('خطأ', 'Error')}</span>
                  <p className="text-red-600 text-xs bg-red-50 rounded-lg p-2">{detailItem.import_error}</p>
                </div>
              )}
              <div className="flex justify-between py-2 border-b border-gray-50">
                <span className="text-gray-500">{t('التصنيف', 'Classification')}</span>
                <span className="font-medium capitalize">{detailItem.data_classification}</span>
              </div>
            </div>
            <div className="flex justify-end pt-4">
              <button onClick={() => setDetailItem(null)}
                className="px-4 py-2 text-sm font-medium bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors">
                {t('إغلاق', 'Close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
