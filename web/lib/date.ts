// All timestamps from the API are UTC. This module formats them in Gulf Standard
// Time (GST = UTC+4, no DST) which is the legal timezone for UAE.

const GST_LOCALE = 'en-AE'
const GST_TZ = 'Asia/Dubai' // IANA identifier for GST (UTC+4, no DST)

/** Format a UTC timestamp as a human-readable date in GST. */
export function formatDate(iso: string | Date): string {
  return new Intl.DateTimeFormat(GST_LOCALE, {
    timeZone: GST_TZ,
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(iso))
}

/** Format a UTC timestamp as date + time in GST. */
export function formatDateTime(iso: string | Date): string {
  return new Intl.DateTimeFormat(GST_LOCALE, {
    timeZone: GST_TZ,
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
