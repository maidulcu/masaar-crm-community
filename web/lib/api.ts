import { getToken, saveSession, getUser, clearSession, saveCompany } from './auth'
import type { AuthUser, Company, LoginResponse, Payment } from '@/types'

const BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

/**
 * Requests to /api/v1/auth/* run in "cookie mode": the refresh token travels only in an HttpOnly
 * cookie (credentials: 'include'), and the custom X-Auth-Mode header makes the request
 * un-forgeable from another site (it forces a CORS preflight the API only grants to our origin).
 */
function authInit(path: string): RequestInit & { headers: Record<string, string> } {
  if (path.startsWith('/api/v1/auth/')) {
    return { credentials: 'include', headers: { 'X-Auth-Mode': 'cookie' } }
  }
  return { headers: {} }
}

/** Tracks an in-flight refresh so concurrent requests don't each kick one off. */
let refreshPromise: Promise<string | null> | null = null

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch(`${BASE}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', 'X-Auth-Mode': 'cookie' },
    })
    if (!res.ok) return null
    const data = await res.json()
    const user = (data.user as AuthUser | undefined) ?? getUser()
    if (!user) return null
    saveSession(data.access_token, user)
    // Keeps plan / trial state current across reloads instead of frozen at the last login.
    if (data.company) saveCompany(data.company as Company)
    return data.access_token as string
  } catch {
    return null
  }
}

/**
 * Exchanges the HttpOnly refresh cookie for a fresh access token. Refresh tokens are single-use,
 * so refreshes from several tabs are serialised with the Web Locks API; otherwise two tabs
 * opening at once would race and one would be logged out.
 */
export async function silentRefresh(): Promise<string | null> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    try {
      if (typeof navigator !== 'undefined' && navigator.locks) {
        return await navigator.locks.request('masaar-refresh', doRefresh)
      }
      return await doRefresh()
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

export interface JurisdictionRegion { code: string; name_en: string; name_ar: string }
export interface JurisdictionRules {
  country: string
  name_en: string
  name_ar: string
  currency: string
  currency_decimals: number
  timezone: string
  weekend: number[] // 0 = Sunday … 6 = Saturday
  vat_rates: Record<string, number>
  tax_id_digits: number
  regions: JurisdictionRegion[]
}
export interface CompanyLocale {
  country: string
  region: string
  currency: string
  currency_decimals: number
  timezone: string
  show_hijri?: boolean
  rules: JurisdictionRules
}
export interface EInvoiceParty {
  name?: string
  tin?: string
  trn?: string
  legal_id?: string
  legal_id_type?: string
  legal_authority?: string
  street?: string
  city?: string
  emirate?: string
  country?: string
  email?: string
}
export interface EInvoiceRequest {
  buyer: EInvoiceParty
  credit_reason?: string
  due_date?: string // YYYY-MM-DD
}
export interface EInvoiceRecord {
  invoice_id: string
  provider: string
  status: 'submitted' | 'accepted' | 'rejected' | 'error'
  provider_ref: string
  reason?: string
  buyer: EInvoiceParty
  credit_reason?: string
  due_date?: string
  submitted_at: string
  decided_at?: string
}
export interface EInvoiceStatus {
  provider: string
  einvoice: EInvoiceRecord | null
  credit_reasons: string[]
}
export interface EInvoiceSettings {
  provider: string
  tin: string
  legal_id_type: 'TL' | 'EID' | 'PAS' | 'CD'
  trade_license_authority: string
  city: string
  service_code: string
  exemption_code: string
}
export interface Consent {
  id: number
  contact_id: string
  channel: 'whatsapp' | 'email' | 'sms' | 'phone'
  status: 'granted' | 'withdrawn'
  source: string
  note?: string
  created_at: string
}

export interface Task {
  id: string
  title: string
  description: string
  type: 'call' | 'meeting' | 'follow_up' | 'todo'
  priority: 'low' | 'normal' | 'high'
  status: 'open' | 'done' | 'cancelled'
  due_at: string | null
  assigned_to: string | null
  related_type: 'lead' | 'contact' | 'deal' | 'listing' | 'viewing' | 'offer' | null
  related_id: string | null
  completed_at?: string
  created_at: string
  assignee_name?: string
  related_label?: string
  overdue: boolean
}

export type TaskInput = Partial<Pick<Task, 'title' | 'description' | 'type' | 'priority' | 'due_at' | 'assigned_to' | 'related_type' | 'related_id'>>

export interface ContactProfile {
  secondary_phones: string[]
  nationality: string
  source: string
  preferred_contact_time: string
  address: string
  notes: string
  job_title: string
  organisation_id: string | null
  organisation_name?: string
}

export interface Organisation {
  id: string
  name: string
  trade_license_no: string
  trn: string
  address: string
  phone: string
  email: string
  website: string
  notes: string
  contact_count: number
}

export interface CustomField {
  id: string
  entity_type: 'contact' | 'lead' | 'deal' | 'listing'
  name: string
  label: string
  type: 'text' | 'number' | 'date' | 'select' | 'boolean'
  options: string[]
  required: boolean
  display_order: number
}

export interface DuplicateGroup {
  reason: 'email' | 'name' | 'phone'
  key: string
  contacts: { id: string; full_name: string; phone_wa: string; email: string; created_at: string }[]
}

export interface ReraField { key: string; en: string; ar: string; long?: boolean }
export interface ReraDefinition {
  type: 'A' | 'B' | 'I' | 'F'
  title_en: string
  title_ar: string
  sections: { en: string; ar: string; fields: ReraField[] }[]
  signers: string[]
}
export interface ReraForm {
  document_id: string
  form_type: ReraDefinition['type']
  listing_id?: string
  contact_id?: string
  offer_id?: string
  data: Record<string, string>
  title: string
  signature_status: string
  created_at: string
}

export interface PortalConnection {
  portal: string
  enabled: boolean
  last_lead_at?: string
  created_at: string
}

export interface PortalLinks { portal: string; lead_webhook_url: string; feed_url: string }

export interface LeadSourceRow {
  source: string
  leads: number
  won_leads: number
  deals: number
  won_revenue: number
}

export type ChequeStatus = 'received' | 'on_hold' | 'deposited' | 'cleared' | 'bounced' | 'replaced' | 'returned'

export interface Cheque {
  id: string
  lease_id: string
  payment_id?: string
  cheque_number: string
  bank_name: string
  drawer_name: string
  amount: number
  currency: string
  cheque_date: string
  status: ChequeStatus
  has_scan: boolean
  deposited_at?: string
  cleared_at?: string
  bounced_at?: string
  bounce_reason?: string
  replaced_by?: string
  notes: string
  tenant_name?: string
  property_name?: string
}

export interface ChequeInput {
  cheque_number: string
  bank_name?: string
  drawer_name?: string
  amount: number
  cheque_date: string
  payment_id?: string
  notes?: string
}

export type NoticeKind = 'rent_change' | 'renewal_terms' | 'non_renewal' | 'other'

export interface LeaseNotice {
  id: string
  lease_id: string
  kind: NoticeKind
  current_rent?: number
  new_rent?: number
  effective_date?: string
  body_en: string
  body_ar: string
  sent_at?: string
  sent_via: string
  created_at: string
}

export interface NoticeDraft {
  body_en: string
  body_ar: string
  current_annual_rent: number
  currency: string
  lease_end: string
  notice_deadline: string
  notice_days: number
}

export interface LeaseExtras { agent_id: string | null; agent_name: string; ejari_expiry: string | null; rent_reminders: boolean }

export interface PropertySettings {
  lease_alert_days: number[]
  rent_notice_days: number
  rent_reminders_enabled: boolean
  rent_reminder_days_before: number
  rent_reminder_wa_template: string
  rent_reminder_wa_language: string
}

export interface PaymentReminderRow {
  id: string
  payment_id: string
  reminder_type: string
  reminder_date: string
  sent_at?: string
  delivery_method: string
  delivery_status: string
  delivery_error: string
}

export interface KYCDocument {
  id: string
  contact_id: string
  doc_type: 'emirates_id' | 'passport' | 'residence_visa' | 'trade_licence' | 'other'
  number: string
  issuing_country: string
  issue_date?: string
  expiry_date?: string
  has_file: boolean
  file_name: string
  created_at: string
}

export interface KYCProfile {
  contact_id: string
  full_name: string
  nationality: string
  date_of_birth: string | null
  risk: '' | 'low' | 'medium' | 'high'
  risk_reason: string
  is_pep: boolean
  screening: 'not_screened' | 'clear' | 'possible_match' | 'match'
  screening_source: string
  screened_at: string | null
  notes: string
  reviewed_at: string | null
  documents: KYCDocument[]
  missing: string[]
}

export interface DealKYC {
  deal_id: string
  contact_id: string
  contact_name: string
  missing: string[]
  override_reason?: string
  override_at?: string
  can_win: boolean
  payment_method: string
  cash_amount: number
  aml_status: string
  aml_reasons: string[]
}

export interface AMLCase {
  deal_id: string
  title: string
  stage: string
  amount: number
  currency: string
  payment_method: string
  cash_amount: number
  status: 'flagged' | 'no_report' | 'reported'
  reasons: string[]
  flagged_at?: string
  reviewed_at?: string
  notes: string
  goaml_reference: string
  contact_id: string
  contact_name: string
  contact_risk: string
  contact_is_pep: boolean
  screening: string
  owner_name: string
}

export interface BeneficialOwner { contact_id: string; full_name: string; ownership_pct: number | null; role: string; kyc_missing: string[] }

export interface BroadcastSegment { tag?: string; lead_stage?: string; source?: string; listing_id?: string }

export interface Broadcast {
  id: string
  name: string
  template_name: string
  language: string
  params: string[]
  segment: BroadcastSegment
  status: 'draft' | 'sending' | 'done' | 'cancelled'
  created_at: string
  started_at?: string
  finished_at?: string
  counts: Record<string, number>
}

export interface HoursSpan { open: string; close: string }
export interface HoursSpecial extends HoursSpan { label: string; from: string; to: string }
export interface InboxSettings {
  broadcast_per_minute: number
  auto_reply_enabled: boolean
  auto_reply_en: string
  auto_reply_ar: string
  hours: { week: HoursSpan[]; specials: HoursSpecial[] } | null
}

export interface JobRun {
  id: number
  job_name: string
  started_at: string
  finished_at: string
  status: 'ok' | 'error'
  items: number
  error?: string
}

async function request<T>(path: string, init: RequestInit = {}, _retry = true): Promise<T> {
  const token = getToken()
  const auth = authInit(path)
  const headers: HeadersInit = {
    // A FormData body needs the browser to set its own multipart Content-Type (with boundary).
    ...(init.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...auth.headers,
    ...(init.headers || {}),
  }

  const res = await fetch(`${BASE}${path}`, { ...init, credentials: auth.credentials, headers })

  // A 401 from a credential-submitting auth endpoint (bad password / OTP / link) is an ordinary
  // error to show the user, not an expired session — don't refresh or bounce to /login.
  if (res.status === 401 && path.startsWith('/api/v1/auth/') && !path.endsWith('/logout')) {
    const err = await res.json().catch(() => ({ error: 'Unauthorized' }))
    throw new Error(err.error || 'Unauthorized')
  }

  if (res.status === 401 && _retry) {
    // Try a silent token refresh before giving up
    const newToken = await silentRefresh()
    if (newToken) {
      return request<T>(path, init, false)
    }
    clearSession()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (res.status === 401) {
    clearSession()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (res.status === 403) {
    const err = await res.json().catch(() => ({ error: 'forbidden' }))
    if (err.error === 'demo_account_read_only') {
      // Fire a custom event so the layout can show a nice upgrade modal
      // instead of an error toast.
      window.dispatchEvent(new CustomEvent('demo:blocked'))
      throw new Error('demo_account_read_only')
    }
    throw new Error(err.error || 'Forbidden')
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    // `details` keeps the whole error body (e.g. the `linked` counts on a refused contact delete).
    throw Object.assign(new Error(err.error || 'Request failed'), { status: res.status, details: err })
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export async function requestBlob(path: string, init: RequestInit = {}): Promise<Blob> {
  const token = getToken()
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init.headers || {}),
  }
  const res = await fetch(`${BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    const newToken = await silentRefresh()
    if (newToken) return requestBlob(path, init)
    clearSession()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || 'Request failed')
  }
  return res.blob()
}

