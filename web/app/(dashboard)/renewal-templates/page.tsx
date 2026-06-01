'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { RenewalTemplate } from '@/types'

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'ar', label: 'Arabic / عربي' },
]

const emptyForm = {
  template_name: '',
  email_subject: '',
  email_body: '',
  whatsapp_message: '',
  language: 'en',
}

export default function RenewalTemplatesPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [templates, setTemplates] = useState<RenewalTemplate[]>([])
  const [loading, setLoading] = useState(true)

  const [showModal, setShowModal] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState({ ...emptyForm })
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')
  const [deleting, setDeleting] = useState<string | null>(null)

  const [activeTab, setActiveTab] = useState<'email' | 'whatsapp'>('email')

  useEffect(() => { load() }, [])

  const load = async () => {
    setLoading(true)
    try {
      const res: any = await api.leaseRenewals.templates.list()
      setTemplates(res?.data ?? res ?? [])
    } catch {
      setTemplates([])
    } finally {
      setLoading(false)
    }
  }

  const openCreate = () => {
    setForm({ ...emptyForm })
    setEditingId(null)
    setSaveError('')
    setActiveTab('email')
    setShowModal(true)
  }

  const openEdit = (tpl: RenewalTemplate) => {
    setForm({
      template_name: tpl.template_name,
      email_subject: tpl.email_subject,
      email_body: tpl.email_body,
      whatsapp_message: tpl.whatsapp_message,
      language: tpl.language || 'en',
    })
    setEditingId(tpl.id)
    setSaveError('')
    setActiveTab('email')
    setShowModal(true)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.template_name.trim()) return
    setSaving(true)
    setSaveError('')
    try {
      if (editingId) {
        await api.leaseRenewals.templates.update(editingId, form)
      } else {
        await api.leaseRenewals.templates.create(form)
      }
      setShowModal(false)
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
      await api.leaseRenewals.templates.delete(id)
      load()
    } catch {
      alert(t('فشل الحذف', 'Failed to delete'))
    } finally {
      setDeleting(null)
    }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('قوالب التجديد', 'Renewal Templates')} />

      <div className="flex-1 overflow-y-auto p-6">
        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : templates.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-4 text-center">
            <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-8 h-8 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </div>
            <div>
              <p className="text-sm text-gray-400">{t('لا توجد قوالب تجديد بعد', 'No renewal templates yet')}</p>
              <p className="text-xs text-gray-300 mt-1">{t('أنشئ قالباً للبريد والواتساب', 'Create email + WhatsApp templates for renewals')}</p>
            </div>
            {isAdmin && (
              <button onClick={openCreate}
                className="mt-2 px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
                {t('+ قالب جديد', '+ New Template')}
              </button>
            )}
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <p className="text-sm text-gray-500">{templates.length} {t('قالب', 'templates')}</p>
              {isAdmin && (
                <button onClick={openCreate}
                  className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 transition-colors">
                  {t('+ قالب جديد', '+ New Template')}
                </button>
              )}
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {templates.map(tpl => (
                <div key={tpl.id} className="bg-white rounded-xl border border-gray-100 p-5 flex flex-col gap-3">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="font-medium text-gray-900 text-sm">{tpl.template_name}</p>
                      <span className="text-[11px] px-1.5 py-0.5 rounded bg-surface-100 text-surface-600 uppercase font-mono mt-1 inline-block">
                        {tpl.language || 'en'}
                      </span>
                    </div>
                    {isAdmin && (
                      <div className="flex gap-2 shrink-0">
                        <button onClick={() => openEdit(tpl)} className="text-xs text-brand-600 hover:text-brand-700 font-medium">
                          {t('تعديل', 'Edit')}
                        </button>
                        <button
                          onClick={() => handleDelete(tpl.id)}
                          disabled={deleting === tpl.id}
                          className="text-xs text-red-500 hover:text-red-700 font-medium disabled:opacity-30"
                        >
                          {t('حذف', 'Del')}
                        </button>
                      </div>
                    )}
                  </div>

                  {tpl.email_subject && (
                    <div>
                      <p className="text-[10px] text-gray-400 uppercase font-medium mb-0.5">{t('موضوع البريد', 'Email subject')}</p>
                      <p className="text-xs text-gray-600 line-clamp-1">{tpl.email_subject}</p>
                    </div>
                  )}

                  {tpl.whatsapp_message && (
                    <div>
                      <p className="text-[10px] text-gray-400 uppercase font-medium mb-0.5">{t('رسالة واتساب', 'WhatsApp msg')}</p>
                      <p className="text-xs text-gray-600 line-clamp-2">{tpl.whatsapp_message}</p>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setShowModal(false)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h2 className="text-lg font-semibold text-gray-900 mb-5">
              {editingId ? t('تعديل القالب', 'Edit Template') : t('قالب جديد', 'New Template')}
            </h2>

            <form onSubmit={handleSave} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="col-span-2">
                  <label className="text-xs text-gray-500 mb-1 block">{t('اسم القالب', 'Template Name')} *</label>
                  <input
                    value={form.template_name}
                    onChange={e => setForm(f => ({ ...f, template_name: e.target.value }))}
                    required
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                    placeholder={t('مثال: تجديد شهري', 'e.g. Monthly Renewal')}
                  />
                </div>
                <div className="col-span-2">
                  <label className="text-xs text-gray-500 mb-1 block">{t('اللغة', 'Language')}</label>
                  <select
                    value={form.language}
                    onChange={e => setForm(f => ({ ...f, language: e.target.value }))}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white"
                  >
                    {LANGUAGES.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
                  </select>
                </div>
              </div>

              {/* Tab switcher */}
              <div className="flex gap-1 p-1 bg-surface-100 rounded-lg">
                {(['email', 'whatsapp'] as const).map(tab => (
                  <button
                    key={tab}
                    type="button"
                    onClick={() => setActiveTab(tab)}
                    className={clsx(
                      'flex-1 py-1.5 text-xs font-medium rounded-md transition-colors',
                      activeTab === tab ? 'bg-white shadow-sm text-gray-900' : 'text-gray-500 hover:text-gray-700'
                    )}
                  >
                    {tab === 'email' ? t('البريد الإلكتروني', 'Email') : 'WhatsApp'}
                  </button>
                ))}
              </div>

              {activeTab === 'email' ? (
                <>
                  <div>
                    <label className="text-xs text-gray-500 mb-1 block">{t('موضوع البريد', 'Email Subject')}</label>
                    <input
                      value={form.email_subject}
                      onChange={e => setForm(f => ({ ...f, email_subject: e.target.value }))}
                      className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
                      placeholder={t('تجديد عقد الإيجار', 'Lease Renewal Notice')}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-gray-500 mb-1 block">{t('نص البريد', 'Email Body')}</label>
                    <textarea
                      value={form.email_body}
                      onChange={e => setForm(f => ({ ...f, email_body: e.target.value }))}
                      rows={6}
                      className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
                      placeholder={t('عزيزي المستأجر...', 'Dear tenant...')}
                    />
                  </div>
                </>
              ) : (
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">{t('رسالة واتساب', 'WhatsApp Message')}</label>
                  <textarea
                    value={form.whatsapp_message}
                    onChange={e => setForm(f => ({ ...f, whatsapp_message: e.target.value }))}
                    rows={6}
                    className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none font-mono"
                    placeholder={t('مرحباً {{name}}، ...', 'Hello {{name}}, ...')}
                  />
                  <p className="text-xs text-gray-300 mt-1">{form.whatsapp_message.length} {t('حرف', 'chars')}</p>
                </div>
              )}

              {saveError && <p className="text-xs text-red-500">{saveError}</p>}

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
