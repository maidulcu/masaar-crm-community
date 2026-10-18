'use client'
import { useCallback, useEffect, useState } from 'react'
import clsx from 'clsx'
import { api, type Cheque, type ChequeInput, type ChequeStatus } from '@/lib/api'
import { formatDate } from '@/lib/date'
import { formatMoney } from '@/lib/money'
import { useLang } from '@/context/LangContext'

type Translate = (ar: string, en: string) => string

export const CHEQUE_STATUS: Record<ChequeStatus, { ar: string; en: string; cls: string }> = {
  received: { ar: 'مستلم', en: 'In hand', cls: 'bg-blue-50 text-blue-700' },
  on_hold: { ar: 'معلّق', en: 'On hold', cls: 'bg-amber-50 text-amber-700' },
  deposited: { ar: 'مودَع', en: 'Deposited', cls: 'bg-indigo-50 text-indigo-700' },
  cleared: { ar: 'محصّل', en: 'Cleared', cls: 'bg-green-50 text-green-700' },
  bounced: { ar: 'مرتجع', en: 'Bounced', cls: 'bg-red-50 text-red-700' },
  replaced: { ar: 'مستبدل', en: 'Replaced', cls: 'bg-gray-100 text-gray-500' },
  returned: { ar: 'مُعاد للمستأجر', en: 'Returned', cls: 'bg-gray-100 text-gray-500' },
}

// The actions offered for each status, mirroring the server's allowed transitions.
const ACTIONS: Partial<Record<ChequeStatus, ChequeStatus[]>> = {
  received: ['deposited', 'on_hold', 'returned'],
  on_hold: ['received', 'deposited', 'returned'],
  deposited: ['cleared', 'bounced'],
  bounced: ['deposited', 'returned'],
}

const ACTION_LABEL: Record<string, [string, string]> = {
  deposited: ['إيداع', 'Deposit'],
  on_hold: ['تعليق', 'Hold'],
  received: ['إلغاء التعليق', 'Release'],
  returned: ['إعادة للمستأجر', 'Return'],
  cleared: ['تم التحصيل', 'Cleared'],
  bounced: ['مرتجع', 'Bounced'],
}

const today = () => new Date().toISOString().slice(0, 10)
const input = 'px-2 py-1.5 border border-gray-300 rounded-lg text-sm'

export function chequeStatusBadge(status: ChequeStatus, t: Translate) {
  const s = CHEQUE_STATUS[status]
  return <span className={clsx('text-[11px] font-medium px-2 py-0.5 rounded-full whitespace-nowrap', s.cls)}>{t(s.ar, s.en)}</span>
}

