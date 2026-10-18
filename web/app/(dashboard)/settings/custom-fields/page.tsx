'use client'
import { useCallback, useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api, type CustomField } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

const ENTITIES: CustomField['entity_type'][] = ['contact', 'lead', 'deal', 'listing']
const TYPES: CustomField['type'][] = ['text', 'number', 'date', 'select', 'boolean']

export default function CustomFieldsPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const router = useRouter()
  const [fields, setFields] = useState<CustomField[]>([])
  const [form, setForm] = useState({ entity_type: 'contact' as CustomField['entity_type'], name: '', label: '', type: 'text' as CustomField['type'], options: '', required: false })
  const [error, setError] = useState('')

  const load = useCallback(() => { api.customFields.list().then(setFields).catch(() => {}) }, [])
  useEffect(() => {
    if (user && user.role !== 'admin') { router.push('/dashboard'); return }
    load()
  }, [user, router, load])

  const create = async (e: React.FormEvent) => {
    e.preventDefault(); setError('')
    try {
      await api.customFields.create({
        entity_type: form.entity_type, name: form.name, label: form.label, type: form.type, required: form.required,
        options: form.type === 'select' ? form.options.split(',').map(s => s.trim()).filter(Boolean) : [],
      })
      setForm(f => ({ ...f, name: '', label: '', options: '' }))
      load()
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }
  const remove = async (f: CustomField) => {
    if (!confirm(t('حذف الحقل وجميع قيمه؟', `Delete "${f.label}" and all its values?`))) return
    await api.customFields.delete(f.id).catch(() => {}); load()
  }

  if (user?.role !== 'admin') return null
  const input = 'px-2.5 py-1.5 text-sm border border-surface-200 rounded-lg bg-white'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('الحقول المخصصة', 'Custom Fields')} />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <form onSubmit={create} className="bg-white rounded-2xl border border-surface-200/70 p-5 grid grid-cols-1 sm:grid-cols-3 gap-3">
            <label className="text-xs text-surface-500">{t('السجل', 'Record')}
              <select className={input + ' w-full'} value={form.entity_type} onChange={e => setForm(f => ({ ...f, entity_type: e.target.value as CustomField['entity_type'] }))}>
                {ENTITIES.map(x => <option key={x} value={x}>{x}</option>)}
              </select>
            </label>
            <label className="text-xs text-surface-500">{t('الاسم الظاهر', 'Label')}
              <input required className={input + ' w-full'} value={form.label} onChange={e => setForm(f => ({ ...f, label: e.target.value }))} />
            </label>
            <label className="text-xs text-surface-500">{t('الاسم البرمجي', 'Machine name (CSV: cf_<name>)')}
              <input required dir="ltr" pattern="[a-z][a-z0-9_]*" className={input + ' w-full'} value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="budget" />
            </label>
            <label className="text-xs text-surface-500">{t('النوع', 'Type')}
              <select className={input + ' w-full'} value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value as CustomField['type'] }))}>
                {TYPES.map(x => <option key={x} value={x}>{x}</option>)}
              </select>
            </label>
            {form.type === 'select' && (
              <label className="text-xs text-surface-500 sm:col-span-2">{t('الخيارات (مفصولة بفواصل)', 'Options (comma separated)')}
                <input required className={input + ' w-full'} value={form.options} onChange={e => setForm(f => ({ ...f, options: e.target.value }))} />
              </label>
            )}
            <label className="flex items-center gap-2 text-sm text-surface-700">
              <input type="checkbox" checked={form.required} onChange={e => setForm(f => ({ ...f, required: e.target.checked }))} /> {t('مطلوب', 'Required')}
            </label>
            <button className="sm:col-span-3 justify-self-start px-3 py-2 text-sm font-medium bg-primary-600 text-white rounded-lg">{t('إضافة حقل', 'Add field')}</button>
            {error && <p role="alert" className="sm:col-span-3 text-xs text-red-600">{error}</p>}
          </form>
          <table className="w-full text-sm bg-white rounded-2xl border border-surface-200/70 overflow-hidden">
            <thead className="bg-surface-50 text-xs text-surface-500">
              <tr>
                <th scope="col" className="px-4 py-2 text-start">{t('السجل', 'Record')}</th>
                <th scope="col" className="px-4 py-2 text-start">{t('الحقل', 'Field')}</th>
                <th scope="col" className="px-4 py-2 text-start">{t('النوع', 'Type')}</th>
                <th scope="col" className="px-4 py-2"><span className="sr-only">{t('إجراءات', 'Actions')}</span></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-100">
              {fields.length === 0 && <tr><td colSpan={4} className="px-4 py-4 text-surface-400">{t('لا توجد حقول', 'No custom fields yet')}</td></tr>}
              {fields.map(f => (
                <tr key={f.id}>
                  <td className="px-4 py-2">{f.entity_type}</td>
                  <td className="px-4 py-2">{f.label} <span dir="ltr" className="text-xs text-surface-400">cf_{f.name}</span>{f.required ? ' *' : ''}</td>
                  <td className="px-4 py-2">{f.type}{f.type === 'select' ? ` (${f.options.join(', ')})` : ''}</td>
                  <td className="px-4 py-2 text-end"><button onClick={() => remove(f)} className="text-xs text-red-600 hover:underline">{t('حذف', 'Delete')}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  )
}
