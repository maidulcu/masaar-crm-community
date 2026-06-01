'use client'
import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/layout/Header'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'
import { useRouter } from 'next/navigation'
import { api } from '@/lib/api'
import {
  DndContext, closestCenter, KeyboardSensor, PointerSensor, useSensor, useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove, SortableContext, sortableKeyboardCoordinates, useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { PipelineStage } from '@/types'

function SortableStage({ stage, onEdit, onDelete }: {
  stage: PipelineStage
  onEdit: (s: PipelineStage) => void
  onDelete: (id: string) => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: stage.id })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div ref={setNodeRef} style={style} className="flex items-center gap-3 p-3 bg-white border border-gray-200 rounded-lg">
      <button {...attributes} {...listeners} className="cursor-grab text-gray-400 hover:text-gray-600">
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 15l4-4 4 4" />
          <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 9l4 4 4-4" />
        </svg>
      </button>
      <div className="w-3 h-3 rounded-full shrink-0" style={{ backgroundColor: stage.color }} />
      <span className="flex-1 text-sm font-medium text-gray-800">{stage.name}</span>
      {stage.is_won && <span className="text-[10px] font-semibold text-green-600 bg-green-50 px-2 py-0.5 rounded">Won</span>}
      {stage.is_lost && <span className="text-[10px] font-semibold text-red-600 bg-red-50 px-2 py-0.5 rounded">Lost</span>}
      <div className="flex gap-1">
        <button onClick={() => onEdit(stage)} className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10" />
          </svg>
        </button>
        <button onClick={() => onDelete(stage.id)} className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  )
}