function ChequeForm({ initial, onSubmit, onCancel, submitLabel }: {
  initial: ChequeInput; onSubmit: (v: ChequeInput) => Promise<void>; onCancel: () => void; submitLabel: string
}) {
  const { t } = useLang()
  const [v, setV] = useState(initial)
  const [busy, setBusy] = useState(false)
  const set = (k: keyof ChequeInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setV((p) => ({ ...p, [k]: k === 'amount' ? Number(e.target.value) : e.target.value }))
  return (
    <form
      className="flex flex-wrap items-end gap-2 bg-gray-50 border border-gray-200 rounded-lg p-3"
      onSubmit={async (e) => { e.preventDefault(); setBusy(true); try { await onSubmit(v) } finally { setBusy(false) } }}
    >
      <label className="text-xs text-gray-600">{t('رقم الشيك', 'Cheque no.')}<input required className={`${input} block w-28`} value={v.cheque_number} onChange={set('cheque_number')} /></label>
      <label className="text-xs text-gray-600">{t('البنك', 'Bank')}<input className={`${input} block w-28`} value={v.bank_name ?? ''} onChange={set('bank_name')} /></label>
      <label className="text-xs text-gray-600">{t('الساحب', 'Drawer')}<input className={`${input} block w-32`} value={v.drawer_name ?? ''} onChange={set('drawer_name')} /></label>
      <label className="text-xs text-gray-600">{t('المبلغ', 'Amount')}<input required type="number" min={0.001} step="any" className={`${input} block w-28`} value={v.amount || ''} onChange={set('amount')} /></label>
      <label className="text-xs text-gray-600">{t('التاريخ', 'Date')}<input required type="date" className={`${input} block`} value={v.cheque_date} onChange={set('cheque_date')} /></label>
      <button type="submit" disabled={busy} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">{submitLabel}</button>
      <button type="button" onClick={onCancel} className="px-3 py-1.5 text-xs rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
    </form>
  )
}

export default function ChequePanel({ leaseId, frequency, canEdit, onChange }: {
  leaseId: string; frequency: string; canEdit: boolean; onChange?: () => void
}) {
  const { t } = useLang()
  const [cheques, setCheques] = useState<Cheque[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState('')
  const [form, setForm] = useState<{ mode: 'add' } | { mode: 'replace'; cheque: Cheque } | null>(null)
  const [schedule, setSchedule] = useState({
    count: ({ monthly: 12, quarterly: 4, semi_annual: 2, annual: 1 } as Record<string, number>)[frequency] ?? 1,
    first_number: '', bank_name: '', drawer_name: '', first_date: '',
  })

  const load = useCallback(() => {
    api.cheques.forLease(leaseId).then(setCheques).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [leaseId])
  useEffect(() => { load() }, [load])

  const run = async (key: string, fn: () => Promise<unknown>) => {
    setBusy(key); setError('')
    try { await fn(); load(); onChange?.() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') } finally { setBusy('') }
  }

  const act = (c: Cheque, status: ChequeStatus) => {
    let reason: string | undefined
    let penalty: number | undefined
    if (status === 'bounced') {
      const r = window.prompt(t('سبب الارتجاع (مثل: رصيد غير كافٍ)', 'Reason the bank gave (e.g. insufficient funds)'), '')
      if (r === null) return
      reason = r
      const p = window.prompt(t('رسوم الشيك المرتجع (اتركه فارغاً لعدم الإضافة)', 'Returned-cheque charge to add as a payment (leave empty for none)'), '')
      if (p === null) return
      penalty = p.trim() ? Number(p) : undefined
      if (penalty !== undefined && !(penalty > 0)) { setError(t('مبلغ غير صالح', 'Invalid amount')); return }
    } else if (status === 'returned' && !window.confirm(t('إعادة الشيك للمستأجر؟', 'Hand this cheque back to the tenant?'))) {
      return
    }
    run(c.id, () => api.cheques.setStatus(c.id, { status, date: today(), reason, penalty_amount: penalty }))
  }

  const total = cheques.filter((c) => c.status !== 'replaced' && c.status !== 'returned').reduce((n, c) => n + c.amount, 0)
  const currency = cheques[0]?.currency

  return (
    <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
      <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between gap-3">
        <div>
          <h2 className="font-semibold text-gray-900 text-sm">{t('الشيكات المؤجلة', 'Post-dated cheques')}</h2>
          {cheques.length > 0 && <p className="text-xs text-gray-500">{cheques.length} · {formatMoney(total, currency)}</p>}
        </div>
        {canEdit && cheques.length > 0 && !form && (
          <button type="button" onClick={() => setForm({ mode: 'add' })} className="text-xs text-primary-600 hover:underline">{t('إضافة شيك', 'Add cheque')}</button>
        )}
      </div>
      {error && <p role="alert" className="mx-5 mt-3 text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}

      {canEdit && cheques.length === 0 && (
        <form
          className="p-5 flex flex-wrap items-end gap-2"
          onSubmit={(e) => { e.preventDefault(); run('schedule', () => api.cheques.schedule(leaseId, { ...schedule, first_date: schedule.first_date || undefined })) }}
        >
          <p className="w-full text-xs text-gray-500 mb-1">
            {t('أنشئ جدول الشيكات: يُقسَّم إيجار مدة العقد على عدد الشيكات، وتُنشأ دفعة لكل شيك.',
              'Create the cheque schedule: the rent for the lease term is split over the cheques, and a payment is created for each.')}
          </p>
          <label className="text-xs text-gray-600">{t('عدد الشيكات', 'Cheques')}<input type="number" min={1} max={24} className={`${input} block w-20`} value={schedule.count} onChange={(e) => setSchedule((s) => ({ ...s, count: Number(e.target.value) }))} /></label>
          <label className="text-xs text-gray-600">{t('رقم أول شيك', 'First cheque no.')}<input className={`${input} block w-28`} value={schedule.first_number} onChange={(e) => setSchedule((s) => ({ ...s, first_number: e.target.value }))} /></label>
          <label className="text-xs text-gray-600">{t('البنك', 'Bank')}<input className={`${input} block w-28`} value={schedule.bank_name} onChange={(e) => setSchedule((s) => ({ ...s, bank_name: e.target.value }))} /></label>
          <label className="text-xs text-gray-600">{t('الساحب', 'Drawer')}<input className={`${input} block w-32`} value={schedule.drawer_name} onChange={(e) => setSchedule((s) => ({ ...s, drawer_name: e.target.value }))} /></label>
          <label className="text-xs text-gray-600">{t('تاريخ أول شيك', 'First cheque date')}<input type="date" className={`${input} block`} value={schedule.first_date} onChange={(e) => setSchedule((s) => ({ ...s, first_date: e.target.value }))} /></label>
          <button type="submit" disabled={busy === 'schedule'} className="px-3 py-1.5 text-xs rounded-lg bg-primary-600 text-white disabled:opacity-50">{t('إنشاء الجدول', 'Create schedule')}</button>
        </form>
      )}
      {!canEdit && cheques.length === 0 && <p className="p-5 text-xs text-gray-400">{t('لا توجد شيكات', 'No cheques')}</p>}

      {form && (
        <div className="px-5 pt-3">
          <ChequeForm
            initial={form.mode === 'replace'
              ? { cheque_number: '', bank_name: form.cheque.bank_name, drawer_name: form.cheque.drawer_name, amount: form.cheque.amount, cheque_date: today() }
              : { cheque_number: '', amount: 0, cheque_date: today() }}
            submitLabel={form.mode === 'replace' ? t('استبدال', 'Replace') : t('إضافة', 'Add')}
            onCancel={() => setForm(null)}
            onSubmit={async (v) => {
              await run('form', () => form.mode === 'replace' ? api.cheques.replace(form.cheque.id, v) : api.cheques.create(leaseId, v))
              setForm(null)
            }}
          />
        </div>
      )}

      {cheques.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-xs text-gray-400">
              <tr>
                <th scope="col" className="text-start px-4 py-2 font-medium">{t('التاريخ', 'Date')}</th>
                <th scope="col" className="text-start px-4 py-2 font-medium">{t('رقم الشيك', 'Cheque')}</th>
                <th scope="col" className="text-end px-4 py-2 font-medium">{t('المبلغ', 'Amount')}</th>
                <th scope="col" className="text-start px-4 py-2 font-medium">{t('الحالة', 'Status')}</th>
                <th scope="col" className="px-4 py-2" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {cheques.map((c) => (
                <tr key={c.id} className={clsx(c.status === 'bounced' && 'bg-red-50/40')}>
                  <td className="px-4 py-2.5 whitespace-nowrap text-gray-700">{formatDate(c.cheque_date)}</td>
                  <td className="px-4 py-2.5">
                    <span className="font-mono text-gray-900">{c.cheque_number}</span>
                    {c.bank_name && <span className="text-xs text-gray-500"> · {c.bank_name}</span>}
                    {c.bounce_reason && <p className="text-[11px] text-red-600">{c.bounce_reason}</p>}
                  </td>
                  <td className="px-4 py-2.5 text-end tabular-nums whitespace-nowrap">{formatMoney(c.amount, c.currency)}</td>
                  <td className="px-4 py-2.5">{chequeStatusBadge(c.status, t)}</td>
                  <td className="px-4 py-2.5">
                    <div className="flex flex-wrap justify-end gap-1.5">
                      {canEdit && (ACTIONS[c.status] ?? []).map((s) => (
                        <button key={s} type="button" disabled={busy === c.id} onClick={() => act(c, s)}
                          className={clsx('text-[11px] px-2 py-0.5 rounded border disabled:opacity-50',
                            s === 'bounced' ? 'border-red-200 text-red-600' : s === 'cleared' ? 'border-green-200 text-green-700' : 'border-gray-200 text-gray-600')}>
                          {t(...ACTION_LABEL[s])}
                        </button>
                      ))}
                      {canEdit && ['received', 'on_hold', 'bounced'].includes(c.status) && (
                        <button type="button" onClick={() => setForm({ mode: 'replace', cheque: c })} className="text-[11px] px-2 py-0.5 rounded border border-gray-200 text-gray-600">
                          {t('استبدال', 'Replace')}
                        </button>
                      )}
                      {c.has_scan && (
                        <button type="button" onClick={() => run(c.id, () => api.cheques.downloadScan(c.id, `cheque-${c.cheque_number}`))}
                          className="text-[11px] px-2 py-0.5 rounded border border-gray-200 text-gray-600">{t('الصورة', 'Scan')}</button>
                      )}
                      {canEdit && (
                        <label className="text-[11px] px-2 py-0.5 rounded border border-gray-200 text-gray-600 cursor-pointer">
                          {c.has_scan ? t('تغيير الصورة', 'Replace scan') : t('رفع صورة', 'Upload scan')}
                          <input type="file" accept="image/jpeg,image/png,application/pdf" className="sr-only"
                            onChange={(e) => { const f = e.target.files?.[0]; if (f) run(c.id, () => api.cheques.uploadScan(c.id, f)); e.target.value = '' }} />
                        </label>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
