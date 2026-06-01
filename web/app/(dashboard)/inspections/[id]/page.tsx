'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Inspection } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  scheduled: 'bg-yellow-100 text-yellow-700',
  in_progress: 'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-700',
}

const severityColor: Record<string, string> = {
  green: 'bg-green-100 text-green-700',
  yellow: 'bg-yellow-100 text-yellow-700',
  red: 'bg-red-100 text-red-700',
}

export default function InspectionDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [inspection, setInspection] = useState<Inspection | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const [showEdit, setShowEdit] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editForm, setEditForm] = useState({
    status: '',
    findings: '',
    severity_level: 'green' as string,
    inspector_id: '',
    completed_date: '',
  })

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const r = await api.inspection.get(id) as { data: Inspection }
      const insp = r.data
      setInspection(insp)
      setEditForm({
        status: insp.status,
        findings: insp.findings || '',
        severity_level: insp.severity_level || 'green',
        inspector_id: insp.inspector_id || '',
        completed_date: insp.completed_date
          ? new Date(insp.completed_date).toISOString().split('T')[0]
          : '',
      })
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleComplete = async () => {
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      await api.inspection.complete(id)
      setSuccess(t('تم الإكمال', 'Completed'))
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setActionLoading(false) }
  }

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        status: editForm.status,
        findings: editForm.findings,
        severity_level: editForm.severity_level,
      }
      if (editForm.inspector_id) payload.inspector_id = editForm.inspector_id
      if (editForm.completed_date) payload.completed_date = new Date(editForm.completed_date).toISOString()
      await api.inspection.update(id, payload)
      setShowEdit(false)
      setSuccess(t('تم الحفظ', 'Saved'))
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch {
      alert(t('فشل الحفظ', 'Failed to save'))
    } finally { setSaving(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'

  if (loading) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!inspection) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Inspection not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تفاصيل الفحص', 'Inspection Details')} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/inspections" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">← {t('العودة', 'Back to Inspections')}</Link>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[inspection.status])}>{inspection.status.replace('_', ' ')}</span>
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', severityColor[inspection.severity_level])}>{inspection.severity_level}</span>
          <span className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded-full">{inspection.inspection_type.replace('_', ' ')}</span>
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الفحص', 'Scheduled Date')}</p><p className="font-semibold text-gray-900">{fmtDate(inspection.scheduled_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الإكمال', 'Completed Date')}</p><p className="font-medium text-gray-700">{fmtDate(inspection.completed_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('رقم العقار', 'Property ID')}</p><p className="font-medium text-gray-700 font-mono text-xs">{inspection.property_id}</p></div>
        </div>

        {inspection.findings && <div className="bg-white rounded-xl border border-gray-100 p-5"><h3 className="text-sm font-semibold text-gray-700 mb-2">{t('النتائج', 'Findings')}</h3><p className="text-sm text-gray-700 whitespace-pre-wrap">{inspection.findings}</p></div>}

        {inspection.photos_urls?.length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('الصور', 'Photos')}</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {inspection.photos_urls.map((url, i) => (
                <a key={i} href={url} target="_blank" rel="noopener noreferrer" className="aspect-video bg-gray-100 rounded-lg flex items-center justify-center text-xs text-gray-500 hover:bg-gray-200 border border-gray-200">
                  {t('صورة', 'Photo')} {i + 1}
                </a>
              ))}
            </div>
          </div>
        )}

        {inspection.checklist_results && Object.keys(inspection.checklist_results).length > 0 && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('نتائج القائمة', 'Checklist Results')}</h3>
            <div className="space-y-2">
              {Object.entries(inspection.checklist_results).map(([itemId, result]) => (
                <div key={itemId} className="flex items-center gap-3 text-sm p-2 bg-gray-50 rounded">
                  <span className={`w-2 h-2 rounded-full ${result.status === 'pass' ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-gray-700">{itemId}</span>
                  {result.notes && <span className="text-xs text-gray-400 ml-auto">{result.notes}</span>}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="flex gap-3">
          {isAgent && inspection.status !== 'completed' && inspection.status !== 'cancelled' && (
            <button onClick={handleComplete} disabled={actionLoading}
              className="px-5 py-2.5 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 disabled:opacity-50">
              {actionLoading ? '...' : t('إكمال الفحص', 'Complete Inspection')}
            </button>
          )}
          {isAgent && (
            <button onClick={() => setShowEdit(true)}
              className="px-5 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
              {t('تعديل', 'Edit')}
            </button>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {showEdit && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowEdit(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('تعديل الفحص', 'Edit Inspection')}</h2>
            <form onSubmit={handleEdit} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('الحالة', 'Status')}</label>
                <select value={editForm.status} onChange={e => setEditForm(f => ({ ...f, status: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="scheduled">{t('مجدول', 'Scheduled')}</option>
                  <option value="in_progress">{t('قيد التنفيذ', 'In Progress')}</option>
                  <option value="completed">{t('مكتمل', 'Completed')}</option>
                  <option value="cancelled">{t('ملغي', 'Cancelled')}</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('المستوى', 'Severity')}</label>
                <select value={editForm.severity_level} onChange={e => setEditForm(f => ({ ...f, severity_level: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="green">{t('جيد', 'Good')}</option>
                  <option value="yellow">{t('بسيط', 'Minor')}</option>
                  <option value="red">{t('حرج', 'Critical')}</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('النتائج', 'Findings')}</label>
                <textarea value={editForm.findings} onChange={e => setEditForm(f => ({ ...f, findings: e.target.value }))} rows={4}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف المفتش', 'Inspector ID')}</label>
                  <input value={editForm.inspector_id} onChange={e => setEditForm(f => ({ ...f, inspector_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('تاريخ الإكمال', 'Completed Date')}</label>
                  <input type="date" value={editForm.completed_date} onChange={e => setEditForm(f => ({ ...f, completed_date: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowEdit(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={saving}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {saving ? '...' : t('حفظ', 'Save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
