'use client'
import { createContext, useContext, useEffect, useState } from 'react'

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

  const setLang = (l: Lang) => {
    setLangState(l)
    localStorage.setItem('masaar_lang', l)
    document.documentElement.dir = l === 'ar' ? 'rtl' : 'ltr'
    document.documentElement.lang = l
  }

  useEffect(() => {
    document.documentElement.dir = lang === 'ar' ? 'rtl' : 'ltr'
    document.documentElement.lang = lang
  }, [lang])

  const t = (ar: string, en: string) => lang === 'ar' ? ar : en

  return (
    <LangContext.Provider value={{ lang, setLang, isRtl: lang === 'ar', t }}>
      {children}
    </LangContext.Provider>
  )
}

export const useLang = () => useContext(LangContext)
