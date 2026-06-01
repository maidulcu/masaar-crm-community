'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { Inspection, InspectionTemplate } from '@/types'

const statusColor: Record<string, string> = {
  completed:  'bg-green-100 text-green-700',
  in_progress:'bg-blue-100 text-blue-700',
  scheduled:  'bg-yellow-100 text-yellow-700',
  cancelled:  'bg-red-100 text-red-700',
}

const severityColor: Record<string, string> = {
  green:  'text-green-600',
  yellow: 'text-yellow-600',
  red:    'text-red-600',
}

const emptyForm = {
  property_id: '',
  inspection_type: 'general',
  scheduled_date: '',
  template_id: '',
  tenant_id: '',
  inspector_id: '',
}

export default function InspectionsPage() {
  const { lang, t } = useLang()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const isAdmin = user?.role === 'admin'

  const [inspections, setInspections] = useState<Inspection[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [showCreate, setShowCreate] = useState(false)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState(emptyForm)
  const [templates, setTemplates] = useState<InspectionTemplate[]>([])

  const [showTemplateModal, setShowTemplateModal] = useState(false)
  const [savingTemplate, setSavingTemplate] = useState(false)
  const [templateForm, setTemplateForm] = useState({ template_name: '', inspection_type: 'general', estimated_duration_minutes: 60 })

  useEffect(() => {
    loadInspections()
  }, [page])

  const loadInspections = async () => {
    setLoading(true)
    try {
      const res: any = await api.inspection.list({ limit, page })
      setInspections(res?.data ?? [])
      setTotal(res?.meta?.total ?? 0)
    } catch {
      setInspections([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const openCreate = async () => {
    setForm(emptyForm)
    try {
      const res: any = await api.inspection.listTemplates()
      setTemplates(res?.data ?? res ?? [])
    } catch {
      setTemplates([])
    }
    setShowCreate(true)
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.property_id || !form.scheduled_date) return
    setCreating(true)
    try {
      const payload: Record<string, unknown> = {
        property_id: form.property_id,
        inspection_type: form.inspection_type,
        scheduled_date: new Date(form.scheduled_date).toISOString(),
      }
      if (form.template_id) payload.template_id = form.template_id
      if (form.tenant_id) payload.tenant_id = form.tenant_id
      if (form.inspector_id) payload.inspector_id = form.inspector_id
      await api.inspection.create(payload)
      setShowCreate(false)
      setPage(1)
      loadInspections()
    } catch {
      alert(t('فشل إنشاء الفحص', 'Failed to create inspection'))
    } finally {
      setCreating(false)
    }
  }

  const handleCreateTemplate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!templateForm.template_name) return
    setSavingTemplate(true)
    try {
      await api.inspection.createTemplate(templateForm)
      setShowTemplateModal(false)
      setTemplateForm({ template_name: '', inspection_type: 'general', estimated_duration_minutes: 60 })
    } catch {
      alert(t('فشل إنشاء القالب', 'Failed to create template'))
    } finally {
      setSavingTemplate(false)
    }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الفحوصات', 'Inspections')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : inspections.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد فحوصات', 'No inspections yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('أنشئ فحصاً لبدء المتابعة', 'Create an inspection to start tracking')}</p>
            </div>
            {isAgent && (
              <button onClick={openCreate}
                className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
                {t('+ فحص جديد', '+ New Inspection')}
              </button>
            )}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{total} {t('فحص', 'inspections')}</p>
              <div className="flex gap-2">
                {isAdmin && (
                  <button onClick={() => setShowTemplateModal(true)}
                    className="px-4 py-2 text-sm font-medium border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors">
                    {t('+ قالب', '+ Template')}
                  </button>
                )}
                {isAgent && (
                  <button onClick={openCreate}
                    className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                    {t('+ فحص جديد', '+ New Inspection')}
                  </button>
                )}
              </div>
            </div>

            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('النوع', 'Type')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('التاريخ', 'Date')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('المستوى', 'Severity')}</th>
                    <th className="px-5 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {inspections.map(insp => (
                    <tr key={insp.id} className="hover:bg-gray-50 transition-colors">
                      <td className="px-5 py-3.5 font-medium text-gray-900 capitalize">{insp.inspection_type.replace(/_/g, ' ')}</td>
                      <td className="px-5 py-3.5 text-gray-600 text-xs">
                        {new Date(insp.scheduled_date).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE')}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', statusColor[insp.status])}>
                          {insp.status.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td className={clsx('px-5 py-3.5 text-xs font-medium', severityColor[insp.severity_level || ''] || 'text-gray-400')}>
                        {insp.severity_level || '—'}
                      </td>
                      <td className="px-5 py-3.5 text-end">
                        <Link href={`/inspections/${insp.id}`}
                          className="text-xs text-brand-600 hover:underline font-medium">
                          {t('عرض', 'View')}
                        </Link>
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

      {/* Create Inspection Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('فحص جديد', 'New Inspection')}</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('معرف العقار', 'Property ID')} *</label>
                <input value={form.property_id} onChange={e => setForm(f => ({ ...f, property_id: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('نوع الفحص', 'Inspection Type')}</label>
                <select value={form.inspection_type} onChange={e => setForm(f => ({ ...f, inspection_type: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="general">{t('عام', 'General')}</option>
                  <option value="pre_lease">{t('قبل الإيجار', 'Pre-Lease')}</option>
                  <option value="end_lease">{t('نهاية الإيجار', 'End of Lease')}</option>
                  <option value="damage_assessment">{t('تقييم الأضرار', 'Damage Assessment')}</option>
                  <option value="safety">{t('السلامة', 'Safety')}</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('تاريخ الفحص', 'Scheduled Date')} *</label>
                <input type="date" value={form.scheduled_date} onChange={e => setForm(f => ({ ...f, scheduled_date: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('القالب', 'Template')}</label>
                <select value={form.template_id} onChange={e => setForm(f => ({ ...f, template_id: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="">{t('بدون قالب', 'No template')}</option>
                  {templates.map(tpl => (
                    <option key={tpl.id} value={tpl.id}>{tpl.template_name}</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف المفتش', 'Inspector ID')}</label>
                  <input value={form.inspector_id} onChange={e => setForm(f => ({ ...f, inspector_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('معرف المستأجر', 'Tenant ID')}</label>
                  <input value={form.tenant_id} onChange={e => setForm(f => ({ ...f, tenant_id: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowCreate(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={creating}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {creating ? '...' : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Create Template Modal */}
      {showTemplateModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowTemplateModal(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('قالب فحص جديد', 'New Inspection Template')}</h2>
            <form onSubmit={handleCreateTemplate} className="space-y-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('اسم القالب', 'Template Name')} *</label>
                <input value={templateForm.template_name} onChange={e => setTemplateForm(f => ({ ...f, template_name: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" required />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('نوع الفحص', 'Inspection Type')}</label>
                <select value={templateForm.inspection_type} onChange={e => setTemplateForm(f => ({ ...f, inspection_type: e.target.value }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                  <option value="general">{t('عام', 'General')}</option>
                  <option value="pre_lease">{t('قبل الإيجار', 'Pre-Lease')}</option>
                  <option value="end_lease">{t('نهاية الإيجار', 'End of Lease')}</option>
                  <option value="damage_assessment">{t('تقييم الأضرار', 'Damage Assessment')}</option>
                  <option value="safety">{t('السلامة', 'Safety')}</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">{t('المدة المقدرة (دقائق)', 'Est. Duration (min)')}</label>
                <input type="number" min={1} value={templateForm.estimated_duration_minutes} onChange={e => setTemplateForm(f => ({ ...f, estimated_duration_minutes: parseInt(e.target.value) || 60 }))}
                  className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowTemplateModal(false)}
                  className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={savingTemplate}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
                  {savingTemplate ? '...' : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
