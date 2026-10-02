/**
 * Returns the URL only if it is safe to put in an href/src: an absolute http(s) URL or a
 * same-site absolute path. Anything else (javascript:, data:, vbscript:, protocol-relative
 * "//host", control characters used to smuggle a scheme) yields undefined so the link is inert.
 *
 * URL fields are user-supplied and React does not block `javascript:` hrefs, so every link built
 * from stored data must go through this.
 */
export function safeUrl(value: string | null | undefined): string | undefined {
  if (!value) return undefined
  const s = value.trim()
  if (!s || /[\u0000-\u001f\u007f]/.test(s)) return undefined
  if (s.startsWith('/') && !s.startsWith('//') && !s.startsWith('/\\')) return s
  try {
    const u = new URL(s)
    if (u.protocol === 'http:' || u.protocol === 'https:') return s
  } catch {
    // not an absolute URL
  }
  return undefined
}
