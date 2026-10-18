'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { api, type LeaseExtras, type PaymentReminderRow } from '@/lib/api'
import { formatDate, formatDateTime } from '@/lib/date'
import { useLang } from '@/context/LangContext'
import type { User } from '@/types'

const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'

// The lease's agent (who receives its alerts), the Ejari registration expiry, a shortcut to
// start the renewal, and the rent reminders sent to the tenant.
export default function LeaseExtrasPanel({ leaseId, canEdit }: { leaseId: string; canEdit: boolean }) {
  const { t } = useLang()
  const [extras, setExtras] = useState<LeaseExtras | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [draft, setDraft] = useState({ agent_id: '', ejari_expiry: '', rent_reminders: true })
  const [reminders, setReminders] = useState<PaymentReminderRow[]>([])
  const [msg, setMsg] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    api.leaseOps.extras(leaseId).then((e) => {
      setExtras(e)
      setDraft({ agent_id: e.agent_id ?? '', ejari_expiry: e.ejari_expiry?.slice(0, 10) ?? '', rent_reminders: e.rent_reminders })
    }).catch(() => {})
    api.leaseOps.reminders(leaseId).then(setReminders).catch(() => {})
    if (canEdit) api.users.list().then((u) => setUsers(Array.isArray(u) ? (u as User[]) : [])).catch(() => {})
  }, [leaseId, canEdit])

  const save = async () => {
    setError(''); setMsg('')
    try {
      await api.leaseOps.setExtras(leaseId, { agent_id: draft.agent_id || null, ejari_expiry: draft.ejari_expiry, rent_reminders: draft.rent_reminders })
      setMsg(t('تم الحفظ', 'Saved'))
    } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }
  const startRenewal = async () => {
    setError(''); setMsg('')
    try {
      await api.leaseRenewals.initiate(leaseId)
      setMsg(t('بدأ التجديد. تابعه من صفحة التجديدات.', 'Renewal started. Continue it on the Renewals page.'))
    } catch (e) { setError(e instanceof Error ? e.message : 'Failed') }
  }

  return (
    <div className="bg-white rounded-xl border border-gray-100 p-5 space-y-4">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-xs text-gray-600">{t('الوكيل المسؤول', 'Agent responsible')}
          {canEdit ? (
            <select className={`${input} block mt-1 min-w-[10rem]`} value={draft.agent_id} onChange={(e) => setDraft((d) => ({ ...d, agent_id: e.target.value }))}>
              <option value="">{t('— منشئ العقد —', '— lease creator —')}</option>
              {users.map((u) => <option key={u.id} value={u.id}>{u.name}</option>)}
            </select>
          ) : <span className="block mt-1 text-sm text-gray-900">{extras?.agent_name || '—'}</span>}
        </label>
        <label className="text-xs text-gray-600">{t('انتهاء تسجيل إيجاري', 'Ejari registration expiry')}
          {canEdit ? (
            <input type="date" className={`${input} block mt-1`} value={draft.ejari_expiry} onChange={(e) => setDraft((d) => ({ ...d, ejari_expiry: e.target.value }))} />
          ) : <span className="block mt-1 text-sm text-gray-900">{extras?.ejari_expiry ? formatDate(extras.ejari_expiry) : '—'}</span>}
        </label>
        <label className="flex items-center gap-1.5 text-xs text-gray-600 pb-2">
          <input type="checkbox" disabled={!canEdit} checked={draft.rent_reminders} onChange={(e) => setDraft((d) => ({ ...d, rent_reminders: e.target.checked }))} />
          {t('تذكيرات الإيجار للمستأجر', 'Rent reminders to the tenant')}
        </label>
        {canEdit && <button type="button" onClick={save} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white">{t('حفظ', 'Save')}</button>}
        {canEdit && <button type="button" onClick={startRenewal} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('بدء التجديد', 'Start renewal')}</button>}
        <Link href="/renewals" className="text-xs text-primary-600 hover:underline pb-1.5">{t('التجديدات', 'Renewals')} →</Link>
      </div>
      <p className="text-[11px] text-gray-500">
        {t('يتلقى الوكيل والمسؤولون تنبيهات قبل انتهاء العقد وتسجيل إيجاري، وإشعارات الشيكات المرتجعة.',
          'The agent and the admins get alerts before the lease and its Ejari registration expire, and bounced-cheque notices.')}
      </p>
      {msg && <p className="text-xs text-green-700">{msg}</p>}
      {error && <p role="alert" className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}

      {reminders.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-gray-700 mb-2">{t('تذكيرات الإيجار المرسلة', 'Rent reminders')}</h3>
          <ul className="text-xs text-gray-600 space-y-1">
            {reminders.slice(0, 10).map((r) => (
              <li key={r.id}>
                {formatDateTime(r.sent_at ?? r.reminder_date)} · {r.delivery_method} · {r.reminder_type === 'overdue' ? t('متأخر', 'overdue') : t('قادم', 'upcoming')} ·{' '}
                <span className={r.delivery_status === 'sent' ? 'text-green-700' : r.delivery_status === 'failed' ? 'text-red-600' : 'text-gray-500'}>
                  {r.delivery_status}{r.delivery_error ? ` (${r.delivery_error})` : ''}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
