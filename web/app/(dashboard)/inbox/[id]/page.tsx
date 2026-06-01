'use client'
import { useEffect, useState, useRef, useCallback } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { AgentAssist } from '@/components/agent/AgentAssist'
import TemplatePicker from '@/components/communication/TemplatePicker'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { createNotificationSocket } from '@/lib/ws'
import type { WhatsAppThread, WhatsAppMessage, WhatsAppOutbound, WSEvent } from '@/types'
import clsx from 'clsx'

const outboundStatusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  sent: 'bg-blue-100 text-blue-700',
  delivered: 'bg-green-100 text-green-700',
  read: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
}

const MEDIA_TYPES = ['image', 'video', 'document', 'audio']

export default function ThreadPage() {
  const { id } = useParams<{ id: string }>()
  const { lang, t } = useLang()
  const router = useRouter()
  const { user } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'

  const [thread, setThread] = useState<WhatsAppThread | null>(null)
  const [messages, setMessages] = useState<WhatsAppMessage[]>([])
  const [outbound, setOutbound] = useState<WhatsAppOutbound[]>([])
  const [summary, setSummary] = useState('')
  const [summarizing, setSummarizing] = useState(false)
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [drafting, setDrafting] = useState(false)

  const [messageText, setMessageText] = useState('')
  const [showTemplates, setShowTemplates] = useState(false)
  const [templateName, setTemplateName] = useState('')
  const [templateParams, setTemplateParams] = useState('')
  const [showQuickReplies, setShowQuickReplies] = useState(false)
  const [showMedia, setShowMedia] = useState(false)
  const [mediaUrl, setMediaUrl] = useState('')
  const [mediaType, setMediaType] = useState('image')
  const [mediaCaption, setMediaCaption] = useState('')

  const [buyerProfile, setBuyerProfile] = useState<Record<string, unknown> | null>(null)
  const [profileLoading, setProfileLoading] = useState(false)
  const [leadCreating, setLeadCreating] = useState(false)
  const [leadCreated, setLeadCreated] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  useEffect(scrollToBottom, [messages, outbound])

  const loadThread = useCallback(async () => {
    if (!id) return
    const [t, msgs, ob] = await Promise.all([
      api.threads.get(id) as Promise<WhatsAppThread>,
      api.threads.messages(id) as Promise<WhatsAppMessage[]>,
      api.whatsapp.getOutboundMessages(id).catch(() => []) as Promise<WhatsAppOutbound[]>,
    ])
    setThread(t)
    setMessages(Array.isArray(msgs) ? msgs : [])
    setOutbound(Array.isArray(ob) ? ob : [])
    setSummary(t.ai_summary || '')
  }, [id])

  useEffect(() => {
    if (!id) return
    Promise.all([
      api.threads.get(id) as Promise<WhatsAppThread>,
      api.threads.messages(id) as Promise<WhatsAppMessage[]>,
      api.whatsapp.getOutboundMessages(id).catch(() => []) as Promise<WhatsAppOutbound[]>,
    ]).then(([t, msgs, ob]) => {
      setThread(t)
      setMessages(Array.isArray(msgs) ? msgs : [])
      setOutbound(Array.isArray(ob) ? ob : [])
      setSummary(t.ai_summary || '')
    }).finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    if (!user?.id || !id) return
    const close = createNotificationSocket(user.id, (event: WSEvent) => {
      if (event.type === 'whatsapp.message') {
        const p = event.payload as Record<string, unknown>
        if (p.thread_id === id) {
          loadThread()
        }
      }
    })
    return () => close()
  }, [user?.id, id, loadThread])

  const handleSummarize = async () => {
    setSummarizing(true)
    try {
      const res = await api.ai.summarize(id) as { summary: string }
      setSummary(res.summary)
    } catch {
      setSummary(t('حدث خطأ أثناء التلخيص', 'Summarization failed'))
    } finally { setSummarizing(false) }
  }

  const handleDraftReply = async () => {
    setDrafting(true)
    try {
      const res = await api.ai.draftReply(id) as { reply: string; summary: string }
      setMessageText(res.reply || '')
      if (res.summary) setSummary(res.summary)
    } catch {} finally { setDrafting(false) }
  }

  const handleExtractProfile = async () => {
    setProfileLoading(true)
    try {
      const res = await api.ai.extractBuyerProfile(id) as Record<string, unknown>
      setBuyerProfile(res)
    } catch {
      setBuyerProfile({ error: t('فشل استخراج الملف', 'Failed to extract profile') })
    } finally { setProfileLoading(false) }
  }

  const handleAutoCreateLead = async () => {
    setLeadCreating(true)
    try {
      await api.messages.autoCreateLead({ thread_id: id })
      setLeadCreated(true)
    } catch {
      alert(t('فشل إنشاء الفرصة', 'Failed to create lead'))
    } finally { setLeadCreating(false) }
  }

  const handleClose = async () => {
    await api.threads.close(id).catch(() => {})
    router.push('/inbox')
  }

  const handleReopen = async () => {
    await api.threads.reopen(id)
    loadThread()
  }

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!messageText.trim() || sending) return
    setSending(true)
    try {
      await api.whatsapp.sendMessage(id, messageText.trim())
      setMessageText('')
      const [msgs, ob] = await Promise.all([
        api.threads.messages(id),
        api.whatsapp.getOutboundMessages(id).catch(() => []),
      ])
      setMessages(Array.isArray(msgs) ? msgs : [])
      setOutbound(Array.isArray(ob) ? ob : [])
    } catch {} finally { setSending(false) }
  }

  const handleSendMedia = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!mediaUrl.trim() || sending) return
    setSending(true)
    try {
      await api.whatsapp.sendMedia(id, { media_url: mediaUrl.trim(), media_type: mediaType, caption: mediaCaption })
      setMediaUrl(''); setMediaCaption(''); setShowMedia(false)
      const [msgs, ob] = await Promise.all([
        api.threads.messages(id),
        api.whatsapp.getOutboundMessages(id).catch(() => []),
      ])
      setMessages(Array.isArray(msgs) ? msgs : [])
      setOutbound(Array.isArray(ob) ? ob : [])
    } catch {} finally { setSending(false) }
  }

  const handleSendTemplate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!templateName.trim() || sending) return
    setSending(true)
    try {
      const params = templateParams ? templateParams.split(',').map(s => s.trim()) : []
      await api.whatsapp.sendTemplate(id, templateName.trim(), params)
      setTemplateName(''); setTemplateParams(''); setShowTemplates(false)
      const [msgs, ob] = await Promise.all([
        api.threads.messages(id),
        api.whatsapp.getOutboundMessages(id).catch(() => []),
      ])
      setMessages(Array.isArray(msgs) ? msgs : [])
      setOutbound(Array.isArray(ob) ? ob : [])
    } catch {} finally { setSending(false) }
  }

  const handleActionClick = (action: string, message?: string) => {
    if (message) setMessageText(message)
  }

  const isMediaPlaceholder = (body: string, direction: string) =>
    direction === 'inbound' && /^\[.*?\]$/.test(body.trim())

  const allMessages = [
    ...messages.map(m => ({
      type: 'inbound' as const, id: m.id, time: new Date(m.sent_at).getTime(),
      body: m.body, mediaUrl: m.media_url, direction: m.direction,
    })),
    ...outbound.filter(o => o.status !== 'pending').map(o => ({
      type: 'outbound' as const, id: `out-${o.id}`,
      time: o.sent_at ? new Date(o.sent_at).getTime() : Date.now(),
      body: o.message_body, mediaUrl: o.media_url,
      direction: 'outbound' as const, status: o.status,
      errorMessage: o.error_message,
    })),
  ].sort((a, b) => a.time - b.time)

  const conversationText = allMessages.map(m => m.body).join('\n')

  const renderMedia = (url: string) => {
    const ext = url.split('.').pop()?.toLowerCase()
    if (!ext) return <a href={url} target="_blank" rel="noopener noreferrer" className="text-xs text-brand-600 hover:underline block mt-1">{t('عرض المرفق', 'View attachment')}</a>
    if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext))
      return <img src={url} alt="" className="max-w-full rounded-lg mt-1 max-h-48 object-cover" loading="lazy" />
    if (['mp4', 'webm', 'mov'].includes(ext))
      return <video src={url} controls className="max-w-full rounded-lg mt-1 max-h-48" />
    return <a href={url} target="_blank" rel="noopener noreferrer" className="text-xs text-brand-600 hover:underline block mt-1">{t('تحميل المرفق', 'Download attachment')}</a>
  }

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={thread?.contact?.full_name ?? t('المحادثة', 'Thread')} />

      {/* Thread meta bar */}
      <div className="bg-white border-b border-gray-100 px-6 py-3 flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-full bg-brand-100 text-brand-700 font-semibold text-sm flex items-center justify-center">
            {thread?.contact?.full_name?.[0]?.toUpperCase() ?? '?'}
          </div>
          <div>
            <p className="font-medium text-sm text-gray-900">{thread?.contact?.full_name}</p>
            <p className="text-xs text-gray-400">{thread?.contact?.phone_wa}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isAgent && (
            <>
              <button onClick={handleDraftReply} disabled={drafting}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-indigo-50 text-indigo-700 rounded-lg hover:bg-indigo-100 disabled:opacity-50">
                {drafting ? '⏳' : '🤖'} {t('مسودة رد', 'Draft Reply')}
              </button>
              <button onClick={handleExtractProfile} disabled={profileLoading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-purple-50 text-purple-700 rounded-lg hover:bg-purple-100 disabled:opacity-50">
                {profileLoading ? '⏳' : '👤'} {t('ملف المشتري', 'Buyer Profile')}
              </button>
              <button onClick={handleAutoCreateLead} disabled={leadCreating || leadCreated}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-emerald-50 text-emerald-700 rounded-lg hover:bg-emerald-100 disabled:opacity-50">
                {leadCreating ? '⏳' : leadCreated ? '✅' : '📋'} {t('إنشاء فرصة', 'Create Lead')}
              </button>
            </>
          )}
          <button onClick={handleSummarize} disabled={summarizing || messages.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-indigo-50 text-indigo-700 rounded-lg hover:bg-indigo-100 disabled:opacity-50">
            {summarizing ? '⏳' : '✨'}
            {summarizing ? t('جاري التلخيص...', 'Summarizing...') : t('تلخيص', 'AI Summarize')}
          </button>
          {thread?.thread_status === 'closed' ? (
            <button onClick={handleReopen}
              className="px-3 py-1.5 text-xs font-medium bg-emerald-100 text-emerald-600 rounded-lg hover:bg-emerald-200">
              {t('إعادة فتح', 'Reopen')}
            </button>
          ) : (
            <button onClick={handleClose}
              className="px-3 py-1.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-lg hover:bg-gray-200">
              {t('إغلاق', 'Close')}
            </button>
          )}
        </div>
      </div>

      {/* AI Summary */}
      {summary && (
        <div className="mx-6 mt-4 p-4 bg-indigo-50 border border-indigo-100 rounded-xl text-sm text-indigo-800">
          <p className="font-semibold text-xs text-indigo-500 mb-1">{t('ملخص', 'AI Summary')}</p>
          <p>{summary}</p>
        </div>
      )}

      {/* Buyer Profile */}
      {buyerProfile && (
        <div className="mx-6 mt-3 p-4 bg-purple-50 border border-purple-100 rounded-xl text-sm text-purple-800">
          <p className="font-semibold text-xs text-purple-500 mb-1">{t('ملف المشتري', 'Buyer Profile')}</p>
          {buyerProfile.error ? (
            <p className="text-red-600">{buyerProfile.error as string}</p>
          ) : (
            <pre className="text-xs whitespace-pre-wrap font-sans">{JSON.stringify(buyerProfile, null, 2)}</pre>
          )}
          <button onClick={() => setBuyerProfile(null)}
            className="mt-2 text-xs text-purple-500 hover:underline">{t('إخفاء', 'Hide')}</button>
        </div>
      )}

      {/* Agent Assist */}
      {isAgent && id && (
        <div className="mx-6 mt-3">
          <AgentAssist
            leadId=""
            threadId={id}
            contactName={thread?.contact?.full_name ?? ''}
            conversation={conversationText}
            onActionClick={handleActionClick}
          />
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-3">
        {allMessages.length === 0 ? (
          <p className="text-center text-sm text-gray-400 mt-10">{t('لا توجد رسائل', 'No messages yet')}</p>
        ) : (
          allMessages.map((msg) => {
            const isInbound = msg.direction === 'inbound'
            const isPureMedia = isMediaPlaceholder(msg.body, msg.direction)
            const displayBody = isPureMedia ? null : msg.body
            return (
              <div key={msg.id} className={clsx('flex', isInbound ? 'justify-start' : 'justify-end')}>
                <div className={clsx('max-w-xs md:max-w-md px-4 py-2.5 rounded-2xl text-sm', isInbound ? 'bg-white border border-gray-100 text-gray-800 rounded-tl-sm' : 'bg-brand-600 text-white rounded-tr-sm')}>
                  {displayBody && (
                    <p className="leading-relaxed whitespace-pre-wrap">{displayBody}</p>
                  )}
                  {'status' in msg && msg.status && (
                    <p className={clsx('text-[10px] mt-1 inline-block px-1.5 py-0.5 rounded', outboundStatusColor[msg.status] || 'text-gray-400')}>
                      {msg.status}
                    </p>
                  )}
                  {'errorMessage' in msg && msg.errorMessage && msg.status === 'failed' && (
                    <p className="text-[10px] mt-1 text-red-400">{msg.errorMessage}</p>
                  )}
                  {msg.mediaUrl && renderMedia(msg.mediaUrl)}
                </div>
              </div>
            )
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Compose bar */}
      {isAgent && thread?.thread_status !== 'closed' && (
        <div className="border-t border-gray-100 bg-white px-4 py-3">
          <form onSubmit={handleSendMessage} className="flex items-end gap-2">
            <div className="flex-1">
              <textarea value={messageText} onChange={e => setMessageText(e.target.value)}
                placeholder={t('اكتب رسالة...', 'Type a message...')}
                rows={2}
                className="w-full text-sm border border-gray-200 rounded-xl px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none"
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSendMessage(e) } }} />
            </div>
            <button type="button" onClick={() => { setShowQuickReplies(!showQuickReplies); setShowTemplates(false); setShowMedia(false) }}
              className="px-3 py-2.5 text-xs font-medium bg-brand-50 text-brand-700 rounded-lg hover:bg-brand-100 shrink-0">
              {t('ردود سريعة', 'Quick Reply')}
            </button>
            <button type="button" onClick={() => { setShowTemplates(!showTemplates); setShowQuickReplies(false); setShowMedia(false) }}
              className="px-3 py-2.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-lg hover:bg-gray-200 shrink-0">
              {t('قالب ميتا', 'Meta Template')}
            </button>
            <button type="button" onClick={() => { setShowMedia(!showMedia); setShowQuickReplies(false); setShowTemplates(false) }}
              className="px-3 py-2.5 text-xs font-medium bg-gray-100 text-gray-600 rounded-lg hover:bg-gray-200 shrink-0">
              {t('وسائط', 'Media')}
            </button>
            <button type="submit" disabled={sending || !messageText.trim()}
              className="px-5 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 shrink-0">
              {sending ? '...' : t('إرسال', 'Send')}
            </button>
          </form>

          {/* Media panel */}
          {showMedia && (
            <form onSubmit={handleSendMedia} className="mt-3 p-4 bg-gray-50 rounded-xl border border-gray-200">
              <p className="text-xs font-semibold text-gray-600 mb-3">{t('إرسال وسائط', 'Send Media')}</p>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="block text-[10px] text-gray-500 mb-1">{t('رابط الملف', 'Media URL')}</label>
                  <input value={mediaUrl} onChange={e => setMediaUrl(e.target.value)}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500"
                    placeholder="https://example.com/image.jpg" />
                </div>
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">{t('النوع', 'Type')}</label>
                  <select value={mediaType} onChange={e => setMediaType(e.target.value)}
                    className="text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500 bg-white">
                    {MEDIA_TYPES.map(mt => <option key={mt} value={mt}>{mt}</option>)}
                  </select>
                </div>
              </div>
              <div className="mt-2">
                <label className="block text-[10px] text-gray-500 mb-1">{t('التعليق (اختياري)', 'Caption (optional)')}</label>
                <input value={mediaCaption} onChange={e => setMediaCaption(e.target.value)}
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500"
                  placeholder={t('أضف تعليقاً...', 'Add a caption...')} />
              </div>
              <div className="flex gap-2 mt-3">
                <button type="submit" disabled={sending || !mediaUrl.trim()}
                  className="px-4 py-2 bg-green-600 text-white text-xs font-medium rounded-lg hover:bg-green-700 disabled:opacity-50">
                  {t('إرسال الوسائط', 'Send Media')}
                </button>
                <button type="button" onClick={() => setShowMedia(false)}
                  className="px-4 py-2 border border-gray-200 text-xs font-medium rounded-lg text-gray-600 hover:bg-gray-50">
                  {t('إلغاء', 'Cancel')}
                </button>
              </div>
            </form>
          )}

          {/* Quick Reply Templates */}
          {showQuickReplies && (
            <TemplatePicker
              onSelect={(body) => {
                setMessageText(body)
                setShowQuickReplies(false)
              }}
              onClose={() => setShowQuickReplies(false)}
            />
          )}

          {/* Meta Template panel */}
          {showTemplates && (
            <form onSubmit={handleSendTemplate} className="mt-3 p-4 bg-gray-50 rounded-xl border border-gray-200">
              <p className="text-xs font-semibold text-gray-600 mb-3">{t('إرسال قالب', 'Send Template')}</p>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="block text-[10px] text-gray-500 mb-1">{t('اسم القالب', 'Template Name')}</label>
                  <input value={templateName} onChange={e => setTemplateName(e.target.value)}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500"
                    placeholder={t('مثلاً: hello_world', 'e.g. hello_world')} />
                </div>
                <div className="flex-1">
                  <label className="block text-[10px] text-gray-500 mb-1">{t('المتغيرات (مفصولة بفواصل)', 'Parameters (comma-sep)')}</label>
                  <input value={templateParams} onChange={e => setTemplateParams(e.target.value)}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500"
                    placeholder={t('اسم العميل, تاريخ', 'Name, Date')} />
                </div>
              </div>
              <div className="flex gap-2 mt-3">
                <button type="submit" disabled={sending || !templateName.trim()}
                  className="px-4 py-2 bg-purple-600 text-white text-xs font-medium rounded-lg hover:bg-purple-700 disabled:opacity-50">
                  {t('إرسال القالب', 'Send Template')}
                </button>
                <button type="button" onClick={() => setShowTemplates(false)}
                  className="px-4 py-2 border border-gray-200 text-xs font-medium rounded-lg text-gray-600 hover:bg-gray-50">
                  {t('إلغاء', 'Cancel')}
                </button>
              </div>
            </form>
          )}
        </div>
      )}
    </div>
  )
}
