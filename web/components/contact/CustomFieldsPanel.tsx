'use client'
import { useEffect, useState } from 'react'
import { api, type CustomField } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface Props {
  entityType: CustomField['entity_type']
  entityId: string
  canEdit: boolean
}

/** Shows and edits a record's custom fields; renders nothing when none are defined. */
export default function CustomFieldsPanel({ entityType, entityId, canEdit }: Props) {
  const { t } = useLang()
  const [fields, setFields] = useState<CustomField[]>([])
  const [values, setValues] = useState<Record<string, string>>({})
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([api.customFields.list(entityType), api.customFields.values(entityType, entityId)])
      .then(([f, v]) => { setFields(f); setValues(v) })
      .catch(() => {})
  }, [entityType, entityId])

  if (fields.length === 0) return null

  const save = async () => {
    setError('')
    try {
      // Send every field so cleared ones are removed.
      const body = Object.fromEntries(fields.map(f => [f.name, draft[f.name] ?? '']))
      await api.customFields.setValues(entityType, entityId, body)
      setValues(Object.fromEntries(Object.entries(body).filter(([, v]) => v !== '')))
      setEditing(false)
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  const input = 'w-full px-2.5 py-1.5 text-sm border border-surface-200 rounded-lg bg-white'

  return (
    <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-surface-900">{t('حقول إضافية', 'Additional fields')}</h3>
        {canEdit && !editing && (
          <button onClick={() => { setDraft(values); setEditing(true) }} className="text-xs text-primary-600 hover:underline">{t('تعديل', 'Edit')}</button>
        )}
      </div>
      <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {fields.map(f => (
          <div key={f.id}>
            <dt className="text-xs text-surface-500">
              <label htmlFor={`cf-${f.id}`}>{f.label}{f.required ? ' *' : ''}</label>
            </dt>
            <dd className="text-sm text-surface-900">
              {!editing ? (values[f.name] ? (f.type === 'boolean' ? (values[f.name] === 'true' ? t('نعم', 'Yes') : t('لا', 'No')) : values[f.name]) : '—') : (
                f.type === 'select' ? (
                  <select id={`cf-${f.id}`} className={input} value={draft[f.name] ?? ''} onChange={e => setDraft(d => ({ ...d, [f.name]: e.target.value }))}>
                    <option value="">—</option>
                    {f.options.map(o => <option key={o} value={o}>{o}</option>)}
                  </select>
                ) : f.type === 'boolean' ? (
                  <select id={`cf-${f.id}`} className={input} value={draft[f.name] ?? ''} onChange={e => setDraft(d => ({ ...d, [f.name]: e.target.value }))}>
                    <option value="">—</option>
                    <option value="true">{t('نعم', 'Yes')}</option>
                    <option value="false">{t('لا', 'No')}</option>
                  </select>
                ) : (
                  <input id={`cf-${f.id}`} className={input} type={f.type === 'number' ? 'number' : f.type === 'date' ? 'date' : 'text'}
                    value={draft[f.name] ?? ''} onChange={e => setDraft(d => ({ ...d, [f.name]: e.target.value }))} />
                )
              )}
            </dd>
          </div>
        ))}
      </dl>
      {editing && (
        <div className="flex gap-2 mt-3">
          <button onClick={save} className="px-3 py-1.5 text-xs font-medium bg-primary-600 text-white rounded-lg">{t('حفظ', 'Save')}</button>
          <button onClick={() => setEditing(false)} className="px-3 py-1.5 text-xs text-surface-500">{t('إلغاء', 'Cancel')}</button>
        </div>
      )}
      {error && <p role="alert" className="text-xs text-red-600 mt-2">{error}</p>}
    </div>
  )
}
