import { currentLocale } from '@/store/locale'

// All timestamps from the API are UTC. This module formats them in the company's
// timezone (Settings → Company), which defaults to Asia/Dubai (GST, UTC+4, no DST).

const GST_LOCALE = 'en-AE'

function tz(): string {
  return currentLocale().timezone || 'Asia/Dubai'
}

/** The date in the Umm al-Qura (Hijri) calendar, e.g. "12 Rabi I 1448 AH". */
export function formatHijri(iso: string | Date): string {
  return new Intl.DateTimeFormat('en-SA-u-ca-islamic-umalqura', {
    timeZone: tz(),
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(iso))
}

/** Format a UTC timestamp as a human-readable date in the company timezone, followed
 *  by the Hijri date when the company enabled it (Settings → Company). */
export function formatDate(iso: string | Date): string {
  const d = new Date(iso)
  const out = new Intl.DateTimeFormat(GST_LOCALE, {
    timeZone: tz(),
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(d)
  return currentLocale().show_hijri && !isNaN(d.getTime()) ? `${out} (${formatHijri(d)})` : out
}

/** Format a UTC timestamp as date + time in GST. */
export function formatDateTime(iso: string | Date): string {
  return new Intl.DateTimeFormat(GST_LOCALE, {
    timeZone: tz(),
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}

/** Format a UTC timestamp as a relative time string (e.g. "3 minutes ago"). */
export function formatRelative(iso: string | Date): string {
  const diff = Date.now() - new Date(iso).getTime()
  const rtf = new Intl.RelativeTimeFormat(GST_LOCALE, { numeric: 'auto' })
  if (Math.abs(diff) < 60_000) return rtf.format(-Math.round(diff / 1000), 'second')
  if (Math.abs(diff) < 3_600_000) return rtf.format(-Math.round(diff / 60_000), 'minute')
  if (Math.abs(diff) < 86_400_000) return rtf.format(-Math.round(diff / 3_600_000), 'hour')
  return rtf.format(-Math.round(diff / 86_400_000), 'day')
}
