'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { DocumentTemplate } from '@/types'

export default function EditTemplatePage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()

  const [template, setTemplate] = useState<DocumentTemplate | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [form, setForm] = useState({ template_name: '', template_content: '' })
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  useEffect(() => { load() }, [id])

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const data = await api.documents.templates.get(id) as DocumentTemplate
      setTemplate(data)
      setForm({ template_name: data.template_name, template_content: data.template_content })
    } catch { setTemplate(null) } finally { setLoading(false) }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setSaving(true)
    try {
      await api.documents.templates.update(id, {
        template_name: form.template_name,
        template_content: form.template_content,
      })
      setSuccess(t('تم الحفظ', 'Saved'))
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('فشل الحفظ', 'Save failed'))
    } finally { setSaving(false) }
  }

  if (loading) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!template) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('غير موجود', 'Template not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={`${t('تعديل', 'Edit')}: ${template.template_name}`} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/documents" className="text-xs text-surface-500 hover:text-surface-700 font-medium flex items-center gap-1">
          ← {t('العودة إلى القوالب', 'Back to Templates')}
        </Link>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        <div className="bg-white rounded-xl border border-surface-200/70 p-5">
          <div className="flex items-center gap-2 mb-4">
            <h2 className="text-sm font-semibold text-surface-900">{t('تحرير القالب', 'Edit Template')}</h2>
            <span className="text-[10px] font-medium px-2 py-0.5 bg-surface-100 text-surface-600 rounded">{template.document_type}</span>
          </div>
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-surface-600 mb-1">{t('اسم القالب', 'Template Name')} *</label>
              <input required value={form.template_name} onChange={e => setForm(f => ({ ...f, template_name: e.target.value }))}
                className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" />
            </div>
            <div>
              <label className="block text-xs font-medium text-surface-600 mb-1">{t('المحتوى', 'Content')}</label>
              <p className="text-[10px] text-surface-400 mb-1">
                {t('استخدم {{field_name}} للحقول المتغيرة (مثل {{tenant_name}})', 'Use {{field_name}} for placeholders (e.g. {{tenant_name}})')}
              </p>
              <textarea required rows={12} value={form.template_content} onChange={e => setForm(f => ({ ...f, template_content: e.target.value }))}
                className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-primary-500/40" />
            </div>
            <div className="flex gap-3">
              <button type="submit" disabled={saving}
                className="px-6 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-60">
                {saving ? t('جاري الحفظ...', 'Saving...') : t('حفظ التغييرات', 'Save Changes')}
              </button>
              <button type="button" onClick={() => router.push('/documents')}
                className="px-6 py-2 border border-surface-200 text-surface-700 rounded-lg text-sm hover:bg-surface-50 font-medium">
                {t('إلغاء', 'Cancel')}
              </button>
            </div>
          </form>
        </div>

        <div className="bg-white rounded-xl border border-surface-200/70 p-5">
          <h3 className="text-sm font-semibold text-surface-900 mb-3">{t('معاينة', 'Preview')}</h3>
          <div className="bg-surface-50 p-4 rounded-lg border border-surface-200 font-serif text-surface-800 whitespace-pre-wrap break-words text-sm">
            {form.template_content || t('لا يوجد محتوى', 'No content yet')}
          </div>
        </div>
      </div>
    </div>
  )
}
