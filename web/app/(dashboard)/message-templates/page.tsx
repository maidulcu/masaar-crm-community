'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { MessageTemplate } from '@/types'

const CATEGORIES = ['marketing', 'utility', 'authentication', 'general']

const emptyForm = {
  name: '',
  body: '',
  category: 'general',
  variables: [] as string[],
  is_active: true,
}

export default function MessageTemplatesPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const canWrite = user?.role === 'admin' || user?.role === 'agent'

  const [templates, setTemplates] = useState<MessageTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [showModal, setShowModal] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState({ ...emptyForm })
  const [variableInput, setVariableInput] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [deleting, setDeleting] = useState<string | null>(null)
  const [preview, setPreview] = useState<MessageTemplate | null>(null)

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res: any = await api.messageTemplates.list({ page, limit })
      setTemplates(res?.data ?? res ?? [])
      setTotal(res?.total ?? 0)
    } catch {
      setTemplates([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const openCreate = () => {
    setForm({ ...emptyForm })
    setVariableInput('')
    setEditingId(null)
    setSaveError('')
    setShowModal(true)
  }

  const openEdit = (tpl: MessageTemplate) => {
    setForm({
      name: tpl.name,
      body: tpl.body,
      category: tpl.category || 'general',
      variables: tpl.variables ?? [],
      is_active: tpl.is_active,
    })
    setVariableInput('')
    setEditingId(tpl.id)
    setSaveError('')
    setShowModal(true)
  }

  const addVariable = () => {
    const v = variableInput.trim()
    if (!v || form.variables.includes(v)) return
    setForm(f => ({ ...f, variables: [...f.variables, v] }))
    setVariableInput('')
  }

  const removeVariable = (v: string) =>
    setForm(f => ({ ...f, variables: f.variables.filter(x => x !== v) }))

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.name.trim() || !form.body.trim()) return
    setSaving(true)
    setSaveError('')
    try {
      if (editingId) {
        await api.messageTemplates.update(editingId, form)
      } else {
        await api.messageTemplates.create(form)
      }
      setShowModal(false)
      setPage(1)
      load()
    } catch {
      setSaveError(t('حدث خطأ في الحفظ', 'Failed to save template'))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('هل أنت متأكد من حذف هذا القالب؟', 'Delete this template?'))) return
    setDeleting(id)
    try {
      await api.messageTemplates.delete(id)
      load()
    } catch {
      alert(t('فشل الحذف', 'Failed to delete'))
    } finally {
      setDeleting(null)
    }
  }

  const resolvedBody = (tpl: MessageTemplate) => {
    let body = tpl.body
    ;(tpl.variables ?? []).forEach((v, i) => {
      body = body.replace(new RegExp(`\\{\\{${i + 1}\\}\\}`, 'g'), `[${v}]`)
    })
    return body
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('قوالب الرسائل', 'Message Templates')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : templates.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد قوالب بعد', 'No templates yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('أنشئ قالباً لإرساله عبر واتساب', 'Create a template to send via WhatsApp')}</p>
            </div>
            {canWrite && (
              <button onClick={openCreate}
                className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
                {t('+ قالب جديد', '+ New Template')}
              </button>
            )}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{total} {t('قالب', 'templates')}</p>
              {canWrite && (
                <button onClick={openCreate}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                  {t('+ قالب جديد', '+ New Template')}
                </button>
              )}
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('الاسم', 'Name')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الفئة', 'Category')}</th>
                    <th className="text-start px-5 py-3 font-medium hidden md:table-cell">{t('المتغيرات', 'Variables')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {templates.map(tpl => (
                    <tr
                      key={tpl.id}
                      className="hover:bg-gray-50 transition-colors cursor-pointer"
                      onClick={() => setPreview(tpl)}
                    >
                      <td className="px-5 py-3.5 font-medium text-gray-900">{tpl.name}</td>
                      <td className="px-5 py-3.5">
                        <span className="text-xs px-2 py-0.5 rounded-full bg-surface-100 text-surface-600 capitalize">
                          {tpl.category || 'general'}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 hidden md:table-cell">
                        {tpl.variables?.length > 0 ? (
                          <div className="flex flex-wrap gap-1">
                            {tpl.variables.slice(0, 3).map(v => (
                              <span key={v} className="text-[11px] px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded font-mono">
                                {v}
                              </span>
                            ))}
                            {tpl.variables.length > 3 && (
                              <span className="text-[11px] text-gray-400">+{tpl.variables.length - 3}</span>
                            )}
                          </div>
                        ) : (
                          <span className="text-gray-300 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx(
                          'text-xs font-medium px-2 py-0.5 rounded-full',
                          tpl.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                        )}>
                          {tpl.is_active ? t('نشط', 'Active') : t('معطل', 'Inactive')}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-end" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-3">
                          {canWrite && (
                            <button
                              onClick={() => openEdit(tpl)}
                              className="text-xs text-brand-600 hover:text-brand-700 font-medium"
                            >
                              {t('تعديل', 'Edit')}
                            </button>
                          )}
                          {isAdmin && (
                            <button
                              onClick={() => handleDelete(tpl.id)}
                              disabled={deleting === tpl.id}
                              className="text-xs text-red-500 hover:text-red-700 font-medium disabled:opacity-30"
                            >
                              {t('حذف', 'Delete')}
                            </button>
                          )}
                        </div>
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
                {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                  const p = page <= 3 ? i + 1 : page - 2 + i
                  return p <= totalPages ? (
                    <button key={p} onClick={() => setPage(p)}
                      className={clsx('px-3 py-1.5 text-sm rounded-lg', p === page ? 'bg-brand-600 text-white' : 'border border-gray-200 hover:bg-gray-50')}>
                      {p}
                    </button>
                  ) : null
                })}
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}
                  className="px-3 py-1.5 text-sm border border-gray-200 rounded-lg disabled:opacity-30 hover:bg-gray-50">
                  {t('التالي', 'Next')}
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Preview panel */}
      {preview && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setPreview(null)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-md mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <div className="flex items-start justify-between mb-4">
              <div>
                <h2 className="text-base font-semibold text-gray-900">{preview.name}</h2>
                <span className="text-xs text-gray-400 capitalize">{preview.category || 'general'}</span>
              </div>
              <button onClick={() => setPreview(null)} className="text-gray-400 hover:text-gray-600">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* WhatsApp bubble preview */}
            <div className="bg-[#ECE5DD] rounded-xl p-4 mb-4">
              <div className="bg-white rounded-lg px-3 py-2 shadow-sm max-w-xs text-sm text-gray-800 whitespace-pre-wrap leading-relaxed">
                {resolvedBody(preview)}
              </div>
            </div>

            {preview.variables?.length > 0 && (
              <div className="mb-4">
                <p className="text-xs text-gray-400 mb-2">{t('المتغيرات', 'Variables')}</p>
                <div className="flex flex-wrap gap-1.5">
                  {preview.variables.map((v, i) => (
                    <span key={v} className="text-xs font-mono px-2 py-0.5 bg-blue-50 text-blue-600 rounded">
                      {`{{${i + 1}}}`} → {v}
                    </span>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2 pt-3 border-t border-gray-100">
              {canWrite && (
                <button
                  onClick={() => { setPreview(null); openEdit(preview) }}
                  className="px-3 py-1.5 text-sm text-brand-600 font-medium hover:text-brand-700"
                >
                  {t('تعديل', 'Edit')}
                </button>
              )}
              <button onClick={() => setPreview(null)}
                className="px-3 py-1.5 text-sm text-gray-500 hover:text-gray-700">
                {t('إغلاق', 'Close')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create / Edit modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowModal(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-5">
              {editingId ? t('تعديل القالب', 'Edit Template') : t('قالب جديد', 'New Template')}
            </h2>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('الاسم', 'Name')} *</label>
                <input
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                  required
                  placeholder={t('مثال: رسالة ترحيب', 'e.g. Welcome Message')}
                />
              </div>

              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('الفئة', 'Category')}</label>
                <select
                  value={form.category}
                  onChange={e => setForm(f => ({ ...f, category: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white"
                >
                  {CATEGORIES.map(c => (
                    <option key={c} value={c}>{c.charAt(0).toUpperCase() + c.slice(1)}</option>
                  ))}
                </select>
              </div>

              <div>
                <label className="text-xs text-gray-500 mb-1 block">
                  {t('نص الرسالة', 'Message Body')} *
                  <span className="ms-1 text-gray-300">{t('استخدم {{1}} {{2}} للمتغيرات', 'Use {{1}} {{2}} for variables')}</span>
                </label>
                <textarea
                  value={form.body}
                  onChange={e => setForm(f => ({ ...f, body: e.target.value }))}
                  rows={5}
                  required
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none font-mono"
                  placeholder={t('مرحباً {{1}}، شكراً لتواصلك معنا.', 'Hello {{1}}, thank you for reaching out.')}
                />
                <p className="text-xs text-gray-300 mt-1">{form.body.length} {t('حرف', 'chars')}</p>
              </div>

              <div>
                <label className="text-xs text-gray-500 mb-2 block">{t('المتغيرات', 'Variables')}</label>
                <div className="flex gap-2 mb-2">
                  <input
                    value={variableInput}
                    onChange={e => setVariableInput(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addVariable() } }}
                    className="flex-1 px-3 py-1.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                    placeholder={t('اسم المتغير (مثل: الاسم)', 'Variable name (e.g. name)')}
                  />
                  <button
                    type="button"
                    onClick={addVariable}
                    className="px-3 py-1.5 text-sm bg-surface-100 text-surface-700 rounded-lg hover:bg-surface-200 transition-colors"
                  >
                    {t('إضافة', 'Add')}
                  </button>
                </div>
                {form.variables.length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {form.variables.map((v, i) => (
                      <span key={v} className="inline-flex items-center gap-1 text-xs font-mono px-2 py-0.5 bg-blue-50 text-blue-600 rounded">
                        {`{{${i + 1}}}`} {v}
                        <button
                          type="button"
                          onClick={() => removeVariable(v)}
                          className="hover:text-red-500 leading-none ms-0.5"
                        >×</button>
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={form.is_active}
                  onChange={e => setForm(f => ({ ...f, is_active: e.target.checked }))}
                  className="rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                />
                <span className="text-sm text-gray-700">{t('قالب نشط', 'Active template')}</span>
              </label>

              {saveError && (
                <p className="text-xs text-red-500">{saveError}</p>
              )}

              <div className="flex justify-end gap-3 pt-2 border-t border-gray-100">
                <button type="button" onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={saving}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {saving ? '...' : editingId ? t('حفظ', 'Save') : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
