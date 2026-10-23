'use client'
import { useCallback, useEffect, useState } from 'react'
import Link from 'next/link'
import clsx from 'clsx'
import { api, type Task, type TaskInput } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { formatDateTime } from '@/lib/date'

export const TASK_TYPE_LABELS: Record<Task['type'], { en: string; ar: string }> = {
  call: { en: 'Call', ar: 'مكالمة' },
  meeting: { en: 'Meeting', ar: 'اجتماع' },
  follow_up: { en: 'Follow-up', ar: 'متابعة' },
  todo: { en: 'To-do', ar: 'مهمة' },
}

const RELATED_PATHS: Record<string, string> = {
  contact: '/contacts/', deal: '/deals/', listing: '/listings/', viewing: '/viewings/',
}

interface Props {
  /** List filter, e.g. { mine: true } or { related_type: 'contact', related_id }. */
  filter: { mine?: boolean; status?: string; overdue?: boolean; related_type?: Task['related_type']; related_id?: string }
  /** When set, new tasks are linked to this record. */
  link?: { related_type: NonNullable<Task['related_type']>; related_id: string }
  canEdit: boolean
  compact?: boolean
  emptyText?: { en: string; ar: string }
}

/** Task list with quick add and complete, used on the Tasks page, record pages and the dashboard. */
export default function TaskList({ filter, link, canEdit, compact, emptyText }: Props) {
  const { t, lang } = useLang()
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState<{ title: string; type: Task['type']; due: string; priority: Task['priority'] }>({
    title: '', type: 'follow_up', due: '', priority: 'normal',
  })

  const key = JSON.stringify(filter)
  const load = useCallback(async () => {
    setLoading(true)
    try {
      const r = await api.tasks.list({ ...JSON.parse(key), limit: compact ? 10 : 100 })
      setTasks(r.data)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setLoading(false) }
  }, [key, compact])

  useEffect(() => { load() }, [load])

  const add = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const data: TaskInput = {
      title: form.title, type: form.type, priority: form.priority,
      due_at: form.due ? new Date(form.due).toISOString() : null,
      ...(link ?? {}),
    }
    try {
      await api.tasks.create(data)
      setForm({ title: '', type: 'follow_up', due: '', priority: 'normal' })
      setAdding(false)
      load()
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  const toggle = async (task: Task) => {
    try {
      await api.tasks.setStatus(task.id, task.status === 'done' ? 'open' : 'done')
      load()
    } catch (err: unknown) { setError(err instanceof Error ? err.message : 'Failed') }
  }

  const input = 'px-2.5 py-1.5 text-sm border border-surface-200 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40'

  return (
    <div>
      {canEdit && (
        adding ? (
          <form onSubmit={add} className="flex flex-wrap gap-2 mb-3">
            <input aria-label={t('عنوان المهمة', 'Task title')} required maxLength={300} placeholder={t('ماذا يجب أن يحدث؟', 'What needs to happen?')}
              className={input + ' flex-1 min-w-[180px]'} value={form.title} onChange={e => setForm(f => ({ ...f, title: e.target.value }))} />
            <select aria-label={t('النوع', 'Type')} className={input} value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value as Task['type'] }))}>
              {(Object.keys(TASK_TYPE_LABELS) as Task['type'][]).map(k => (
                <option key={k} value={k}>{lang === 'ar' ? TASK_TYPE_LABELS[k].ar : TASK_TYPE_LABELS[k].en}</option>
              ))}
            </select>
            <input aria-label={t('موعد الاستحقاق', 'Due')} type="datetime-local" className={input} value={form.due} onChange={e => setForm(f => ({ ...f, due: e.target.value }))} />
            <select aria-label={t('الأولوية', 'Priority')} className={input} value={form.priority} onChange={e => setForm(f => ({ ...f, priority: e.target.value as Task['priority'] }))}>
              <option value="low">{t('منخفضة', 'Low')}</option>
              <option value="normal">{t('عادية', 'Normal')}</option>
              <option value="high">{t('عالية', 'High')}</option>
            </select>
            <button type="submit" className="px-3 py-1.5 text-sm font-medium bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500">{t('إضافة', 'Add')}</button>
            <button type="button" onClick={() => setAdding(false)} className="px-3 py-1.5 text-sm text-surface-500 hover:text-surface-700 hover:bg-surface-100 rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500">{t('إلغاء', 'Cancel')}</button>
          </form>
        ) : (
          <button onClick={() => setAdding(true)} className="mb-3 text-xs font-medium text-primary-600 hover:text-primary-700 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 rounded px-1.5 py-0.5">
            {t('+ مهمة جديدة', '+ New task')}
          </button>
        )
      )}
      {error && <p role="alert" className="text-xs text-red-600 mb-2">{error}</p>}
      {loading ? (
        <p className="text-sm text-surface-400">{t('جاري التحميل...', 'Loading...')}</p>
      ) : tasks.length === 0 ? (
        <p className="text-sm text-surface-400">{emptyText ? (lang === 'ar' ? emptyText.ar : emptyText.en) : t('لا توجد مهام', 'No tasks')}</p>
      ) : (
        <ul className="divide-y divide-surface-100">
          {tasks.map(task => (
            <li key={task.id} className="py-2 flex items-start gap-3">
              <input type="checkbox" className="mt-1 rounded border-surface-300 text-primary-600 focus:ring-2 focus:ring-primary-500/40" checked={task.status === 'done'} disabled={!canEdit || task.status === 'cancelled'}
                onChange={() => toggle(task)} aria-label={task.status === 'done' ? t(`تحديد "${task.title}" كغير مكتملة`, `Mark "${task.title}" as incomplete`) : t(`تحديد "${task.title}" كمكتملة`, `Mark "${task.title}" as complete`)} />
              <div className="flex-1 min-w-0">
                <p className={clsx('text-sm text-surface-900', task.status !== 'open' && 'line-through text-surface-400')}>{task.title}</p>
                <p className="text-xs text-surface-500 flex flex-wrap gap-x-2">
                  <span>{lang === 'ar' ? TASK_TYPE_LABELS[task.type].ar : TASK_TYPE_LABELS[task.type].en}</span>
                  {task.due_at && (
                    <span className={clsx(task.overdue && 'text-red-600 font-medium')}>
                      {task.overdue ? t('متأخرة · ', 'Overdue · ') : ''}{formatDateTime(task.due_at)}
                    </span>
                  )}
                  {task.priority === 'high' && <span className="text-amber-700">{t('أولوية عالية', 'High priority')}</span>}
                  {!compact && task.assignee_name && <span>{task.assignee_name}</span>}
                  {!link && task.related_type && task.related_id && (
                    RELATED_PATHS[task.related_type]
                      ? <Link href={RELATED_PATHS[task.related_type] + task.related_id} className="text-primary-600 hover:underline">{task.related_label || task.related_type}</Link>
                      : <span>{task.related_label || task.related_type}</span>
                  )}
                </p>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
