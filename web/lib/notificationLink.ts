// Where a notification leads, from the JSON in its `data` field (set by the API's jobs and
// handlers). Returns null when there is nowhere specific to go.
export function notificationHref(data: string | null | undefined): string | null {
  if (!data) return null
  let d: Record<string, unknown>
  try { d = JSON.parse(data) } catch { return null }
  const id = (k: string) => (typeof d[k] === 'string' && /^[0-9a-f-]{36}$/i.test(d[k] as string) ? (d[k] as string) : null)
  if (id('lease_id')) return `/leases/${id('lease_id')}`
  if (id('invoice_id')) return `/invoices/${id('invoice_id')}`
  if (d.page === 'cheques') return '/cheques'
  if (id('task_id')) return '/tasks'
  if (id('listing_id')) return `/listings/${id('listing_id')}`
  if (id('offer_id')) return '/offers'
  if (id('viewing_id')) return '/viewings'
  return null
}
