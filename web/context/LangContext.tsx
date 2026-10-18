'use client'
import { createContext, useContext, useEffect, useState, useCallback, useMemo } from 'react'

type Lang = 'ar' | 'en'

interface LangContextValue {
  lang: Lang
  setLang: (l: Lang) => void
  isRtl: boolean
  t: (ar: string, en: string) => string
}

const LangContext = createContext<LangContextValue>({
  lang: 'ar',
  setLang: () => {},
  isRtl: true,
  t: (ar) => ar,
})

export function LangProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>('ar')

  useEffect(() => {
    const saved = localStorage.getItem('masaar_lang') as Lang | null
    if (saved) setLangState(saved)
  }, [])

  // Memoize setLang to preserve function reference identity
  const setLang = useCallback((l: Lang) => {
    setLangState(l)
    localStorage.setItem('masaar_lang', l)
    document.documentElement.dir = l === 'ar' ? 'rtl' : 'ltr'
    document.documentElement.lang = l
  }, [])

  useEffect(() => {
    document.documentElement.dir = lang === 'ar' ? 'rtl' : 'ltr'
    document.documentElement.lang = lang
  }, [lang])

  // Memoize translation function to prevent recreating function reference on every re-render
  const t = useCallback((ar: string, en: string) => (lang === 'ar' ? ar : en), [lang])

  // Memoize context value object to prevent cascading re-renders in all consumers across the app
  const value = useMemo(
    () => ({
      lang,
      setLang,
      isRtl: lang === 'ar',
      t,
    }),
    [lang, setLang, t]
  )

  return (
    <LangContext.Provider value={value}>
      {children}
    </LangContext.Provider>
  )
}

export const useLang = () => useContext(LangContext)
