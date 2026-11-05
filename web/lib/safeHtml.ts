/**
 * Escapes characters with special meaning in HTML to prevent XSS when
 * interpolating untrusted text into raw HTML strings (e.g. Leaflet popups/tooltips).
 */
export function escapeHtml(value: string | null | undefined): string {
  if (value == null) return ''
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
