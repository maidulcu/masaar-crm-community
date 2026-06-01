'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { MaintenanceTask, MaintenancePhoto } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  pending: 'bg-gray-100 text-gray-600',
  scheduled: 'bg-yellow-100 text-yellow-700',
  in_progress: 'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-700',
}

const priorityColor: Record<string, string> = {
  low: 'bg-green-100 text-green-700',
  medium: 'bg-yellow-100 text-yellow-700',
  high: 'bg-orange-100 text-orange-700',
  urgent: 'bg-red-100 text-red-700',
}

export default function MaintenanceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [task, setTask] = useState<MaintenanceTask | null>(null)
  const [photos, setPhotos] = useState<MaintenancePhoto[]>([])
  const [loading, setLoading] = useState(true)
  const [completeOpen, setCompleteOpen] = useState(false)
  const [actualCost, setActualCost] = useState('')
  const [photoUrl, setPhotoUrl] = useState('')
  const [photoStage, setPhotoStage] = useState('after')
  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const [tRes, pRes] = await Promise.all([
        api.maintenance.get(id) as Promise<{ data: MaintenanceTask }>,
        api.maintenance.getPhotos(id) as Promise<{ data: MaintenancePhoto[] }>,
      ])
      setTask(tRes.data)
      setPhotos(pRes.data ?? [])
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleComplete = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      const cost = parseFloat(actualCost)
      await api.maintenance.complete(id, isNaN(cost) ? undefined : { actual_cost: cost })
      setSuccess(t('تم الإكمال', 'Completed'))
      setCompleteOpen(false); load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setActionLoading(false) }
  }

  const handleAddPhoto = async () => {
    if (!photoUrl.trim()) return
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      await api.maintenance.addPhoto(id, photoUrl, photoStage)
      setPhotoUrl(''); load()
      setSuccess(t('تمت إضافة الصورة', 'Photo added'))
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setActionLoading(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'
  const fmtAmount = (v: number | null | undefined) => v != null ? `AED ${v.toLocaleString()}` : '—'

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'

  if (loading) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!task) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Task not found')}</div>

  const canComplete = isAgent && task.status !== 'completed' && task.status !== 'cancelled'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تفاصيل الصيانة', 'Maintenance Details')} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/maintenance" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">← {t('العودة', 'Back to Maintenance')}</Link>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[task.status])}>{task.status.replace('_', ' ')}</span>
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', priorityColor[task.priority])}>{task.priority}</span>
          <span className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded-full">{task.maintenance_type}</span>
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div className="col-span-2"><p className="text-xs text-gray-400 mb-1">{t('الوصف', 'Description')}</p><p className="font-semibold text-gray-900">{task.description}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الصيانة', 'Scheduled Date')}</p><p className="font-medium text-gray-700">{fmtDate(task.scheduled_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الاستحقاق', 'Due Date')}</p><p className="font-medium text-gray-700">{fmtDate(task.due_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('تاريخ الإكمال', 'Completed Date')}</p><p className="font-medium text-gray-700">{fmtDate(task.completion_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('التكلفة التقديرية', 'Est. Cost')}</p><p className="font-medium text-gray-700">{fmtAmount(task.estimated_cost)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('التكلفة الفعلية', 'Actual Cost')}</p><p className="font-medium text-gray-700">{fmtAmount(task.actual_cost)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('المقاول', 'Contractor')}</p><p className="font-medium text-gray-700">{task.contractor_name || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('رقم المقاول', 'Contractor Contact')}</p><p className="font-medium text-gray-700">{task.contractor_contact || '—'}</p></div>
        </div>

        {task.notes && <div className="bg-white rounded-xl border border-gray-100 p-5"><h3 className="text-sm font-semibold text-gray-700 mb-2">{t('ملاحظات', 'Notes')}</h3><p className="text-sm text-gray-700 whitespace-pre-wrap">{task.notes}</p></div>}

        {/* Photos */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-gray-700">{t('الصور', 'Photos')}</h3>
            {isAgent && (
              <div className="flex gap-2 items-center">
                <select className="text-xs border border-gray-200 rounded px-2 py-1" value={photoStage} onChange={e => setPhotoStage(e.target.value)}>
                  <option value="before">{t('قبل', 'Before')}</option>
                  <option value="during">{t('أثناء', 'During')}</option>
                  <option value="after">{t('بعد', 'After')}</option>
                </select>
                <input className="text-xs border border-gray-200 rounded px-2 py-1 w-40" value={photoUrl} onChange={e => setPhotoUrl(e.target.value)} placeholder="https://..." />
                <button onClick={handleAddPhoto} disabled={actionLoading || !photoUrl.trim()}
                  className="text-xs px-3 py-1.5 bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50">{t('إضافة', 'Add')}</button>
              </div>
            )}
          </div>
          {photos.length === 0 ? (
            <p className="text-xs text-gray-400">{t('لا توجد صور', 'No photos yet')}</p>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {photos.map(ph => (
                <div key={ph.id} className="border border-gray-200 rounded-lg p-3">
                  <a href={ph.photo_url} target="_blank" rel="noopener noreferrer" className="text-xs text-brand-600 hover:underline block mb-1 break-all">{ph.photo_url}</a>
                  <p className="text-[10px] text-gray-400">{ph.photo_stage} · {fmtDate(ph.uploaded_at)}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        {canComplete && (
          <button onClick={() => setCompleteOpen(true)} className="px-5 py-2.5 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700">
            {t('إكمال مهمة الصيانة', 'Complete Task')}
          </button>
        )}
      </div>

      {/* Complete Modal */}
      {completeOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setCompleteOpen(false)}>
          <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('إكمال مهمة الصيانة', 'Complete Task')}</h3>
            <form onSubmit={handleComplete} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('التكلفة الفعلية (AED)', 'Actual Cost (AED)')}</label>
                <input type="number" min={0} step={0.01} className={inputCls} value={actualCost} onChange={e => setActualCost(e.target.value)} placeholder={t('اختياري', 'Optional')} />
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setCompleteOpen(false)} className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={actionLoading} className="flex-1 py-2.5 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700">{t('تأكيد الإكمال', 'Confirm Complete')}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
