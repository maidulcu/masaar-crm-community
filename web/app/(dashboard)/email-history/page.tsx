'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { Header } from '@/components/layout/Header'
import clsx from 'clsx'
import type { EmailHistory, EmailStatus } from '@/types'

const STATUS_COLORS: Record<EmailStatus, string> = {
  sent:    'bg-green-100 text-green-700',
  pending: 'bg-yellow-100 text-yellow-700',
  failed:  'bg-red-100 text-red-700',
  bounced: 'bg-orange-100 text-orange-700',
}

const STATUS_LABELS: Record<EmailStatus, { en: string; ar: string }> = {
  sent:    { en: 'Sent',    ar: 'مُرسل' },
  pending: { en: 'Pending', ar: 'قيد الإرسال' },
  failed:  { en: 'Failed',  ar: 'فشل' },
  bounced: { en: 'Bounced', ar: 'مرتد' },
}

function fmtDate(iso: string | null) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

export default function EmailHistoryPage() {
  const { lang, t } = useLang()

  const [emails, setEmails] = useState<EmailHistory[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 50

  const [selected, setSelected] = useState<EmailHistory | null>(null)
  const [statusFilter, setStatusFilter] = useState<EmailStatus | ''>('')

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const res: any = await api.email.list(page, limit)
      setEmails(res?.emails ?? [])
      setTotal(res?.total ?? 0)
    } catch {
      setEmails([])
    } finally {
      setLoading(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / limit))

  const filtered = statusFilter
    ? emails.filter(e => e.status === statusFilter)
    : emails

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('سجل البريد الإلكتروني', 'Email History')} />

      <div className="flex-1 overflow-y-auto p-6">
        {/* Filters */}
        <div className="flex items-center gap-3 mb-4">
          <button
            onClick={() => setStatusFilter('')}
            className={clsx(
              'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
              statusFilter === '' ? 'bg-brand-600 text-white' : 'bg-surface-100 text-surface-600 hover:bg-surface-200'
            )}
          >
            {t('الكل', 'All')} {total > 0 && `(${total})`}
          </button>
          {(['sent', 'failed', 'bounced', 'pending'] as EmailStatus[]).map(s => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={clsx(
                'px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                statusFilter === s ? 'bg-brand-600 text-white' : 'bg-surface-100 text-surface-600 hover:bg-surface-200'
              )}
            >
              {lang === 'ar' ? STATUS_LABELS[s].ar : STATUS_LABELS[s].en}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-40 text-sm text-gray-400">
            {t('جاري التحميل...', 'Loading...')}
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 gap-3 text-center">
            <div className="w-14 h-14 rounded-full bg-gray-100 flex items-center justify-center">
              <svg className="w-7 h-7 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            </div>
            <p className="text-sm text-gray-400">{t('لا توجد رسائل', 'No emails found')}</p>
          </div>
        ) : (
          <>
            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 uppercase tracking-wide">
                    <th className="text-start px-5 py-3 font-medium">{t('المستلم', 'To')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الموضوع', 'Subject')}</th>
                    <th className="text-start px-5 py-3 font-medium hidden md:table-cell">{t('النوع', 'Type')}</th>
                    <th className="text-start px-5 py-3 font-medium">{t('الحالة', 'Status')}</th>
                    <th className="text-start px-5 py-3 font-medium hidden lg:table-cell">{t('التاريخ', 'Date')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50">
                  {filtered.map(email => (
                    <tr
                      key={email.id}
                      className="hover:bg-gray-50 transition-colors cursor-pointer"
                      onClick={() => setSelected(email)}
                    >
                      <td className="px-5 py-3.5 text-gray-900 font-medium max-w-[180px] truncate">
                        {email.to_email}
                      </td>
                      <td className="px-5 py-3.5 text-gray-600 max-w-[240px] truncate">
                        {email.subject || <span className="text-gray-300 italic">{t('بدون موضوع', 'No subject')}</span>}
                      </td>
                      <td className="px-5 py-3.5 hidden md:table-cell">
                        {email.related_to ? (
                          <span className="text-xs px-2 py-0.5 rounded-full bg-surface-100 text-surface-600 capitalize">
                            {email.related_to}
                          </span>
                        ) : (
                          <span className="text-gray-300 text-xs">—</span>
                        )}
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx(
                          'text-xs font-medium px-2 py-0.5 rounded-full',
                          STATUS_COLORS[email.status] ?? 'bg-gray-100 text-gray-500'
                        )}>
                          {lang === 'ar'
                            ? STATUS_LABELS[email.status]?.ar ?? email.status
                            : STATUS_LABELS[email.status]?.en ?? email.status}
                        </span>
                      </td>
                      <td className="px-5 py-3.5 text-gray-400 text-xs hidden lg:table-cell">
                        {fmtDate(email.sent_at ?? email.created_at)}
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

      {/* Detail panel */}
      {selected && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={() => setSelected(null)}>
          <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl max-h-[85vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <div className="flex items-start justify-between mb-4">
              <h2 className="text-base font-semibold text-gray-900 pe-4">{selected.subject || t('بدون موضوع', 'No subject')}</h2>
              <button onClick={() => setSelected(null)} className="shrink-0 text-gray-400 hover:text-gray-600">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="space-y-2 mb-4 text-sm">
              <div className="flex gap-2">
                <span className="text-gray-400 w-16 shrink-0">{t('من', 'From')}</span>
                <span className="text-gray-700">{selected.from_email}</span>
              </div>
              <div className="flex gap-2">
                <span className="text-gray-400 w-16 shrink-0">{t('إلى', 'To')}</span>
                <span className="text-gray-700">{selected.to_email}</span>
              </div>
              <div className="flex gap-2">
                <span className="text-gray-400 w-16 shrink-0">{t('التاريخ', 'Date')}</span>
                <span className="text-gray-700">{fmtDate(selected.sent_at ?? selected.created_at)}</span>
              </div>
              <div className="flex gap-2 items-center">
                <span className="text-gray-400 w-16 shrink-0">{t('الحالة', 'Status')}</span>
                <span className={clsx('text-xs font-medium px-2 py-0.5 rounded-full', STATUS_COLORS[selected.status] ?? 'bg-gray-100 text-gray-500')}>
                  {lang === 'ar' ? STATUS_LABELS[selected.status]?.ar : STATUS_LABELS[selected.status]?.en}
                </span>
              </div>
              {selected.error_message && (
                <div className="flex gap-2">
                  <span className="text-gray-400 w-16 shrink-0">{t('الخطأ', 'Error')}</span>
                  <span className="text-red-600 text-xs">{selected.error_message}</span>
                </div>
              )}
            </div>

            <div className="border border-gray-100 rounded-lg p-4 bg-gray-50">
              <p className="text-xs text-gray-400 mb-2">{t('محتوى الرسالة', 'Message body')}</p>
              <div className="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">
                {selected.body || <span className="text-gray-300 italic">{t('فارغ', 'Empty')}</span>}
              </div>
            </div>

            <div className="flex justify-end mt-4">
              <button onClick={() => setSelected(null)}
                className="px-4 py-2 text-sm text-gray-500 hover:text-gray-700">
                {t('إغلاق', 'Close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
