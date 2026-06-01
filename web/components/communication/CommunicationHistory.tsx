'use client'
import { useLang } from '@/context/LangContext'
import type { CommunicationHistory, CommunicationType } from '@/types'
import clsx from 'clsx'

const typeIcons: Record<CommunicationType, string> = {
  whatsapp_inbound: '📱',
  whatsapp_outbound: '💬',
  email_sent: '📧',
  email_received: '✉️',
  call: '☎️',
}

const typeLabels: Record<CommunicationType, { ar: string; en: string }> = {
  whatsapp_inbound: { ar: 'رسالة واتساب واردة', en: 'WhatsApp Inbound' },
  whatsapp_outbound: { ar: 'رسالة واتساب صادرة', en: 'WhatsApp Outbound' },
  email_sent: { ar: 'بريد إلكتروني مرسل', en: 'Email Sent' },
  email_received: { ar: 'بريد إلكتروني وارد', en: 'Email Received' },
  call: { ar: 'مكالمة', en: 'Call' },
}

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  sent: 'bg-blue-100 text-blue-700',
  delivered: 'bg-green-100 text-green-700',
  read: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  bounced: 'bg-red-100 text-red-700',
}

interface Props {
  communications: CommunicationHistory[]
  loading?: boolean
}

export function CommunicationHistoryComponent({ communications, loading }: Props) {
  const { t, lang } = useLang()
  const isAR = lang === 'ar'

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-500" />
      </div>
    )
  }

  if (!communications || communications.length === 0) {
    return (
      <div className="py-6 text-center text-gray-400">
        {t('لا توجد رسائل حتى الآن', 'No communications yet')}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {communications.map((comm, idx) => (
        <div key={comm.id} className="flex gap-3">
          {/* Timeline connector */}
          <div className="flex flex-col items-center">
            <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center text-lg">
              {typeIcons[comm.communication_type]}
            </div>
            {idx < communications.length - 1 && (
              <div className="w-0.5 h-12 bg-gray-200 mt-1" />
            )}
          </div>

          {/* Content */}
          <div className="flex-1 pb-2">
            {/* Header: type, timestamp, status */}
            <div className="flex flex-col gap-1 mb-1">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs font-medium text-gray-700">
                  {isAR ? typeLabels[comm.communication_type].ar : typeLabels[comm.communication_type].en}
                </span>
                <span className={clsx(
                  'text-[10px] font-medium px-1.5 py-0.5 rounded',
                  statusColors[comm.status] ?? 'bg-gray-100 text-gray-600'
                )}>
                  {comm.status}
                </span>
              </div>
              <p className="text-[11px] text-gray-400">
                {new Date(comm.created_at).toLocaleString(isAR ? 'ar-SA' : 'en-US')}
              </p>
            </div>

            {/* Message body */}
            {comm.body && (
              <p className="text-sm text-gray-700 bg-gray-50 rounded-lg p-2.5 mt-1 break-words">
                {comm.body}
              </p>
            )}

            {/* From/To info */}
            <div className="flex gap-2 text-[10px] text-gray-400 mt-1.5 flex-wrap">
              {comm.from_identifier && (
                <span>{t('من', 'From')}: {comm.from_identifier}</span>
              )}
              {comm.to_identifier && (
                <span>{t('إلى', 'To')}: {comm.to_identifier}</span>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
