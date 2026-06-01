'use client'
import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import { useLang } from '@/context/LangContext'

export default function SettingsPage() {
  const { user, updateUser, company } = useAuthStore()
  const { lang, setLang, t } = useLang()
  const isDemo = !!company?.is_demo

  // Change password form
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [pwdSubmitting, setPwdSubmitting] = useState(false)
  const [pwdError, setPwdError] = useState('')
  const [pwdSuccess, setPwdSuccess] = useState(false)

  // Language form
  const [langSubmitting, setLangSubmitting] = useState(false)
  const [langError, setLangError] = useState('')
  const [langSuccess, setLangSuccess] = useState(false)

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setPwdError('')
    setPwdSuccess(false)

    if (newPassword !== confirmPassword) {
      setPwdError(t('كلمتا المرور غير متطابقتين', 'Passwords do not match'))
      return
    }
    if (newPassword.length < 8) {
      setPwdError(t('كلمة المرور يجب أن تكون 8 أحرف على الأقل', 'Password must be at least 8 characters'))
      return
    }

    setPwdSubmitting(true)
    try {
      await api.users.changePassword(currentPassword, newPassword)
      setPwdSuccess(true)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (err: unknown) {
      setPwdError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setPwdSubmitting(false)
    }
  }

  const handleUpdateLang = async (newLang: 'ar' | 'en') => {
    setLangError('')
    setLangSuccess(false)
    setLangSubmitting(true)
    try {
      // Demo: apply language locally without persisting to backend
      if (isDemo) {
        setLang(newLang)
        updateUser({ lang_pref: newLang })
        setLangSuccess(true)
        return
      }
      await api.users.updateLang(newLang)
      setLang(newLang)
      updateUser({ lang_pref: newLang })
      setLangSuccess(true)
    } catch (err: unknown) {
      setLangError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setLangSubmitting(false)
    }
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('الإعدادات', 'Settings')} />

      <main className="flex-1 overflow-y-auto p-6 bg-gray-50">
        <div className="max-w-lg space-y-6">

          {/* Demo notice */}
          {isDemo && (
            <div className="flex items-center gap-2.5 p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800">
              <svg className="w-4 h-4 shrink-0 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
              </svg>
              <span>
                <strong>Demo account</strong> — password changes are disabled.{' '}
                <a href="/signup" className="underline font-semibold">Start a free trial</a> to manage your own account.
              </span>
            </div>
          )}

          {/* Profile info */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-4">
              {t('معلومات الحساب', 'Account Info')}
            </h2>
            <div className="space-y-2 text-sm text-gray-600">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-brand-600 text-white font-semibold flex items-center justify-center text-base">
                  {user?.name?.[0]?.toUpperCase() ?? 'U'}
                </div>
                <div>
                  <p className="font-medium text-gray-800">{user?.name}</p>
                  <p className="text-gray-500 text-xs">{user?.email}</p>
                </div>
              </div>
              <p className="text-xs text-gray-400 pt-1">
                {t('الدور:', 'Role:')} <span className="font-medium text-gray-600">{user?.role}</span>
              </p>
            </div>
          </div>

          {/* Language preference */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-4">
              {t('اللغة', 'Language')}
            </h2>
            <div className="flex gap-3">
              <button
                onClick={() => handleUpdateLang('en')}
                disabled={langSubmitting || lang === 'en'}
                className={`flex-1 py-2.5 rounded-lg text-sm font-medium border transition-colors ${
                  lang === 'en'
                    ? 'bg-brand-600 text-white border-brand-600'
                    : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                }`}
              >
                English
              </button>
              <button
                onClick={() => handleUpdateLang('ar')}
                disabled={langSubmitting || lang === 'ar'}
                className={`flex-1 py-2.5 rounded-lg text-sm font-medium border transition-colors ${
                  lang === 'ar'
                    ? 'bg-brand-600 text-white border-brand-600'
                    : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                }`}
              >
                العربية
              </button>
            </div>
            {langSuccess && (
              <p className="mt-3 text-xs text-green-600">
                {t('تم تحديث اللغة', 'Language updated')}
              </p>
            )}
            {langError && (
              <p className="mt-3 text-xs text-red-500">{langError}</p>
            )}
          </div>

          {/* Change password */}
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h2 className="text-sm font-semibold text-gray-700 mb-4">
              {t('تغيير كلمة المرور', 'Change Password')}
            </h2>
            <form onSubmit={handleChangePassword} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  {t('كلمة المرور الحالية', 'Current Password')}
                </label>
                <input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  {t('كلمة المرور الجديدة', 'New Password')}
                </label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  minLength={8}
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">
                  {t('تأكيد كلمة المرور', 'Confirm New Password')}
                </label>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent"
                />
              </div>

              {pwdError && (
                <p className="text-xs text-red-500">{pwdError}</p>
              )}
              {pwdSuccess && (
                <p className="text-xs text-green-600">
                  {t('تم تغيير كلمة المرور بنجاح', 'Password changed successfully')}
                </p>
              )}

              <button
                type="submit"
                disabled={pwdSubmitting || isDemo}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 disabled:opacity-50 transition-colors"
              >
                {pwdSubmitting
                  ? t('جارٍ التحديث...', 'Updating...')
                  : isDemo
                  ? t('غير متاح في الحساب التجريبي', 'Not available in demo')
                  : t('تحديث كلمة المرور', 'Update Password')}
              </button>
            </form>
          </div>

        </div>
      </main>
    </div>
  )
}