function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

/** Opens the tab synchronously (so popup blockers allow it), then points it at the fetched Blob. */
async function openBlobInTab(load: () => Promise<Blob>) {
  const win = window.open('', '_blank')
  try {
    const blob = await load()
    const url = URL.createObjectURL(blob)
    if (win) win.location.href = url
    else saveBlob(blob, 'download')
    setTimeout(() => URL.revokeObjectURL(url), 60_000)
  } catch (e) {
    win?.close()
    throw e
  }
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data?: unknown) => request<T>(path, { method: 'POST', body: data ? JSON.stringify(data) : undefined }),
  put: <T>(path: string, data?: unknown) => request<T>(path, { method: 'PUT', body: data ? JSON.stringify(data) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  
  auth: {
    login: (email: string, password: string) =>
      request('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    requestMagicLink: (email: string, lang_pref?: 'ar' | 'en') =>
      request('/api/v1/auth/magic-link/request', {
        method: 'POST',
        body: JSON.stringify({ email, lang_pref }),
      }),
    verifyMagicLink: (token: string) =>
      request('/api/v1/auth/magic-link/verify', {
        method: 'POST',
        body: JSON.stringify({ token }),
      }),
    logout: () => request('/api/v1/auth/logout', { method: 'DELETE' }),
    requestSMSOTP: (phone: string, lang?: string) =>
      request('/api/v1/auth/sms/request', {
        method: 'POST',
        body: JSON.stringify({ phone, lang }),
      }),
    verifySMSOTP: (phone: string, otp: string) =>
      request('/api/v1/auth/sms/verify', {
        method: 'POST',
        body: JSON.stringify({ phone, otp }),
      }),
    forgotPassword: (email: string) =>
      request('/api/v1/auth/forgot-password', {
        method: 'POST',
        body: JSON.stringify({ email }),
      }),
    resetPassword: (token: string, password: string) =>
      request('/api/v1/auth/reset-password', {
        method: 'POST',
        // The API field is new_password; sending `password` made every reset/invite fail validation.
        body: JSON.stringify({ token, new_password: password }),
      }),
    verifyTOTP: (pending_token: string, code: string) =>
      request<LoginResponse>('/api/v1/auth/verify-totp', { method: 'POST', body: JSON.stringify({ pending_token, code }) }),
    enrollTOTP: (pending_token: string) =>
      request<{ secret: string; otpauth_url: string }>('/api/v1/auth/totp/enroll', { method: 'POST', body: JSON.stringify({ pending_token }) }),
    confirmEnrollTOTP: (pending_token: string, code: string) =>
      request<LoginResponse>('/api/v1/auth/totp/enroll/confirm', { method: 'POST', body: JSON.stringify({ pending_token, code }) }),
    register: (data: { name: string; email: string; password: string; company_name: string; subdomain: string; turnstile_token?: string }) =>
      request('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  },

  // ─── Contacts ───────────────────────────────────────────────────────────────

  organisations: {
    list: (search = '') => request<{ data: Organisation[]; total: number }>(`/api/v1/organisations?search=${encodeURIComponent(search)}&limit=200`),
    get: (id: string) => request<{ organisation: Organisation; contacts: { id: string; full_name: string; phone_wa: string; email: string }[] }>(`/api/v1/organisations/${id}`),
    create: (data: Partial<Organisation>) => request<Organisation>('/api/v1/organisations', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Organisation>) => request<Organisation>(`/api/v1/organisations/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) => request(`/api/v1/organisations/${id}`, { method: 'DELETE' }),
  },

  customFields: {
    list: (entityType = '') => request<CustomField[]>(`/api/v1/custom-fields?entity_type=${entityType}`),
    create: (data: Partial<CustomField>) => request<CustomField>('/api/v1/custom-fields', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<CustomField>) => request<CustomField>(`/api/v1/custom-fields/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) => request(`/api/v1/custom-fields/${id}`, { method: 'DELETE' }),
    values: (entityType: string, id: string) => request<Record<string, string>>(`/api/v1/custom-fields/values/${entityType}/${id}`),
    setValues: (entityType: string, id: string, values: Record<string, string>) =>
      request(`/api/v1/custom-fields/values/${entityType}/${id}`, { method: 'PUT', body: JSON.stringify(values) }),
  },

  duplicates: {
    list: () => request<DuplicateGroup[]>('/api/v1/contacts/duplicates'),
    merge: (keepId: string, mergeIds: string[]) =>
      request('/api/v1/contacts/merge', { method: 'POST', body: JSON.stringify({ keep_id: keepId, merge_ids: mergeIds }) }),
  },

  rera: {
    definitions: () => request<Record<string, ReraDefinition>>('/api/v1/rera-forms/definitions'),
    list: (params: { listing_id?: string; contact_id?: string }) => {
      const q = new URLSearchParams()
      if (params.listing_id) q.set('listing_id', params.listing_id)
      if (params.contact_id) q.set('contact_id', params.contact_id)
      return request<ReraForm[]>(`/api/v1/rera-forms?${q}`)
    },
    get: (id: string) => request<{ form: ReraForm; definition: ReraDefinition }>(`/api/v1/rera-forms/${id}`),
    create: (data: { form_type: string; listing_id?: string; contact_id?: string; offer_id?: string; fields?: Record<string, string> }) =>
      request<ReraForm>('/api/v1/rera-forms', { method: 'POST', body: JSON.stringify(data) }),
  },

  tasks: {
    list: (params: { mine?: boolean; status?: string; overdue?: boolean; related_type?: string; related_id?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '' && v !== false) q.set(k, String(v)) })
      return request<{ data: Task[]; total: number }>(`/api/v1/tasks?${q}`)
    },
    create: (data: TaskInput) => request<Task>('/api/v1/tasks', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: TaskInput) => request<Task>(`/api/v1/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    setStatus: (id: string, status: Task['status']) =>
      request(`/api/v1/tasks/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
    delete: (id: string) => request(`/api/v1/tasks/${id}`, { method: 'DELETE' }),
  },

  privacy: {
    consents: (contactId: string) =>
      request<{ current: Record<string, Consent>; history: Consent[] }>(`/api/v1/contacts/${contactId}/consents`),
    recordConsent: (contactId: string, channel: string, status: 'granted' | 'withdrawn', note = '') =>
      request<Consent>(`/api/v1/contacts/${contactId}/consents`, { method: 'POST', body: JSON.stringify({ channel, status, note }) }),
    exportContact: async (contactId: string) =>
      saveBlob(await requestBlob(`/api/v1/contacts/${contactId}/export`), `contact-${contactId}.json`),
    erase: (contactId: string, reason: string) =>
      request(`/api/v1/contacts/${contactId}/erase`, { method: 'POST', body: JSON.stringify({ confirm: 'ERASE', reason }) }),
  },

  contacts: {
    list: (params: { search?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.search) q.set('search', params.search)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/contacts?${q}`)
    },
    get: (id: string) => request(`/api/v1/contacts/${id}`),
    create: (data: unknown) =>
      request('/api/v1/contacts', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/contacts/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    // force: also delete the contact's leads, deals, offers, viewings and WhatsApp threads
    delete: (id: string, force = false) =>
      request(`/api/v1/contacts/${id}${force ? '?force=true' : ''}`, { method: 'DELETE' }),
  },

  // ─── Leads ──────────────────────────────────────────────────────────────────

  leads: {
    kanban: () => request('/api/v1/leads'),
    /** Newest `perStage` leads of every stage plus the real size of each stage. */
    board: (perStage: number = 50) => request(`/api/v1/leads/board?per_stage=${perStage}`),
    get: (id: string) => request(`/api/v1/leads/${id}`),
    create: (data: unknown) =>
      request('/api/v1/leads', { method: 'POST', body: JSON.stringify(data) }),
    updateStage: (id: string, stage: string, closedReason?: string) => {
      const body: Record<string, unknown> = { stage }
      if (closedReason) body.closed_reason = closedReason
      return request(`/api/v1/leads/${id}/stage`, {
        method: 'PATCH',
        body: JSON.stringify(body),
      })
    },
    updateNotes: (id: string, notes: string) =>
      request(`/api/v1/leads/${id}/notes`, {
        method: 'PATCH',
        body: JSON.stringify({ notes }),
      }),
    communications: (id: string, limit: number = 100) => {
      const q = new URLSearchParams()
      q.set('limit', String(limit))
      return request(`/api/v1/leads/${id}/communications?${q}`)
    },
    getTags: (id: string) =>
      request(`/api/v1/leads/${id}/tags`),
    addTag: (id: string, tag: string, category?: string) =>
      request(`/api/v1/leads/${id}/tags`, {
        method: 'POST',
        body: JSON.stringify({ tag, category: category || 'manual' }),
      }),
    removeTag: (id: string, tag: string) =>
      request(`/api/v1/leads/${id}/tags/${encodeURIComponent(tag)}`, { method: 'DELETE' }),
    search: (params: { q?: string; stage?: string; source?: string; assigned_to?: string; contact_id?: string; page?: number; limit?: number; before?: string; before_id?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.q) q.set('q', params.q)
      if (params.stage) q.set('stage', params.stage)
      if (params.source) q.set('source', params.source)
      if (params.assigned_to) q.set('assigned_to', params.assigned_to)
      if (params.contact_id) q.set('contact_id', params.contact_id)
      if (params.page) q.set('offset', String(((params.page || 1) - 1) * (params.limit || 50)))
      if (params.limit) q.set('limit', String(params.limit))
      // Keyset cursor (created_at + id of the last lead already loaded): stays correct when cards move meanwhile.
      if (params.before) q.set('before', params.before)
      if (params.before_id) q.set('before_id', params.before_id)
      return request(`/api/v1/leads/search?${q}`)
    },
    assign: (id: string, userId: string | null) =>
      request(`/api/v1/leads/${id}/assign`, {
        method: 'PATCH',
        body: JSON.stringify({ assigned_to: userId }),
      }),
    delete: (id: string) =>
      request(`/api/v1/leads/${id}`, { method: 'DELETE' }),
    reScore: (id: string) =>
      request(`/api/v1/ai/score-lead/${id}`, { method: 'POST' }),
  },

  // ─── WhatsApp ───────────────────────────────────────────────────────────────

  threads: {
    list: (params: { status?: string; contact_id?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.status) q.set('status', params.status)
      if (params.contact_id) q.set('contact_id', params.contact_id)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/threads?${q}`)
    },
    get: (id: string) => request(`/api/v1/threads/${id}`),
    messages: (id: string) => request(`/api/v1/threads/${id}/messages`),
    close: (id: string) =>
      request(`/api/v1/threads/${id}/close`, { method: 'POST' }),
    reopen: (id: string) =>
      request(`/api/v1/threads/${id}/reopen`, { method: 'POST' }),
  },

  // ─── AI ─────────────────────────────────────────────────────────────────────

  ai: {
    summarize: (threadId: string) =>
      request(`/api/v1/ai/summarize/${threadId}`, { method: 'POST' }),
    extractBuyerProfile: (threadId: string) =>
      request(`/api/v1/ai/extract-buyer-profile/${threadId}`, { method: 'POST' }),
    scoreLead: (id: string) =>
      request(`/api/v1/ai/score-lead/${id}`, { method: 'POST' }),
    scoreContact: (id: string) =>
      request(`/api/v1/ai/score-contact/${id}`, { method: 'POST' }),
    draftReply: (threadId: string) =>
      request(`/api/v1/ai/draft-reply/${threadId}`, { method: 'POST' }),
    describeListing: (data: unknown) =>
      request('/api/v1/ai/describe-listing', { method: 'POST', body: JSON.stringify(data) }),
  },

  // ─── Deals ──────────────────────────────────────────────────────────────────

  deals: {
    list: (params: { page?: number; limit?: number; stage?: string; owner_id?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      if (params.stage) q.set('stage', params.stage)
      if (params.owner_id) q.set('owner_id', params.owner_id)
      return request(`/api/v1/deals?${q}`)
    },
    get: (id: string) => request(`/api/v1/deals/${id}`),
    create: (data: unknown) =>
      request('/api/v1/deals', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/deals/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    updateStage: (id: string, stage: string) =>
      request(`/api/v1/deals/${id}/stage`, {
        method: 'PATCH',
        body: JSON.stringify({ stage }),
      }),
    invoices: (id: string) => request(`/api/v1/deals/${id}/invoices`),
  },

  // ─── Invoices ────────────────────────────────────────────────────────────────

  invoices: {
    list: (page = 1, limit = 50, status = '') =>
      request(`/api/v1/invoices?page=${page}&limit=${limit}${status ? `&status=${status}` : ''}`),
    create: (data: unknown) =>
      request('/api/v1/invoices', { method: 'POST', body: JSON.stringify(data) }),
    get: (id: string) => request(`/api/v1/invoices/${id}`),
    getPDF: (id: string) => {
      return `${BASE}/api/v1/invoices/${id}/pdf`
    },
    send: (id: string) =>
      request(`/api/v1/invoices/${id}/send`, { method: 'POST' }),
    updateStatus: (id: string, status: string, reason?: string) =>
      request(`/api/v1/invoices/${id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status, reason }),
      }),
    creditNotes: (id: string) => request(`/api/v1/invoices/${id}/credit-notes`),
    createCreditNote: (id: string, data: { reason: string; lines?: unknown[] }) =>
      request(`/api/v1/invoices/${id}/credit-notes`, { method: 'POST', body: JSON.stringify(data) }),
  },

  // ─── E-invoicing (PINT AE) ───────────────────────────────────────────────────

  einvoice: {
    status: (invoiceId: string) => request<EInvoiceStatus>(`/api/v1/invoices/${invoiceId}/einvoice`),
    /** Builds the PINT AE XML without submitting it, and saves it as a file. */
    exportXML: async (invoiceId: string, body: EInvoiceRequest, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/invoices/${invoiceId}/einvoice/xml`, { method: 'POST', body: JSON.stringify(body) }), filename),
    submit: (invoiceId: string, body: EInvoiceRequest) =>
      request<EInvoiceRecord>(`/api/v1/invoices/${invoiceId}/einvoice`, { method: 'POST', body: JSON.stringify(body) }),
    /** The XML that was submitted. */
    storedXML: async (invoiceId: string, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/invoices/${invoiceId}/einvoice.xml`), filename),
    settings: () => request<{ settings: EInvoiceSettings; providers: string[] }>('/api/v1/settings/einvoicing'),
    updateSettings: (s: EInvoiceSettings) =>
      request<EInvoiceSettings>('/api/v1/settings/einvoicing', { method: 'PUT', body: JSON.stringify(s) }),
  },

  // ─── Stats ───────────────────────────────────────────────────────────────────

  stats: {
    overview: () => request('/api/v1/stats'),
  },

  // ─── User Settings ───────────────────────────────────────────────────────────

  users: {
    me: () => request('/api/v1/users/me'),
    changePassword: (currentPassword: string, newPassword: string) =>
      request('/api/v1/users/me/password', {
        method: 'PATCH',
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
      }),
    updateLang: (lang: 'ar' | 'en') =>
      request<{ lang_pref: string }>('/api/v1/users/me/lang', {
        method: 'PATCH',
        body: JSON.stringify({ lang }),
      }),
    list: () => request('/api/v1/users'),
    invite: (data: { name: string; email: string; role: string }) =>
      request('/api/v1/users/invite', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: { name: string; role: string }) =>
      request(`/api/v1/users/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    setBRN: (id: string, brn: string, brnExpiry: string) =>
      request(`/api/v1/users/${id}/brn`, { method: 'PATCH', body: JSON.stringify({ brn, brn_expiry: brnExpiry }) }),
    setActive: (id: string, active: boolean) =>
      request(`/api/v1/users/${id}/active`, { method: 'PATCH', body: JSON.stringify({ active }) }),
    delete: (id: string) =>
      request(`/api/v1/users/${id}`, { method: 'DELETE' }),
  },

  // ─── Notifications ───────────────────────────────────────────────────────────

  notifications: {
    list: (params: { page?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      return request(`/api/v1/notifications?${q}`)
    },
    markRead: (id: string) =>
      request(`/api/v1/notifications/${id}/read`, { method: 'PATCH' }),
    markAllRead: () =>
      request(`/api/v1/notifications/read-all`, { method: 'PATCH' }),
  },

  // ─── Rental Properties ────────────────────────────────────────────────────────

  rentalProperties: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/rental-properties?${q}`)
    },
    get: (id: string) => request(`/api/v1/rental-properties/${id}`),
    create: (data: unknown) =>
      request('/api/v1/rental-properties', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/rental-properties/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/rental-properties/${id}`, { method: 'DELETE' }),
  },

  // ─── Listings ─────────────────────────────────────────────────────────────────

  listings: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/listings?${q}`)
    },
    get: (id: string) => request(`/api/v1/listings/${id}`),
    create: (data: unknown) =>
      request('/api/v1/listings', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/listings/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    updateStatus: (id: string, status: string) =>
      request(`/api/v1/listings/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
    setPermitOverride: (id: string, reason: string) =>
      request(`/api/v1/listings/${id}/permit-override`, { method: 'POST', body: JSON.stringify({ reason }) }),
    clearPermitOverride: (id: string) =>
      request(`/api/v1/listings/${id}/permit-override`, { method: 'DELETE' }),
    // force: also delete the offers on the listing
    delete: (id: string, force = false) =>
      request(`/api/v1/listings/${id}${force ? '?force=true' : ''}`, { method: 'DELETE' }),
  },

  // ─── Tenants ──────────────────────────────────────────────────────────────────

  tenants: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/tenants?${q}`)
    },
    get: (id: string) => request(`/api/v1/tenants/${id}`),
    create: (data: unknown) =>
      request('/api/v1/tenants', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/tenants/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/tenants/${id}`, { method: 'DELETE' }),
    verify: (id: string, verificationNotes: string) =>
      request(`/api/v1/tenants/${id}/verify`, {
        method: 'POST',
        body: JSON.stringify({ verification_notes: verificationNotes }),
      }),
  },

  // ─── Messages ─────────────────────────────────────────────────────────────────

  messages: {
    suggestAction: (data: unknown) =>
      request('/api/v1/messages/suggest-action', { method: 'POST', body: JSON.stringify(data) }),
    analyze: (data: unknown) =>
      request('/api/v1/messages/analyze', { method: 'POST', body: JSON.stringify(data) }),
    autoCreateLead: (data: unknown) =>
      request('/api/v1/messages/auto-create-lead', { method: 'POST', body: JSON.stringify(data) }),
  },

  // ─── Lease Templates ──────────────────────────────────────────────────────────

  leaseTemplates: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/lease-templates?${q}`)
    },
    get: (id: string) => request(`/api/v1/lease-templates/${id}`),
    create: (data: unknown) =>
      request('/api/v1/lease-templates', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/lease-templates/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/lease-templates/${id}`, { method: 'DELETE' }),
  },

  // ─── Leases ────────────────────────────────────────────────────────────────────

  leases: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/leases?${q}`)
    },
    get: (id: string) => request(`/api/v1/leases/${id}`),
    create: (data: unknown) =>
      request('/api/v1/leases', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/leases/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/leases/${id}`, { method: 'DELETE' }),
  },

  // ─── Payments ─────────────────────────────────────────────────────────────────

  payments: {
    list: (params: { page?: number; limit?: number; lease_id?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      if (params.lease_id) q.set('lease_id', params.lease_id)
      return request(`/api/v1/payments?${q}`)
    },
    get: (id: string) => request(`/api/v1/payments/${id}`),
    create: (data: unknown) =>
      request('/api/v1/payments', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/payments/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/payments/${id}`, { method: 'DELETE' }),
  },

  // ─── Bank Integrations ────────────────────────────────────────────────────────

  bankIntegrations: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/bank-integrations?${q}`)
    },
    get: (id: string) => request(`/api/v1/bank-integrations/${id}`),
    create: (data: unknown) =>
      request('/api/v1/bank-integrations', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/bank-integrations/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/bank-integrations/${id}`, { method: 'DELETE' }),
  },

  // ─── Bank Statements ──────────────────────────────────────────────────────

  bankStatements: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/bank-statements?${q}`)
    },
    get: (id: string) => request(`/api/v1/bank-statements/${id}`),
    upload: (file: File, bankIntegrationId: string) => {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('bank_integration_id', bankIntegrationId)
      const token = getToken()
      return fetch(`${BASE}/api/v1/bank-statements/upload`, {
        method: 'POST',
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      }).then(async (res) => {
        if (!res.ok) throw new Error((await res.json()).error || 'Upload failed')
        return res.json()
      })
    },
    // Authenticated download: a plain <a href> cannot send the bearer token.
    download: async (id: string, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/bank-statements/${id}/download`), filename),
    delete: (id: string) =>
      request(`/api/v1/bank-statements/${id}`, { method: 'DELETE' }),
  },

  // ─── Payment Confirmations ────────────────────────────────────────────────

  paymentConfirmations: {
    getByPayment: (paymentId: string) =>
      request(`/api/v1/payments/${paymentId}/confirmation`),
    send: (paymentId: string) =>
      request(`/api/v1/payments/${paymentId}/send-confirmation`, { method: 'POST' }),
  },

  // ─── Analytics ────────────────────────────────────────────────────────────

  analytics: {
    getTenantOverview: () =>
      request('/api/v1/analytics/tenant-overview'),
    listProperties: (params: { limit?: number; offset?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.limit) q.set('limit', String(params.limit))
      if (params.offset) q.set('offset', String(params.offset))
      return request(`/api/v1/analytics/properties?${q}`)
    },
    getProperty: (propertyId: string) =>
      request(`/api/v1/analytics/properties/${propertyId}`),
    listTenants: (params: { limit?: number; offset?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.limit) q.set('limit', String(params.limit))
      if (params.offset) q.set('offset', String(params.offset))
      return request(`/api/v1/analytics/tenants?${q}`)
    },
    getTenant: (tenantId: string) =>
      request(`/api/v1/analytics/tenants/${tenantId}`),
    getFinancial: (startDate?: string, endDate?: string) => {
      const q = new URLSearchParams()
      if (startDate) q.set('startDate', startDate)
      if (endDate) q.set('endDate', endDate)
      return request(`/api/v1/analytics/financial?${q}`)
    },
    getMaintenance: () =>
      request('/api/v1/analytics/maintenance'),
  },

  // ─── Inspections ───────────────────────────────────────────────────────────

  inspection: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('offset', String((params.page - 1) * (params.limit || 20)))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/inspections?${q}`)
    },
    get: (id: string) =>
      request(`/api/v1/inspections/${id}`),
    create: (data: any) =>
      request('/api/v1/inspections', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: any) =>
      request(`/api/v1/inspections/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
    complete: (id: string) =>
      request(`/api/v1/inspections/${id}/complete`, { method: 'POST' }),
    listTemplates: () =>
      request('/api/v1/inspection-templates'),
    createTemplate: (data: any) =>
      request('/api/v1/inspection-templates', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  },

  // ─── Maintenance Tasks ───────────────────────────────────────────────────────

  maintenance: {
    list: (params: { page?: number; limit?: number; status?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('offset', String((params.page - 1) * (params.limit || 20)))
      if (params.limit) q.set('limit', String(params.limit))
      if (params.status) q.set('status', params.status)
      return request(`/api/v1/maintenance-tasks?${q}`)
    },
    get: (id: string) =>
      request(`/api/v1/maintenance-tasks/${id}`),
    create: (data: any) =>
      request('/api/v1/maintenance-tasks', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: any) =>
      request(`/api/v1/maintenance-tasks/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
      }),
    complete: (id: string, data?: any) =>
      request(`/api/v1/maintenance-tasks/${id}/complete`, {
        method: 'POST',
        body: data ? JSON.stringify(data) : undefined,
      }),
    addPhoto: (id: string, photoUrl: string, stage: string) =>
      request(`/api/v1/maintenance-tasks/${id}/photos`, {
        method: 'POST',
        body: JSON.stringify({ photo_url: photoUrl, photo_stage: stage }),
      }),
    getPhotos: (id: string) =>
      request(`/api/v1/maintenance-tasks/${id}/photos`),
    delete: (id: string) =>
      request(`/api/v1/maintenance-tasks/${id}`, { method: 'DELETE' }),
  },

  // ─── Billing & Plans ─────────────────────────────────────────────────────────

  billing: {
    get: () => request('/api/v1/billing'),
    getUsage: () => request('/api/v1/billing/usage'),
    checkout: (plan: string) =>
      request('/api/v1/billing/checkout', { method: 'POST', body: JSON.stringify({ plan }) }),
    portal: () =>
      request('/api/v1/billing/portal', { method: 'POST' }),
  },

  // ─── Settings ────────────────────────────────────────────────────────────────

  settings: {
    locale: () => request<CompanyLocale>('/api/v1/settings/locale'),
    jurisdictions: () => request<JurisdictionRules[]>('/api/v1/settings/jurisdictions'),
    getCompany: () => request('/api/v1/settings/company'),
    updateCompany: (data: unknown) =>
      request('/api/v1/settings/company', { method: 'PATCH', body: JSON.stringify(data) }),
    getBOS24: () => request('/api/v1/settings/bos24'),
    updateBOS24: (data: unknown) =>
      request('/api/v1/settings/bos24', { method: 'PATCH', body: JSON.stringify(data) }),
    apiKeys: {
      list: () => request('/api/v1/settings/api-keys'),
      create: (data: unknown) =>
        request('/api/v1/settings/api-keys', { method: 'POST', body: JSON.stringify(data) }),
      revoke: (id: string) =>
        request(`/api/v1/settings/api-keys/${id}`, { method: 'DELETE' }),
    },
    webhooks: {
      list: () => request('/api/v1/settings/webhooks'),
      create: (data: unknown) =>
        request('/api/v1/settings/webhooks', { method: 'POST', body: JSON.stringify(data) }),
      delete: (id: string) =>
        request(`/api/v1/settings/webhooks/${id}`, { method: 'DELETE' }),
      test: (id: string) =>
        request(`/api/v1/settings/webhooks/${id}/test`, { method: 'POST' }),
    },
    leadRotation: {
      get: () => request('/api/v1/settings/lead-rotation'),
      update: (data: { mode?: string; enabled: boolean; max_per_agent?: number }) =>
        request('/api/v1/settings/lead-rotation', { method: 'PATCH', body: JSON.stringify(data) }),
    },
  },

  // ─── WhatsApp Outbound ─────────────────────────────────────────────────────────

  whatsapp: {
    sendMessage: (threadId: string, body: string) =>
      request(`/api/v1/threads/${threadId}/send-message`, {
        method: 'POST',
        body: JSON.stringify({ message: body }),
      }),
    sendMedia: (threadId: string, data: { media_url: string; media_type: string; caption?: string }) =>
      request(`/api/v1/threads/${threadId}/send-media`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    /** Sends a file from the user's device (uploaded to WhatsApp, not a public link). */
    sendMediaFile: (threadId: string, file: File, mediaType: string, caption?: string) => {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('media_type', mediaType)
      if (caption) fd.append('caption', caption)
      return request(`/api/v1/threads/${threadId}/send-media`, { method: 'POST', body: fd })
    },
    /** Fetches a message attachment with the Authorization header (a plain URL cannot carry it). */
    fetchMedia: (threadId: string, messageId: string) =>
      requestBlob(`/api/v1/threads/${threadId}/messages/${messageId}/media`),
    downloadMedia: async (threadId: string, messageId: string, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/threads/${threadId}/messages/${messageId}/media`), filename),
    sendTemplate: (threadId: string, templateName: string, params: string[]) =>
      request(`/api/v1/threads/${threadId}/send-template`, {
        method: 'POST',
        body: JSON.stringify({ template_name: templateName, parameters: params }),
      }),
    getOutboundMessages: (threadId: string) =>
      request(`/api/v1/threads/${threadId}/outbound-messages`),
  },

  // ─── Email ────────────────────────────────────────────────────────────────────

  email: {
    list: (page = 1, limit = 50) =>
      request(`/api/v1/emails?page=${page}&limit=${limit}`),
    send: (data: unknown) =>
      request('/api/v1/emails/send', { method: 'POST', body: JSON.stringify(data) }),
    history: (relatedTo: string, relatedId: number) =>
      request(`/api/v1/emails/history?related_to=${relatedTo}&related_id=${relatedId}`),
  },

  // ─── Expenses ─────────────────────────────────────────────────────────────────

  expenses: {
    listCategories: () => request('/api/v1/expense-categories'),
    createCategory: (data: { category_name: string; category_type: string; description?: string }) =>
      request('/api/v1/expense-categories', { method: 'POST', body: JSON.stringify(data) }),
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('offset', String((params.page - 1) * (params.limit || 20)))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/expenses?${q}`)
    },
    get: (id: string) => request(`/api/v1/expenses/${id}`),
    create: (data: {
      category_id: string; amount: number; expense_date: string; description: string;
      property_id?: string; tenant_id?: string; vendor_name?: string;
      vendor_contact?: string; payment_method?: string; receipt_url?: string; notes?: string;
    }) => request('/api/v1/expenses', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: { amount?: number; description?: string; payment_status?: string }) =>
      request(`/api/v1/expenses/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/expenses/${id}`, { method: 'DELETE' }),
    approve: (id: string, data?: { comments?: string }) =>
      request(`/api/v1/expenses/${id}/approve`, { method: 'POST', body: data ? JSON.stringify(data) : undefined }),
  },

  // ─── Lease Renewals ───────────────────────────────────────────────────────────

  leaseRenewals: {
    list: (params: { page?: number; limit?: number; status?: string } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      if (params.status) q.set('status', params.status)
      return request(`/api/v1/lease-renewals?${q}`)
    },
    get: (id: string) => request(`/api/v1/lease-renewals/${id}`),
    initiate: (leaseId: string) =>
      request(`/api/v1/lease-renewals/${leaseId}/initiate`, { method: 'POST' }),
    propose: (id: string, data: { proposed_rent_amount: number; proposed_terms?: Record<string, unknown> }) =>
      request(`/api/v1/lease-renewals/${id}/propose`, { method: 'PUT', body: JSON.stringify(data) }),
    sendOffer: (id: string, data: { template_id: string; communication_type: string }) =>
      request(`/api/v1/lease-renewals/${id}/send-offer`, { method: 'POST', body: JSON.stringify(data) }),
    accept: (id: string) =>
      request(`/api/v1/lease-renewals/${id}/accept`, { method: 'PUT' }),
    reject: (id: string) =>
      request(`/api/v1/lease-renewals/${id}/reject`, { method: 'PUT' }),
    counterOffer: (id: string, data: { counter_offer_amount: number }) =>
      request(`/api/v1/lease-renewals/${id}/counter-offer`, { method: 'POST', body: JSON.stringify(data) }),
    templates: {
      list: () => request('/api/v1/renewal-templates'),
      create: (data: { template_name: string; email_subject: string; email_body: string; whatsapp_message: string; language: string }) =>
        request('/api/v1/renewal-templates', { method: 'POST', body: JSON.stringify(data) }),
      update: (id: string, data: unknown) =>
        request(`/api/v1/renewal-templates/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
      delete: (id: string) =>
        request(`/api/v1/renewal-templates/${id}`, { method: 'DELETE' }),
    },
  },

  // ─── Public (no auth) ───────────────────────────────────────────────────────

  public: {
    getSignature: (signatureId: string) =>
      fetch(`${BASE}/api/public/sign/${signatureId}`).then((r) => r.json()),
    sign: (signatureId: string) =>
      fetch(`${BASE}/api/public/sign/${signatureId}`, { method: 'POST' }).then((r) => r.json()),
  },

  // ─── Documents ──────────────────────────────────────────────────────────────

  documents: {
    // Templates
    templates: {
      list: (page: number = 1, limit: number = 20) =>
        request(`/api/v1/documents/templates?page=${page}&limit=${limit}`),
      get: (id: string) => request(`/api/v1/documents/templates/${id}`),
      create: (data: unknown) =>
        request('/api/v1/documents/templates', { method: 'POST', body: JSON.stringify(data) }),
      update: (id: string, data: unknown) =>
        request(`/api/v1/documents/templates/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
      delete: (id: string) =>
        request(`/api/v1/documents/templates/${id}`, { method: 'DELETE' }),
    },

    // Documents
    list: (entityType: string, entityId: string) =>
      request(`/api/v1/documents?entity_type=${entityType}&entity_id=${entityId}`),
    get: (id: string) => request(`/api/v1/documents/${id}`),
    create: (data: unknown) =>
      request('/api/v1/documents', { method: 'POST', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/documents/${id}`, { method: 'DELETE' }),

    // Signatures
    requestSignature: (docId: string, signerName: string, signerEmail: string) =>
      request(`/api/v1/documents/${docId}/request-signature`, {
        method: 'POST',
        body: JSON.stringify({ signer_name: signerName, signer_email: signerEmail }),
      }),
    markSigned: (signatureId: string) =>
      request(`/api/v1/documents/signatures/${signatureId}/mark-signed`, { method: 'PATCH' }),
  },

  // ─── BOS24 / DLD Market Data ──────────────────────────────────────────────

  bos24: {
    search: (query: string, limit: number = 20) =>
      request('/api/v1/properties/search', {
        method: 'POST',
        body: JSON.stringify({ query, limit }),
      }),
    transactions: (filters: Record<string, string | number | undefined> = {}) => {
      const q = new URLSearchParams()
      Object.entries(filters).forEach(([k, v]) => { if (v !== undefined) q.set(k, String(v)) })
      return request(`/api/v1/properties/transactions?${q}`)
    },
    reportPdf: (payload: Record<string, unknown>) =>
      requestBlob('/api/v1/properties/report/pdf', {
        method: 'POST',
        body: JSON.stringify(payload),
      }),

    // ── Map data endpoints ─────────────────────────────────────────────────
    map: {
      areas: () => request('/api/v1/properties/map/areas'),
      bounds: () => request('/api/v1/properties/map/bounds'),
      heatmap: () => request('/api/v1/properties/map/heatmap'),
      areaLocation: (name: string) => request(`/api/v1/properties/map/area/${encodeURIComponent(name)}`),
      poiCategories: () => request('/api/v1/properties/map/poi-categories'),
      pois: (lat: number, lng: number, radiusKm: number, category?: string) => {
        const q = new URLSearchParams({ lat: String(lat), lng: String(lng), radius_km: String(radiusKm), limit: '50' })
        if (category) q.set('category', category)
        return request(`/api/v1/properties/pois?${q}`)
      },
      transactionAreas: (propertyType?: string, transType?: string) => {
        const q = new URLSearchParams({ limit: '50' })
        if (propertyType) q.set('property_type', propertyType)
        if (transType) q.set('trans_type', transType)
        return request(`/api/v1/properties/transactions/areas?${q}`)
      },
    },
  },

  // ─── Message Templates ───────────────────────────────────────────────────

  // ─── Audit Log ────────────────────────────────────────────────────────────
  auditLog: {
    list: (params: {
      entity_type?: string
      entity_id?: string
      actor_id?: string
      action?: string
      page?: number
      limit?: number
    } = {}) => {
      const q = new URLSearchParams()
      if (params.entity_type) q.set('entity_type', params.entity_type)
      if (params.entity_id) q.set('entity_id', params.entity_id)
      if (params.actor_id) q.set('actor_id', params.actor_id)
      if (params.action) q.set('action', params.action)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/audit-logs?${q}`)
    },
  },

  cheques: {
    list: (params: { status?: string; from?: string; to?: string; lease_id?: string } = {}) => {
      const q = new URLSearchParams()
      Object.entries(params).forEach(([k, v]) => { if (v) q.set(k, v) })
      return request<Cheque[]>(`/api/v1/cheques?${q}`)
    },
    forLease: (leaseId: string) => request<Cheque[]>(`/api/v1/leases/${leaseId}/cheques`),
    schedule: (leaseId: string, data: { count: number; first_date?: string; first_number?: string; bank_name?: string; drawer_name?: string }) =>
      request<Cheque[]>(`/api/v1/leases/${leaseId}/cheques/schedule`, { method: 'POST', body: JSON.stringify(data) }),
    create: (leaseId: string, data: ChequeInput) =>
      request<Cheque>(`/api/v1/leases/${leaseId}/cheques`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: ChequeInput) =>
      request<Cheque>(`/api/v1/cheques/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    setStatus: (id: string, data: { status: ChequeStatus; date?: string; reason?: string; penalty_amount?: number }) =>
      request<Cheque>(`/api/v1/cheques/${id}/status`, { method: 'POST', body: JSON.stringify(data) }),
    replace: (id: string, data: ChequeInput) =>
      request<Cheque>(`/api/v1/cheques/${id}/replace`, { method: 'POST', body: JSON.stringify(data) }),
    uploadScan: (id: string, file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return request<void>(`/api/v1/cheques/${id}/scan`, { method: 'POST', body: fd })
    },
    downloadScan: async (id: string, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/cheques/${id}/scan`), filename),
  },

  leaseOps: {
    extras: (leaseId: string) => request<LeaseExtras>(`/api/v1/leases/${leaseId}/extras`),
    setExtras: (leaseId: string, data: { agent_id: string | null; ejari_expiry: string; rent_reminders?: boolean }) =>
      request<void>(`/api/v1/leases/${leaseId}/extras`, { method: 'PATCH', body: JSON.stringify(data) }),
    payments: (leaseId: string) => request<Payment[]>(`/api/v1/leases/${leaseId}/payments`),
    reminders: (leaseId: string) => request<PaymentReminderRow[]>(`/api/v1/leases/${leaseId}/reminders`),
    notices: (leaseId: string) => request<LeaseNotice[]>(`/api/v1/leases/${leaseId}/notices`),
    draftNotice: (leaseId: string, data: { kind: NoticeKind; new_annual_rent?: number; effective_date?: string }) =>
      request<NoticeDraft>(`/api/v1/leases/${leaseId}/notices/draft`, { method: 'POST', body: JSON.stringify(data) }),
    createNotice: (leaseId: string, data: { kind: NoticeKind; new_annual_rent?: number; effective_date?: string; body_en: string; body_ar: string }) =>
      request<LeaseNotice>(`/api/v1/leases/${leaseId}/notices`, { method: 'POST', body: JSON.stringify(data) }),
    notice: (id: string) => request<LeaseNotice>(`/api/v1/lease-notices/${id}`),
    markSent: (id: string, data: { via: string; sent_at?: string }) =>
      request<void>(`/api/v1/lease-notices/${id}/sent`, { method: 'POST', body: JSON.stringify(data) }),
    settings: () => request<PropertySettings>('/api/v1/settings/property'),
    updateSettings: (data: PropertySettings) =>
      request<PropertySettings>('/api/v1/settings/property', { method: 'PUT', body: JSON.stringify(data) }),
  },

  broadcasts: {
    preview: (segment: BroadcastSegment) => request<{ total: number; eligible: number; no_consent: number; opted_out: number }>('/api/v1/broadcasts/preview', { method: 'POST', body: JSON.stringify(segment) }),
    list: () => request<Broadcast[]>('/api/v1/broadcasts'),
    get: (id: string) => request<Broadcast>(`/api/v1/broadcasts/${id}`),
    create: (data: { name: string; template_name: string; language: string; params: string[]; segment: BroadcastSegment }) =>
      request<Broadcast>('/api/v1/broadcasts', { method: 'POST', body: JSON.stringify(data) }),
    start: (id: string) => request<Broadcast>(`/api/v1/broadcasts/${id}/start`, { method: 'POST' }),
    cancel: (id: string) => request<void>(`/api/v1/broadcasts/${id}/cancel`, { method: 'POST' }),
    recipients: (id: string, status = '') =>
      request<{ contact_id: string; name: string; phone: string; status: string; skip_reason?: string; error?: string; sent_at?: string }[]>(`/api/v1/broadcasts/${id}/recipients?status=${status}`),
    assignThread: (threadId: string, userId: string | null) =>
      request<void>(`/api/v1/threads/${threadId}/assign`, { method: 'PATCH', body: JSON.stringify({ user_id: userId }) }),
    inboxSettings: () => request<InboxSettings>('/api/v1/settings/inbox'),
    updateInboxSettings: (data: InboxSettings) => request<InboxSettings>('/api/v1/settings/inbox', { method: 'PUT', body: JSON.stringify(data) }),
  },

  kyc: {
    profile: (contactId: string) => request<KYCProfile>(`/api/v1/contacts/${contactId}/kyc`),
    update: (contactId: string, data: Partial<KYCProfile>) =>
      request<KYCProfile>(`/api/v1/contacts/${contactId}/kyc`, { method: 'PUT', body: JSON.stringify(data) }),
    screen: (contactId: string) => request<KYCProfile>(`/api/v1/contacts/${contactId}/kyc/screen`, { method: 'POST' }),
    addDocument: (contactId: string, form: FormData) =>
      request<KYCDocument>(`/api/v1/contacts/${contactId}/kyc/documents`, { method: 'POST', body: form }),
    downloadDocument: async (id: string, filename: string) =>
      saveBlob(await requestBlob(`/api/v1/kyc-documents/${id}/file`), filename),
    deleteDocument: (id: string) => request<void>(`/api/v1/kyc-documents/${id}`, { method: 'DELETE' }),
    owners: (orgId: string) => request<BeneficialOwner[]>(`/api/v1/organisations/${orgId}/owners`),
    setOwner: (orgId: string, data: { contact_id: string; ownership_pct?: number | null; role: string }) =>
      request<void>(`/api/v1/organisations/${orgId}/owners`, { method: 'POST', body: JSON.stringify(data) }),
    removeOwner: (orgId: string, contactId: string) => request<void>(`/api/v1/organisations/${orgId}/owners/${contactId}`, { method: 'DELETE' }),
    deal: (dealId: string) => request<DealKYC>(`/api/v1/deals/${dealId}/kyc`),
    override: (dealId: string, reason: string) =>
      request<void>(`/api/v1/deals/${dealId}/kyc-override`, { method: 'POST', body: JSON.stringify({ reason }) }),
    setPayment: (dealId: string, payment_method: string, cash_amount: number) =>
      request<{ aml_status: string; aml_reasons: string[] }>(`/api/v1/deals/${dealId}/payment`, { method: 'PUT', body: JSON.stringify({ payment_method, cash_amount }) }),
    amlQueue: (status = 'flagged') => request<AMLCase[]>(`/api/v1/compliance/aml?status=${status}`),
    review: (dealId: string, data: { decision: 'no_report' | 'reported'; notes: string; goaml_reference?: string }) =>
      request<void>(`/api/v1/compliance/aml/${dealId}/review`, { method: 'POST', body: JSON.stringify(data) }),
    exportCSV: async (from: string, to: string) =>
      saveBlob(await requestBlob(`/api/v1/compliance/aml/export.csv?from=${from}&to=${to}`), `aml-cases-${from}.csv`),
    settings: () => request<{ cash_threshold: number; flag_virtual_asset: boolean }>('/api/v1/settings/aml'),
    updateSettings: (data: { cash_threshold: number; flag_virtual_asset: boolean }) =>
      request<{ cash_threshold: number; flag_virtual_asset: boolean }>('/api/v1/settings/aml', { method: 'PUT', body: JSON.stringify(data) }),
  },

  twoFactor: {
    status: () => request<{ enabled: boolean; required_by_company: boolean; backup_codes_left: number }>('/api/v1/users/me/2fa'),
    setup: () => request<{ secret: string; otpauth_url: string }>('/api/v1/users/me/2fa/setup', { method: 'POST' }),
    enable: (code: string) => request<{ backup_codes: string[] }>('/api/v1/users/me/2fa/enable', { method: 'POST', body: JSON.stringify({ code }) }),
    disable: (password: string, code: string) =>
      request<void>('/api/v1/users/me/2fa/disable', { method: 'POST', body: JSON.stringify({ password, code }) }),
    newBackupCodes: (code: string) =>
      request<{ backup_codes: string[] }>('/api/v1/users/me/2fa/backup-codes', { method: 'POST', body: JSON.stringify({ code }) }),
    resetUser: (userId: string) => request<void>(`/api/v1/users/${userId}/2fa/reset`, { method: 'POST' }),
    companySettings: () => request<{ require_2fa: boolean; users_without_2fa: number }>('/api/v1/settings/security'),
    setCompanyRequirement: (require_2fa: boolean) =>
      request<void>('/api/v1/settings/security', { method: 'PUT', body: JSON.stringify({ require_2fa }) }),
  },

  portals: {
    list: () => request<{ connections: PortalConnection[]; portals: Record<string, string> }>('/api/v1/settings/portals'),
    connect: (portal: string) => request<PortalLinks>(`/api/v1/settings/portals/${portal}`, { method: 'POST' }),
    setEnabled: (portal: string, enabled: boolean) =>
      request<void>(`/api/v1/settings/portals/${portal}`, { method: 'PATCH', body: JSON.stringify({ enabled }) }),
    disconnect: (portal: string) => request<void>(`/api/v1/settings/portals/${portal}`, { method: 'DELETE' }),
  },

  reports: {
    leadSources: (from?: string, to?: string) => {
      const q = new URLSearchParams()
      if (from) q.set('from', from)
      if (to) q.set('to', to)
      return request<{ from: string; to: string; rows: LeadSourceRow[] }>(`/api/v1/reports/lead-sources?${q}`)
    },
  },

  jobs: {
    runs: (limit = 100) => request<JobRun[]>(`/api/v1/admin/jobs/runs?limit=${limit}`),
  },

  importExport: {
    // Downloads are fetched with the Authorization header and saved from a Blob. Never put the
    // JWT in a URL: it would leak through history, proxy/server logs and the Referer header.
    downloadTemplate: async (entity: 'contacts' | 'leads' | 'listings') =>
      saveBlob(await requestBlob(`/api/v1/import/template/${entity}`), `${entity}-template.csv`),
    importContacts: async (file: File) => {
      const fd = new FormData(); fd.append('file', file)
      const token = (await import('./auth')).getToken()
      const res = await fetch(`${BASE}/api/v1/import/contacts`, {
        method: 'POST', body: fd,
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) { const e = await res.json().catch(() => ({})); throw new Error(e.error || 'Import failed') }
      return res.json()
    },
    importLeads: async (file: File) => {
      const fd = new FormData(); fd.append('file', file)
      const token = (await import('./auth')).getToken()
      const res = await fetch(`${BASE}/api/v1/import/leads`, {
        method: 'POST', body: fd,
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) { const e = await res.json().catch(() => ({})); throw new Error(e.error || 'Import failed') }
      return res.json()
    },
    exportContacts: async (search?: string) => {
      const q = search ? `?search=${encodeURIComponent(search)}` : ''
      saveBlob(await requestBlob(`/api/v1/export/contacts${q}`), 'contacts.csv')
    },
    exportLeads: async (stage?: string) => {
      const q = stage ? `?stage=${encodeURIComponent(stage)}` : ''
      saveBlob(await requestBlob(`/api/v1/export/leads${q}`), 'leads.csv')
    },
    exportListings: async () =>
      saveBlob(await requestBlob('/api/v1/export/listings'), 'listings.csv'),
  },

  marketing: {
    publicListing: (id: string) => fetch(`${BASE}/api/public/listings/${id}`).then(r => r.json()),
    // Authenticated endpoints: open the result from a Blob (a plain <a href> cannot send the bearer token).
    // Saved rather than opened: a blob: tab inherits this page's CSP, which (object-src 'none') can block PDF viewers.
    downloadBrochure: async (id: string) =>
      saveBlob(await requestBlob(`/api/v1/listings/${id}/brochure`), 'brochure.pdf'),
    openQr: (id: string) => openBlobInTab(() => requestBlob(`/api/v1/listings/${id}/qr`)),
    emailCampaign: (id: string, data: { contact_ids: string[]; subject?: string; message?: string }) =>
      request(`/api/v1/listings/${id}/email-campaign`, { method: 'POST', body: JSON.stringify(data) }),
  },

  viewings: {
    list: (params: {
      agent_id?: string; contact_id?: string; listing_id?: string
      status?: string; from?: string; to?: string; page?: number; limit?: number
    } = {}) => {
      const q = new URLSearchParams()
      Object.entries(params).forEach(([k, v]) => { if (v !== undefined) q.set(k, String(v)) })
      return request(`/api/v1/viewings?${q}`)
    },
    get: (id: string) => request(`/api/v1/viewings/${id}`),
    create: (data: {
      contact_id: string; scheduled_at: string; duration_min?: number
      listing_id?: string; agent_id?: string; lead_id?: string
      address?: string; notes?: string
    }) => request('/api/v1/viewings', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/viewings/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    updateStatus: (id: string, status: string) =>
      request(`/api/v1/viewings/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
    delete: (id: string) =>
      request(`/api/v1/viewings/${id}`, { method: 'DELETE' }),
  },

  performance: {
    leaderboard: (period?: string, metric?: string) => {
      const q = new URLSearchParams()
      if (period) q.set('period', period)
      if (metric) q.set('metric', metric)
      return request(`/api/v1/performance/leaderboard?${q}`)
    },
    agentKPIs: (agentId: string, period?: string) => {
      const q = new URLSearchParams()
      if (period) q.set('period', period)
      return request(`/api/v1/performance/agent/${agentId}?${q}`)
    },
    agentTrends: (agentId: string) =>
      request(`/api/v1/performance/agent/${agentId}/trends`),
    agentTargets: (agentId: string, period?: string) => {
      const q = new URLSearchParams()
      if (period) q.set('period', period)
      return request(`/api/v1/performance/agent/${agentId}/targets?${q}`)
    },
    upsertTarget: (data: { agent_id: string; metric: string; target_value: number; period?: string }) =>
      request('/api/v1/performance/targets', { method: 'POST', body: JSON.stringify(data) }),
    deleteTarget: (id: string) =>
      request(`/api/v1/performance/targets/${id}`, { method: 'DELETE' }),
  },

  commissions: {
    list: (params: { agent_id?: string; status?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.agent_id) q.set('agent_id', params.agent_id)
      if (params.status) q.set('status', params.status)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/commissions?${q}`)
    },
    create: (data: unknown) =>
      request('/api/v1/commissions', { method: 'POST', body: JSON.stringify(data) }),
    updateStatus: (id: string, status: string, paymentReference?: string) =>
      request(`/api/v1/commissions/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status, payment_reference: paymentReference }) }),
    updateAmount: (id: string, total: number, notes?: string) =>
      request(`/api/v1/commissions/${id}/amount`, { method: 'PATCH', body: JSON.stringify({ total_commission: total, notes }) }),
    calculate: (agentId: string, periodFrom: string, periodTo: string) =>
      request('/api/v1/commissions/calculate', { method: 'POST', body: JSON.stringify({ agent_id: agentId, period_from: periodFrom, period_to: periodTo }) }),
    structures: {
      list: () => request('/api/v1/commissions/structures'),
      create: (data: unknown) => request('/api/v1/commissions/structures', { method: 'POST', body: JSON.stringify(data) }),
      update: (id: string, data: unknown) => request(`/api/v1/commissions/structures/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
      delete: (id: string) => request(`/api/v1/commissions/structures/${id}`, { method: 'DELETE' }),
    },
  },

  offers: {
    list: (params: { listing_id?: string; contact_id?: string; status?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.listing_id) q.set('listing_id', params.listing_id)
      if (params.contact_id) q.set('contact_id', params.contact_id)
      if (params.status) q.set('status', params.status)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/offers?${q}`)
    },
    get: (id: string) => request(`/api/v1/offers/${id}`),
    create: (data: {
      listing_id: string; contact_id: string; offer_amount: number;
      currency?: string; terms?: string; notes?: string; valid_until?: string
    }) => request('/api/v1/offers', { method: 'POST', body: JSON.stringify(data) }),
    updateStatus: (id: string, status: string) =>
      request(`/api/v1/offers/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
    counter: (id: string, data: { offer_amount: number; terms?: string; notes?: string; valid_until?: string }) =>
      request(`/api/v1/offers/${id}/counter`, { method: 'POST', body: JSON.stringify(data) }),
    accept: (id: string) =>
      request(`/api/v1/offers/${id}/accept`, { method: 'POST' }),
    delete: (id: string) =>
      request(`/api/v1/offers/${id}`, { method: 'DELETE' }),
  },

  messageTemplates: {
    list: (params: { page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/message-templates?${q}`)
    },
    get: (id: string) => request(`/api/v1/message-templates/${id}`),
    create: (data: unknown) =>
      request('/api/v1/message-templates', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/message-templates/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/message-templates/${id}`, { method: 'DELETE' }),
  },

  pipelineStages: {
    list: (entityType = 'lead') =>
      request(`/api/v1/pipeline-stages?entity_type=${entityType}`),
    create: (data: unknown) =>
      request('/api/v1/pipeline-stages', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request(`/api/v1/pipeline-stages/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    delete: (id: string) =>
      request(`/api/v1/pipeline-stages/${id}`, { method: 'DELETE' }),
    reorder: (ids: string[]) =>
      request('/api/v1/pipeline-stages/reorder', { method: 'POST', body: JSON.stringify({ ids }) }),
    resetDefault: (entityType = 'lead') =>
      request(`/api/v1/pipeline-stages/reset-default?entity_type=${entityType}`, { method: 'POST' }),
  },

  approvals: {
    getConfig: () => request('/api/v1/approval-config'),
    saveConfig: (data: { listing_approval?: boolean; deal_approval_above?: number; offer_approval_above?: number }) =>
      request('/api/v1/approval-config', { method: 'PATCH', body: JSON.stringify(data) }),
    listRequests: (params: { status?: string; entity_type?: string; page?: number; limit?: number } = {}) => {
      const q = new URLSearchParams()
      if (params.status) q.set('status', params.status)
      if (params.entity_type) q.set('entity_type', params.entity_type)
      if (params.page) q.set('page', String(params.page))
      if (params.limit) q.set('limit', String(params.limit))
      return request(`/api/v1/approval-requests?${q}`)
    },
    review: (id: string, status: string, note?: string) =>
      request(`/api/v1/approval-requests/${id}/review`, { method: 'POST', body: JSON.stringify({ status, note: note || '' }) }),
  },
}

export default api
