'use client'
import { create } from 'zustand'
import { api, type CompanyLocale } from '@/lib/api'

// UAE defaults, used until the company's locale has loaded (or if it cannot be loaded).
export const DEFAULT_LOCALE: CompanyLocale = {
  country: 'AE',
  region: '',
  currency: 'AED',
  currency_decimals: 2,
  timezone: 'Asia/Dubai',
  rules: {
    country: 'AE',
    name_en: 'United Arab Emirates',
    name_ar: 'الإمارات العربية المتحدة',
    currency: 'AED',
    currency_decimals: 2,
    timezone: 'Asia/Dubai',
    weekend: [6, 0],
    vat_rates: { standard: 0.05, zero_rated: 0, exempt: 0, out_of_scope: 0 },
    tax_id_digits: 15,
    regions: [],
  },
}

interface LocaleState {
  locale: CompanyLocale
  /** True once the company locale has loaded or failed (defaults are then kept). */
  ready: boolean
  load: () => Promise<void>
  set: (l: CompanyLocale) => void
}

export const useLocaleStore = create<LocaleState>((set) => ({
  locale: DEFAULT_LOCALE,
  ready: false,
  load: async () => {
    try {
      set({ locale: await api.settings.locale(), ready: true })
    } catch {
      set({ ready: true })
    }
  },
  set: (locale) => set({ locale }),
}))

/** The current company locale, for use outside React components. */
export function currentLocale(): CompanyLocale {
  return useLocaleStore.getState().locale
}