export default function PipelineSettingsPage() {
  const { user, company } = useAuthStore()
  const { t } = useLang()
  const router = useRouter()
  const isDemo = !!company?.is_demo

  const [stages, setStages] = useState<PipelineStage[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  // Edit modal
  const [editStage, setEditStage] = useState<Partial<PipelineStage> | null>(null)
  const [showModal, setShowModal] = useState(false)

  const loadStages = useCallback(async () => {
    try {
      const data = await api.pipelineStages.list('lead') as PipelineStage[]
      setStages(data)
    } catch { }
    setLoading(false)
  }, [])

  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    loadStages()
  }, [user, router, loadStages])

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    if (!over || active.id === over.id) return

    const oldIndex = stages.findIndex(s => s.id === active.id)
    const newIndex = stages.findIndex(s => s.id === over.id)
    if (oldIndex === -1 || newIndex === -1) return

    const reordered = arrayMove(stages, oldIndex, newIndex)
    setStages(reordered)
    await api.pipelineStages.reorder(reordered.map(s => s.id)).catch(() => loadStages())
  }

  const handleOpenNew = () => {
    setEditStage({ name: '', color: '#6366f1', entity_type: 'lead', is_won: false, is_lost: false })
    setShowModal(true)
  }

  const handleOpenEdit = (s: PipelineStage) => {
    setEditStage({ ...s })
    setShowModal(true)
  }

  const handleSaveStage = async () => {
    if (!editStage || !editStage.name?.trim()) return
    if (isDemo) return
    setSaving(true)
    try {
      if (editStage.id) {
        await api.pipelineStages.update(editStage.id, editStage)
      } else {
        await api.pipelineStages.create(editStage)
      }
      setShowModal(false)
      setEditStage(null)
      await loadStages()
    } catch (err: any) {
      setError(err.message || 'Failed to save')
    } finally { setSaving(false) }
  }

  const handleDelete = async (id: string) => {
    if (isDemo || !confirm(t('حذف هذه المرحلة؟ سيتم إبقاء العملاء في هذه المرحلة بدون تغيير.', 'Delete this stage? Leads in this stage will remain unchanged.'))) return
    try {
      await api.pipelineStages.delete(id)
      await loadStages()
    } catch { }
  }

  const handleReset = async () => {
    if (isDemo || !confirm(t('إعادة تعيين المراحل الافتراضية؟ سيتم فقدان أي مراحل مخصصة.', 'Reset to default stages? Any custom stages will be lost.'))) return
    try {
      await api.pipelineStages.resetDefault('lead')
      await loadStages()
    } catch { }
  }

  if (user?.role !== 'admin') return null

  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('مراحل البيع', 'Pipeline Stages')} />
      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-2xl space-y-6">

          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span><strong>Demo account</strong> — settings are read-only.</span>
            </div>
          )}

          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-sm font-semibold text-gray-700">{t('المراحل', 'Stages')}</h2>
                <p className="text-xs text-gray-500 mt-0.5">{t('اسحب لإعادة الترتيب', 'Drag to reorder')}</p>
              </div>
              <div className="flex gap-2">
                <button onClick={handleReset} className="text-xs text-gray-500 hover:text-gray-700 px-3 py-1.5 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                  {t('إعادة تعيين', 'Reset Defaults')}
                </button>
                <button onClick={handleOpenNew} disabled={isDemo}
                  className="px-3 py-1.5 bg-brand-600 text-white text-xs font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {t('إضافة مرحلة', 'Add Stage')}
                </button>
              </div>
            </div>

            {loading ? (
              <div className="text-center py-8 text-sm text-gray-400">{t('جارٍ التحميل...', 'Loading...')}</div>
            ) : (
              <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
                <SortableContext items={stages.map(s => s.id)} strategy={verticalListSortingStrategy}>
                  <div className="space-y-2">
                    {stages.map(s => (
                      <SortableStage key={s.id} stage={s} onEdit={handleOpenEdit} onDelete={handleDelete} />
                    ))}
                  </div>
                </SortableContext>
              </DndContext>
            )}
          </div>

          {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
          {saved && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{t('تم الحفظ', 'Saved')}</p>}

          {/* Modal */}
          {showModal && editStage && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setShowModal(false)}>
              <div className="bg-white rounded-xl shadow-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
                <h3 className="text-sm font-semibold text-gray-800 mb-4">
                  {editStage.id ? t('تعديل المرحلة', 'Edit Stage') : t('إضافة مرحلة جديدة', 'New Stage')}
                </h3>
                <div className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">{t('الاسم', 'Name')}</label>
                    <input type="text" value={editStage.name || ''}
                      onChange={e => setEditStage(s => ({ ...s, name: e.target.value }))}
                      className={inputCls} />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">{t('اللون', 'Color')}</label>
                    <input type="color" value={editStage.color || '#6366f1'}
                      onChange={e => setEditStage(s => ({ ...s, color: e.target.value }))}
                      className="w-full h-10 rounded cursor-pointer" />
                  </div>
                  <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2 text-sm text-gray-700">
                      <input type="checkbox" checked={!!editStage.is_won}
                        onChange={e => { const checked = e.target.checked; setEditStage({ ...editStage, is_won: checked, is_lost: checked ? false : (editStage.is_lost ?? false) }) }}
                        className="accent-brand-600" />
                      {t('مرحلة فوز', 'Won Stage')}
                    </label>
                    <label className="flex items-center gap-2 text-sm text-gray-700">
                      <input type="checkbox" checked={!!editStage.is_lost}
                        onChange={e => { const checked = e.target.checked; setEditStage({ ...editStage, is_lost: checked, is_won: checked ? false : (editStage.is_won ?? false) }) }}
                        className="accent-brand-600" />
                      {t('مرحلة خسارة', 'Lost Stage')}
                    </label>
                  </div>
                </div>
                <div className="flex gap-2 mt-6">
                  <button onClick={() => setShowModal(false)}
                    className="flex-1 py-2 text-sm border border-gray-200 rounded-lg text-gray-600 hover:bg-gray-50 transition-colors">
                    {t('إلغاء', 'Cancel')}
                  </button>
                  <button onClick={handleSaveStage} disabled={saving || !editStage.name?.trim()}
                    className="flex-1 py-2 text-sm bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                    {saving ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ', 'Save')}
                  </button>
                </div>
              </div>
            </div>
          )}

        </div>
      </main>
    </div>
  )
}
