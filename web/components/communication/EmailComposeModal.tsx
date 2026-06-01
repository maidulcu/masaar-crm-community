'use client'
import { useState } from 'react'
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
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-2xl p-6 w-full max-w-lg mx-4 shadow-xl" onClick={e => e.stopPropagation()}>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          {toName ? t('إرسال بريد إلى', `Email to ${toName}`) : t('بريد إلكتروني جديد', 'New Email')}
        </h2>
        <form onSubmit={handleSend} className="space-y-4">
          <div>
            <label className="text-xs text-gray-500 mb-1 block">{t('إلى', 'To')} *</label>
            <input
              type="email"
              value={to}
              onChange={e => setTo(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
              required
            />
          </div>
          <div>
            <label className="text-xs text-gray-500 mb-1 block">{t('الموضوع', 'Subject')} *</label>
            <input
              value={subject}
              onChange={e => setSubject(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500"
              required
            />
          </div>
          <div>
            <label className="text-xs text-gray-500 mb-1 block">{t('النص', 'Body')}</label>
            <textarea
              value={body}
              onChange={e => setBody(e.target.value)}
              rows={6}
              className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
            />
          </div>
          {error && <p className="text-xs text-red-600 bg-red-50 p-2 rounded">{error}</p>}
          {success && <p className="text-xs text-green-600 bg-green-50 p-2 rounded">{success}</p>}
          <div className="flex justify-end gap-3 pt-2 border-t border-gray-100">
            <button type="button" onClick={onClose}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-700">
              {t('إلغاء', 'Cancel')}
            </button>
            <button type="submit" disabled={sending || !to.trim() || !subject.trim()}
              className="px-4 py-2 text-sm font-medium bg-brand-600 text-white rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors">
              {sending ? '...' : t('إرسال', 'Send')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
