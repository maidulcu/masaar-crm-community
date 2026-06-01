'use client'
import { useState, useEffect } from 'react'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import clsx from 'clsx'

interface Action {
  action: string
  reasoning: string
  suggested_message?: string
  properties_to_show?: Array<{ id: string; name: string }>
}

interface Props {
  leadId: string
  threadId: string
  contactName: string
  conversation: string
  onActionClick?: (action: string, message?: string) => void
}

export function AgentAssist({ leadId, threadId, contactName, conversation, onActionClick }: Props) {
  const { t } = useLang()
  const [action, setAction] = useState<Action | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [expanded, setExpanded] = useState(false)

  useEffect(() => {
    fetchSuggestedAction()
  }, [threadId])

  const fetchSuggestedAction = async () => {
    if (!threadId || !conversation) return

    setLoading(true)
    setError('')
    try {
      const response = await api.messages.suggestAction({
        message: conversation.split('\n').slice(-1)[0] || '',
        thread_summary: conversation,
      })
      setAction(response as Action)
    } catch (err) {
      console.error('Failed to fetch suggested action:', err)
      setError(t('فشل في الحصول على الإجراء المقترح', 'Failed to get suggestion'))
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-2 px-3 bg-brand-50 rounded-lg">
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-brand-500" />
        <span className="text-xs text-brand-600">{t('جاري التحليل...', 'Analyzing...')}</span>
      </div>
    )
  }

  if (error || !action) {
    return null
  }

  const actionEmojis: Record<string, string> = {
    send_message: '💬',
    send_proposal: '📄',
    schedule_viewing: '📅',
    send_template: '📧',
    share_properties: '🏠',
    follow_up: '⏰',
    qualify: '✓',
  }

  const emoji = actionEmojis[action.action.toLowerCase()] || '💡'

  return (
    <div className="bg-gradient-to-r from-brand-50 to-blue-50 rounded-lg border border-brand-200 p-3">
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-start gap-2 hover:opacity-80 transition-opacity"
      >
        <span className="text-lg">{emoji}</span>
        <div className="flex-1 text-left">
          <p className="text-xs font-medium text-brand-700">
            {t('الإجراء المقترح', 'Suggested Action')}
          </p>
          <p className="text-sm font-semibold text-gray-800 capitalize">
            {action.action.replace(/_/g, ' ')}
          </p>
        </div>
        <svg
          className={clsx('w-4 h-4 text-gray-400 transition-transform', {
            'rotate-180': expanded,
          })}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M19 14l-7 7m0 0l-7-7m7 7V3"
          />
        </svg>
      </button>

      {/* Expanded Details */}
      {expanded && (
        <div className="mt-3 space-y-3 border-t border-brand-200 pt-3">
          {/* Reasoning */}
          {action.reasoning && (
            <div>
              <p className="text-xs font-medium text-gray-600 mb-1">
                {t('السبب', 'Why')}:
              </p>
              <p className="text-sm text-gray-700 bg-white rounded px-2 py-1.5">
                {action.reasoning}
              </p>
            </div>
          )}

          {/* Suggested Message */}
          {action.suggested_message && (
            <div>
              <p className="text-xs font-medium text-gray-600 mb-1">
                {t('الرسالة المقترحة', 'Suggested Message')}:
              </p>
              <div className="bg-white rounded px-2 py-1.5 border border-gray-200">
                <p className="text-sm text-gray-700 italic">
                  &quot;{action.suggested_message}&quot;
                </p>
              </div>
              <button
                onClick={() => onActionClick?.(action.action, action.suggested_message)}
                className="mt-2 w-full px-3 py-1.5 bg-brand-600 text-white text-xs font-medium rounded hover:bg-brand-700 transition-colors"
              >
                {t('استخدم هذه الرسالة', 'Use This Message')}
              </button>
            </div>
          )}

          {/* Properties to Show */}
          {action.properties_to_show && action.properties_to_show.length > 0 && (
            <div>
              <p className="text-xs font-medium text-gray-600 mb-1">
                {t('العقارات المقترحة', 'Suggested Properties')}:
              </p>
              <div className="space-y-1">
                {action.properties_to_show.map((prop) => (
                  <button
                    key={prop.id}
                    onClick={() => onActionClick?.('share_properties', prop.id)}
                    className="w-full text-left px-2 py-1.5 bg-white border border-gray-200 rounded text-xs text-gray-700 hover:bg-blue-50 transition-colors"
                  >
                    🏠 {prop.name}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Action Button */}
          {!action.suggested_message && (
            <button
              onClick={() => onActionClick?.(action.action)}
              className="w-full px-3 py-2 bg-brand-600 text-white text-xs font-medium rounded hover:bg-brand-700 transition-colors"
            >
              {t('تنفيذ الإجراء', 'Take Action')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
