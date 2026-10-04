import type { AuthUser, Company } from '@/types'

// The access token lives only in memory: it is gone on reload and unreachable from other
// origins/tabs, and the long-lived refresh token is an HttpOnly cookie the page cannot read.
// Only the non-secret profile (user/company, for instant first paint) is kept in localStorage.
let accessToken: string | null = null

const USER_KEY = 'masaar_user'
const COMPANY_KEY = 'masaar_company'
// Pre-cookie versions stored credentials here; wipe them so old tokens do not linger.
const LEGACY_KEYS = ['masaar_access_token', 'masaar_refresh_token', 'access_token']

export function getToken(): string | null {
  return accessToken
}

export function setToken(token: string | null) {
  accessToken = token
}

export function purgeLegacyTokens() {
  if (typeof window === 'undefined') return
  try { LEGACY_KEYS.forEach((k) => localStorage.removeItem(k)) } catch { /* storage blocked */ }
}

function readJSON<T>(key: string): T | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : null
  } catch { return null }
}

function writeJSON(key: string, value: unknown) {
  try { localStorage.setItem(key, JSON.stringify(value)) } catch { /* storage blocked */ }
}

export function getUser(): AuthUser | null {
  return readJSON<AuthUser>(USER_KEY)
}

export function saveSession(token: string, user: AuthUser) {
  accessToken = token
  writeJSON(USER_KEY, user)
}

export function updateUser(updates: Partial<AuthUser>) {
  const current = getUser()
  if (!current) return
  writeJSON(USER_KEY, { ...current, ...updates })
}

export function getCompany(): Company | null {
  return readJSON<Company>(COMPANY_KEY)
}

export function saveCompany(company: Company) {
  writeJSON(COMPANY_KEY, company)
}

export function clearCompany() {
  try { localStorage.removeItem(COMPANY_KEY) } catch { /* storage blocked */ }
}

export function clearSession() {
  accessToken = null
  try {
    localStorage.removeItem(USER_KEY)
    localStorage.removeItem(COMPANY_KEY)
  } catch { /* storage blocked */ }
  purgeLegacyTokens()
}

/** Returns true only if an in-memory token exists AND its exp claim is in the future. */
export function isLoggedIn(): boolean {
  const token = accessToken
  if (!token) return false
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return typeof payload.exp === 'number' && payload.exp * 1000 > Date.now()
  } catch {
    return false
  }
}
