'use client'
import { create } from 'zustand'
import type { AuthUser, Company } from '@/types'
import { getUser, getToken, saveSession, clearSession, updateUser, isLoggedIn, getCompany, saveCompany, clearCompany } from '@/lib/auth'

interface AuthState {
  user: AuthUser | null
  company: Company | null
  token: string | null
  setSession: (token: string, refreshToken: string, user: AuthUser, company?: Company) => void
  updateUser: (updates: Partial<AuthUser>) => void
  logout: () => void
  init: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  company: null,
  token: null,

  init: () => {
    if (isLoggedIn()) {
      set({ user: getUser(), company: getCompany(), token: getToken() })
    } else {
      clearSession()
      clearCompany()
      set({ user: null, company: null, token: null })
    }
  },

  setSession: (token, refreshToken, user, company) => {
    saveSession(token, refreshToken, user)
    if (company) saveCompany(company)
    set({ user, company, token })
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
