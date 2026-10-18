'use client'
import { useState, useEffect } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface Props {
  open: boolean
  onClose: () => void
  toEmail?: string
  toName?: string
  relatedTo?: string
  relatedId?: string
  onSent?: () => void
}

export default function EmailComposeModal({ open, onClose, toEmail, toName, relatedTo, relatedId, onSent }: Props) {
  const { t } = useLang()
  const [to, setTo] = useState(toEmail || '')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  useEffect(() => {
    if (open) {
      setTo(toEmail || '')
      setSubject('')
      setBody('')
      setError('')
      setSuccess('')
    }
  }, [open, toEmail])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  if (!open) return null

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!to.trim() || !subject.trim()) return
    setSending(true); setError(''); setSuccess('')
    try {
      const payload: Record<string, unknown> = {
        to_email: to.trim(),
        subject: subject.trim(),
        body: body.trim(),
      }
      if (relatedTo && relatedId) {
        payload.related_to = relatedTo
        payload.related_id = parseInt(relatedId, 10) || relatedId
      }
      await api.email.send(payload)
      setSuccess(t('تم إرسال البريد', 'Email sent'))
      setTimeout(() => { onClose(); onSent?.() }, 1500)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('فشل الإرسال', 'Failed to send'))
    } finally {
      setSending(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="email-modal-title"
    >
      <div
        className="bg-white rounded-2xl p-6 w-full max-w-lg shadow-2xl relative animate-scale-in"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 id="email-modal-title" className="text-lg font-semibold text-surface-900">
            {toName ? t(`إرسال بريد إلى ${toName}`, `Email to ${toName}`) : t('بريد إلكتروني جديد', 'New Email')}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('إغلاق', 'Close')}
            className="inline-flex items-center justify-center w-8 h-8 rounded-lg text-surface-400 hover:text-surface-900 hover:bg-surface-100 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSend} className="space-y-4">
          <div>
            <label htmlFor="email-to" className="text-xs font-medium text-surface-700 mb-1 block">
              {t('إلى', 'To')} <span className="text-red-500">*</span>
            </label>
            <input
              id="email-to"
              type="email"
              value={to}
              onChange={e => setTo(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-surface-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 transition-colors"
              required
            />
          </div>

          <div>
            <label htmlFor="email-subject" className="text-xs font-medium text-surface-700 mb-1 block">
              {t('الموضوع', 'Subject')} <span className="text-red-500">*</span>
            </label>
            <input
              id="email-subject"
              type="text"
              value={subject}
              onChange={e => setSubject(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-surface-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 transition-colors"
              required
            />
          </div>

          <div>
            <label htmlFor="email-body" className="text-xs font-medium text-surface-700 mb-1 block">
              {t('النص', 'Body')}
            </label>
            <textarea
              id="email-body"
              value={body}
              onChange={e => setBody(e.target.value)}
              rows={6}
              className="w-full px-3 py-2 text-sm border border-surface-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 resize-none transition-colors"
            />
          </div>

          {error && <p className="text-xs text-red-600 bg-red-50 border border-red-100 p-2.5 rounded-lg">{error}</p>}
          {success && <p className="text-xs text-green-700 bg-green-50 border border-green-100 p-2.5 rounded-lg">{success}</p>}

          <div className="flex justify-end gap-3 pt-3 border-t border-surface-100">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-surface-600 hover:text-surface-900 transition-colors rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
            >
              {t('إلغاء', 'Cancel')}
            </button>
            <button
              type="submit"
              disabled={sending || !to.trim() || !subject.trim()}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
            >
              {sending ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  <span>{t('جاري الإرسال...', 'Sending...')}</span>
                </>
              ) : (
                t('إرسال', 'Send')
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
