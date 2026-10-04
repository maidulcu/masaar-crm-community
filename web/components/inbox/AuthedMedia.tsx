'use client'
import { useEffect, useRef, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'

interface Props {
  threadId: string
  messageId: string
  mime?: string
  filename?: string
  size?: number
}

const IMAGE = /^image\/(jpeg|png|webp|gif)$/
const AUDIO = /^audio\//
const VIDEO = /^video\/(mp4|3gpp)$/

function formatSize(bytes?: number) {
  if (!bytes) return ''
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

/**
 * Shows a WhatsApp attachment. The file endpoint needs the Authorization header, which <img src>
 * and <a href> cannot send, so the file is fetched with the app's authenticated client and
 * displayed from a Blob URL. Images load when they scroll into view; audio, video and documents
 * only when asked, to avoid downloading large files nobody opens.
 */
export default function AuthedMedia({ threadId, messageId, mime = '', filename, size }: Props) {
  const { t } = useLang()
  const [url, setUrl] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const holder = useRef<HTMLDivElement>(null)
  const started = useRef(false)

  const kind = IMAGE.test(mime) ? 'image' : AUDIO.test(mime) ? 'audio' : VIDEO.test(mime) ? 'video' : 'file'

  const load = async () => {
    if (started.current) return
    started.current = true
    setLoading(true)
    setError('')
    try {
      const blob = await api.whatsapp.fetchMedia(threadId, messageId)
      setUrl(URL.createObjectURL(blob))
    } catch (e) {
      started.current = false
      setError(e instanceof Error ? e.message : t('تعذر تحميل المرفق', 'Could not load the attachment'))
    } finally {
      setLoading(false)
    }
  }

  // Images load lazily, when they become visible.
  useEffect(() => {
    if (kind !== 'image' || !holder.current) return
    const el = holder.current
    const io = new IntersectionObserver((entries) => {
      if (entries.some((en) => en.isIntersecting)) {
        io.disconnect()
        load()
      }
    })
    io.observe(el)
    return () => io.disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, threadId, messageId])

  useEffect(() => () => { if (url) URL.revokeObjectURL(url) }, [url])

  const download = async () => {
    setLoading(true)
    setError('')
    try {
      await api.whatsapp.downloadMedia(threadId, messageId, filename || 'attachment')
    } catch (e) {
      setError(e instanceof Error ? e.message : t('تعذر تحميل المرفق', 'Could not load the attachment'))
    } finally {
      setLoading(false)
    }
  }

  const button = 'text-xs underline underline-offset-2 hover:no-underline disabled:opacity-50'

  return (
    <div ref={holder} className="mt-1">
      {kind === 'image' && (
        url
          ? <img src={url} alt={filename || ''} className="max-w-full rounded-lg max-h-64 object-contain" />
          : <div className="h-24 w-40 rounded-lg bg-black/5 flex items-center justify-center text-[11px] opacity-70">{loading ? t('جاري التحميل...', 'Loading...') : t('صورة', 'Image')}</div>
      )}
      {kind === 'audio' && (url
        ? <audio src={url} controls className="max-w-full" />
        : <button type="button" onClick={load} disabled={loading} className={button}>{loading ? t('جاري التحميل...', 'Loading...') : t('▶ تشغيل الصوت', '▶ Play audio')} {formatSize(size)}</button>)}
      {kind === 'video' && (url
        ? <video src={url} controls className="max-w-full rounded-lg max-h-64" />
        : <button type="button" onClick={load} disabled={loading} className={button}>{loading ? t('جاري التحميل...', 'Loading...') : t('▶ تشغيل الفيديو', '▶ Play video')} {formatSize(size)}</button>)}
      {kind === 'file' && (
        <button type="button" onClick={download} disabled={loading} className={button}>
          {loading ? t('جاري التحميل...', 'Loading...') : `⬇ ${filename || t('تحميل المرفق', 'Download attachment')} ${formatSize(size)}`}
        </button>
      )}
      {error && <p role="alert" className="text-[10px] mt-1 text-red-500">{error}</p>}
    </div>
  )
}
