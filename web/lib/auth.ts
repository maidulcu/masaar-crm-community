import type { AuthUser, Company } from '@/types'

const TOKEN_KEY = 'masaar_access_token'
const REFRESH_KEY = 'masaar_refresh_token'
const USER_KEY = 'masaar_user'
const COMPANY_KEY = 'masaar_company'

export function getToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null
  return localStorage.getItem(REFRESH_KEY)
}

export function getUser(): AuthUser | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function saveSession(accessToken: string, refreshToken: string, user: AuthUser) {
  localStorage.setItem(TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_KEY, refreshToken)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function updateUser(updates: Partial<AuthUser>) {
  const current = getUser()
  if (!current) return
  localStorage.setItem(USER_KEY, JSON.stringify({ ...current, ...updates }))
}

export function getCompany(): Company | null {
  if (typeof window === 'undefined') return null
  const raw = localStorage.getItem(COMPANY_KEY)
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function saveCompany(company: Company) {
  localStorage.setItem(COMPANY_KEY, JSON.stringify(company))
}

export function clearCompany() {
  localStorage.removeItem(COMPANY_KEY)
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem(COMPANY_KEY)
}

/** Returns true only if a token exists AND its exp claim is in the future. */
export function isLoggedIn(): boolean {
  const token = getToken()
  if (!token) return false
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return typeof payload.exp === 'number' && payload.exp * 1000 > Date.now()
  } catch {
    return false
  }
}
