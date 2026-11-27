/**
 * Escapes characters with special meaning in HTML strings (&, <, >, ", ')
 * to prevent XSS vulnerabilities when inserting user-supplied content into
 * raw HTML contexts (such as Leaflet map popups and tooltips).
 */
export function escapeHtml(value: unknown): string {
  if (value === null || value === undefined) return ''
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
