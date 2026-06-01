'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { DocumentTemplate, PaginatedResult } from '@/types'
import clsx from 'clsx'

const docTypes = ['lease', 'contract', 'invoice', 'agreement', 'other'] as const

export default function DocumentsPage() {
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [templates, setTemplates] = useState<DocumentTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({ template_name: '', document_type: '', template_content: '', language: 'en', signature_required: false })
  const [error, setError] = useState('')

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.documents.templates.list(page, limit) as PaginatedResult<DocumentTemplate>
      setTemplates(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch { setTemplates([]) } finally { setLoading(false) }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setCreating(true)
    try {
      await api.documents.templates.create({
        template_name: form.template_name,
        document_type: form.document_type,
        template_content: form.template_content,
        language: form.language,
        signature_required: form.signature_required,
      })
      setShowCreate(false)
      setForm({ template_name: '', document_type: '', template_content: '', language: 'en', signature_required: false })
      setPage(1); load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Failed'))
    } finally { setCreating(false) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('حذف القالب؟', 'Delete this template?'))) return
    try { await api.documents.templates.delete(id); load() } catch {}
  }

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('قوالب المستندات', 'Document Templates')} />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface-900">
            {t('قوالب المستندات', 'Templates')}
            <span className="ml-2 text-sm font-normal text-surface-500">({total})</span>
          </h2>
          {isAgent && (
            <button onClick={() => setShowCreate(!showCreate)}
              className="px-4 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 transition-colors">
              + {t('قالب جديد', 'New Template')}
            </button>
          )}
        </div>

        {/* Create Form */}
        {showCreate && (
          <div className="bg-white rounded-xl border border-surface-200/70 p-5 mb-6 shadow-sm">
            <h3 className="text-sm font-semibold text-surface-900 mb-4">{t('إنشاء قالب', 'Create Template')}</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-surface-600 mb-1">{t('اسم القالب', 'Template Name')} *</label>
                  <input required value={form.template_name} onChange={e => setForm(f => ({ ...f, template_name: e.target.value }))}
                    className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40"
                    placeholder={t('مثل: عقد إيجار قياسي', 'e.g. Standard Lease')} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-600 mb-1">{t('نوع المستند', 'Document Type')} *</label>
                  <select required value={form.document_type} onChange={e => setForm(f => ({ ...f, document_type: e.target.value }))}
                    className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40">
                    <option value="">{t('اختر...', 'Select...')}</option>
                    {docTypes.map(dt => <option key={dt} value={dt}>{dt}</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-surface-600 mb-1">{t('اللغة', 'Language')}</label>
                  <select value={form.language} onChange={e => setForm(f => ({ ...f, language: e.target.value }))}
                    className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40">
                    <option value="en">English</option>
                    <option value="ar">العربية</option>
                  </select>
                </div>
                <div className="flex items-end pb-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={form.signature_required} onChange={e => setForm(f => ({ ...f, signature_required: e.target.checked }))}
                      className="w-4 h-4 rounded accent-primary-600" />
                    <span className="text-xs font-medium text-surface-600">{t('يتطلب توقيع', 'Requires signature')}</span>
                  </label>
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('المحتوى', 'Content')}</label>
                <textarea rows={6} value={form.template_content} onChange={e => setForm(f => ({ ...f, template_content: e.target.value }))}
                  className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary-500/40"
                  placeholder={t('استخدم {{placeholders}} للحقول المتغيرة', 'Use {{placeholders}} for dynamic fields')} />
              </div>
              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              <div className="flex gap-2 pt-2">
                <button type="button" onClick={() => { setShowCreate(false); setError('') }}
                  className="flex-1 py-2 border border-surface-200 rounded-lg text-sm text-surface-600 hover:bg-surface-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={creating}
                  className="flex-1 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-60">
                  {creating ? t('جاري الإنشاء...', 'Creating...') : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Templates List */}
        {loading ? (
          <div className="text-center py-12 text-surface-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : templates.length === 0 ? (
          <div className="bg-white rounded-xl border border-surface-200/70 p-12 text-center">
            <p className="text-surface-400 text-sm mb-3">{t('لا توجد قوالب', 'No templates yet')}</p>
            {isAgent && <button onClick={() => setShowCreate(true)} className="text-primary-600 text-sm font-medium hover:underline">+ {t('أنشئ أول قالب', 'Create your first template')}</button>}
          </div>
        ) : (
          <div className="space-y-2">
            {templates.map(tmpl => (
              <div key={tmpl.id} className="bg-white rounded-xl border border-surface-200/70 p-4 hover:shadow-sm transition-shadow">
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0">
                    <Link href={`/documents/templates/${tmpl.id}`} className="font-medium text-surface-900 hover:text-primary-600 text-sm">
                      {tmpl.template_name}
                    </Link>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-[10px] font-medium px-1.5 py-0.5 bg-surface-100 text-surface-600 rounded">{tmpl.document_type}</span>
                      <span className="text-[10px] text-surface-400">{tmpl.language?.toUpperCase()}</span>
                      {tmpl.signature_required && (
                        <span className="text-[10px] font-medium px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded">{t('توقيع', 'Signature')}</span>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-1.5 shrink-0">
                    <Link href={`/documents/templates/${tmpl.id}`}
                      className="px-3 py-1 text-xs bg-surface-100 text-surface-700 rounded-md hover:bg-surface-200 font-medium">
                      {t('تعديل', 'Edit')}
                    </Link>
                    {isAgent && (
                      <button onClick={() => handleDelete(tmpl.id)}
                        className="px-3 py-1 text-xs bg-red-50 text-red-600 rounded-md hover:bg-red-100 font-medium">
                        {t('حذف', 'Delete')}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {totalPages > 1 && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
              className="px-4 py-2 border border-surface-200 rounded-lg disabled:opacity-40 hover:bg-surface-50 text-sm">{t('السابق', 'Prev')}</button>
            <span className="flex items-center px-3 text-sm text-surface-500">{page} / {totalPages}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}
              className="px-4 py-2 border border-surface-200 rounded-lg disabled:opacity-40 hover:bg-surface-50 text-sm">{t('التالي', 'Next')}</button>
          </div>
        )}
      </div>
    </div>
  )
}
