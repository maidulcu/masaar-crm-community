'use client'
import { create } from 'zustand'
import type { AuthUser, Company } from '@/types'
import { getUser, getToken, saveSession, clearSession, updateUser, isLoggedIn, getCompany, saveCompany, clearCompany, purgeLegacyTokens } from '@/lib/auth'
import { silentRefresh } from '@/lib/api'

interface AuthState {
  user: AuthUser | null
  company: Company | null
  token: string | null
  /** False until the first session restore (cookie -> access token) has finished. */
  ready: boolean
  setSession: (token: string, user: AuthUser, company?: Company) => void
  updateUser: (updates: Partial<AuthUser>) => void
  logout: () => void
  init: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  company: null,
  token: null,
  ready: false,

  // The access token is memory-only, so after a reload it is restored from the HttpOnly
  // refresh cookie. Client-side navigations keep the in-memory token and skip the round-trip.
  init: async () => {
    purgeLegacyTokens()
    if (isLoggedIn()) {
      set({ user: getUser(), company: getCompany(), token: getToken(), ready: true })
      return
    }
    const token = await silentRefresh()
    if (token) {
      set({ user: getUser(), company: getCompany(), token, ready: true })
    } else {
      clearSession()
      set({ user: null, company: null, token: null, ready: true })
    }
  },

  setSession: (token, user, company) => {
    saveSession(token, user)
    if (company) saveCompany(company)
    set({ user, company: company ?? getCompany(), token, ready: true })
  },

  updateUser: (updates) => {
    updateUser(updates)
    set((state) => ({ user: state.user ? { ...state.user, ...updates } : null }))
  },

  logout: () => {
    clearSession()
    clearCompany()
    set({ user: null, company: null, token: null })
  },
}))
