'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Document, DocumentSignature } from '@/types'
import clsx from 'clsx'

const typeLabels: Record<string, { en: string; ar: string }> = {
  lease: { en: 'Lease', ar: 'عقد إيجار' },
  offer: { en: 'Offer', ar: 'عرض' },
  inspection_report: { en: 'Inspection Report', ar: 'تقرير فحص' },
  maintenance_waiver: { en: 'Maintenance Waiver', ar: 'تنازل الصيانة' },
  custom: { en: 'Custom', ar: 'مخصص' },
}

const sigStatusColors: Record<string, string> = {
  not_required: 'bg-gray-100 text-gray-600',
  pending: 'bg-yellow-100 text-yellow-700',
  signed: 'bg-green-100 text-green-700',
}

const docTypeColors: Record<string, string> = {
  lease: 'bg-blue-100 text-blue-700',
  offer: 'bg-purple-100 text-purple-700',
  inspection_report: 'bg-indigo-100 text-indigo-700',
  maintenance_waiver: 'bg-orange-100 text-orange-700',
  custom: 'bg-gray-100 text-gray-600',
}

export default function DocumentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [doc, setDoc] = useState<Document | null>(null)
  const [sigs, setSigs] = useState<DocumentSignature[]>([])
  const [loading, setLoading] = useState(true)
  const [sigOpen, setSigOpen] = useState(false)
  const [sigName, setSigName] = useState('')
  const [sigEmail, setSigEmail] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const res: any = await api.documents.get(id)
      setDoc(res.document ?? null)
      setSigs(res.signatures ?? [])
    } catch { setDoc(null) } finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleRequestSignature = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setActionLoading(true)
    try {
      await api.documents.requestSignature(id, sigName, sigEmail)
      setSuccess(t('تم إرسال طلب التوقيع', 'Signature request sent'))
      setSigOpen(false); setSigName(''); setSigEmail(''); load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Error'))
    } finally { setActionLoading(false) }
  }

  const handleMarkSigned = async (sigId: string) => {
    setError(''); setActionLoading(true)
    try { await api.documents.markSigned(sigId); load() }
    catch (err: unknown) { setError(err instanceof Error ? err.message : t('حدث خطأ', 'Error')) }
    finally { setActionLoading(false) }
  }

  const handleDelete = async () => {
    if (!confirm(t('حذف المستند؟', 'Delete this document?'))) return
    try { await api.documents.delete(id); router.push('/documents') }
    catch { setError(t('فشل الحذف', 'Delete failed')) }
  }

  const fmtDate = (d: string) => new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' })
  const fmtDateTime = (d: string) => new Date(d).toLocaleString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  const fmtBytes = (b: number) => b > 1024 * 1024 ? `${(b / 1024 / 1024).toFixed(1)} MB` : b > 1024 ? `${(b / 1024).toFixed(1)} KB` : `${b} B`

  if (loading) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!doc) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('غير موجود', 'Document not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={doc.document_title} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <div className="flex items-center justify-between">
          <Link href="/documents" className="text-xs text-surface-500 hover:text-surface-700 font-medium flex items-center gap-1">← {t('العودة إلى المستندات', 'Back to Documents')}</Link>
          <div className="flex gap-2">
            {doc.file_url && (
              <a href={doc.file_url} target="_blank" rel="noopener noreferrer"
                className="px-4 py-2 border border-surface-200 text-surface-700 text-sm font-medium rounded-lg hover:bg-surface-50 inline-flex items-center gap-1.5">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
                {t('تحميل', 'Download')}
              </a>
            )}
            {isAgent && (
              <button onClick={() => setSigOpen(true)}
                className="px-4 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700">
                {t('طلب توقيع', 'Request Signature')}
              </button>
            )}
            {isAgent && (
              <button onClick={handleDelete}
                className="px-4 py-2 bg-red-500 text-white text-sm font-medium rounded-lg hover:bg-red-600">
                {t('حذف', 'Delete')}
              </button>
            )}
          </div>
        </div>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        {/* Document Info */}
        <div className="bg-white rounded-xl border border-surface-200/70 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div className="col-span-2">
            <p className="text-xs text-surface-400 mb-1">{t('العنوان', 'Title')}</p>
            <p className="font-semibold text-surface-900">{doc.document_title}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('النوع', 'Type')}</p>
            <span className={clsx('px-2 py-0.5 rounded text-[11px] font-medium', docTypeColors[doc.document_type] || 'bg-gray-100 text-gray-600')}>
              {lang === 'ar' ? (typeLabels[doc.document_type]?.ar || doc.document_type) : (typeLabels[doc.document_type]?.en || doc.document_type)}
            </span>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('حالة التوقيع', 'Signature Status')}</p>
            <span className={clsx('px-2 py-0.5 rounded text-[11px] font-medium', sigStatusColors[doc.signature_status])}>
              {doc.signature_status === 'not_required' ? (lang === 'ar' ? 'غير مطلوب' : 'Not Required')
                : doc.signature_status === 'pending' ? (lang === 'ar' ? 'معلق' : 'Pending')
                : (lang === 'ar' ? 'موقع' : 'Signed')}
            </span>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('التصنيف', 'Classification')}</p>
            <p className="font-medium text-surface-700 capitalize">{doc.data_classification}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('الحجم', 'File Size')}</p>
            <p className="font-medium text-surface-700">{fmtBytes(doc.file_size_bytes)}</p>
          </div>
          {doc.related_entity_type && (
            <div>
              <p className="text-xs text-surface-400 mb-1">{t('مرتبط بـ', 'Related To')}</p>
              <p className="font-medium text-surface-700 text-sm">{doc.related_entity_type}: {doc.related_entity_id?.substring(0, 8)}...</p>
            </div>
          )}
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('تاريخ الإنشاء', 'Created')}</p>
            <p className="font-medium text-surface-700 text-sm">{fmtDateTime(doc.created_at)}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('آخر تحديث', 'Updated')}</p>
            <p className="font-medium text-surface-700 text-sm">{fmtDateTime(doc.updated_at)}</p>
          </div>
        </div>

        {/* Signatures */}
        <div className="bg-white rounded-xl border border-surface-200/70 overflow-hidden">
          <div className="px-5 py-3.5 border-b border-surface-200/70 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-surface-900">{t('التوقيعات', 'Signatures')}</h3>
            <span className="text-xs text-surface-500">{sigs.length}</span>
          </div>
          {sigs.length === 0 ? (
            <p className="px-5 py-8 text-sm text-surface-400 text-center">
              {t('لا توجد توقيعات', 'No signatures')}
              {doc.signature_status !== 'not_required' && (
                <button onClick={() => setSigOpen(true)} className="block mx-auto mt-2 text-primary-600 hover:underline text-xs">
                  {t('طلب توقيع', 'Request signature')}
                </button>
              )}
            </p>
          ) : (
            <div className="divide-y divide-surface-100">
              {sigs.map(s => (
                <div key={s.id} className="flex items-center gap-3 px-5 py-3">
                  <div className="flex-1">
                    <p className="text-sm font-medium text-surface-900">{s.signer_name}</p>
                    <p className="text-xs text-surface-500">{s.signer_email}</p>
                    {s.signed_at && <p className="text-[10px] text-surface-400 mt-0.5">{t('تم في', 'Signed')} {fmtDateTime(s.signed_at)}</p>}
                  </div>
                  <span className={clsx('px-2 py-0.5 rounded text-[10px] font-medium', sigStatusColors[s.signature_status])}>
                    {s.signature_status === 'signed' ? (lang === 'ar' ? 'موقع' : 'Signed')
                      : s.signature_status === 'pending' ? (lang === 'ar' ? 'معلق' : 'Pending')
                      : (lang === 'ar' ? 'غير مطلوب' : 'Not Required')}
                  </span>
                  {isAgent && s.signature_status === 'pending' && (
                    <button onClick={() => handleMarkSigned(s.id)} disabled={actionLoading}
                      className="text-xs px-3 py-1 bg-green-100 text-green-700 rounded-md hover:bg-green-200 font-medium">
                      {t('تأكيد', 'Mark Signed')}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Request Signature Modal */}
      {sigOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => { setSigOpen(false); setError('') }}>
          <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-surface-900 mb-4">{t('طلب توقيع', 'Request Signature')}</h3>
            <form onSubmit={handleRequestSignature} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('الاسم', 'Name')} *</label>
                <input required className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={sigName} onChange={e => setSigName(e.target.value)} placeholder={t('اسم الموقّع', 'Signer name')} />
              </div>
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('البريد', 'Email')} *</label>
                <input required type="email" className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={sigEmail} onChange={e => setSigEmail(e.target.value)} placeholder="signer@example.com" />
              </div>
              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              <div className="flex gap-2 pt-2">
                <button type="button" onClick={() => { setSigOpen(false); setError('') }}
                  className="flex-1 py-2 border border-surface-200 rounded-lg text-sm text-surface-600 hover:bg-surface-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={actionLoading}
                  className="flex-1 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-60">
                  {actionLoading ? t('جاري الإرسال...', 'Sending...') : t('إرسال', 'Send')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
