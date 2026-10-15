'use client'
import { useCallback, useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { BackupCodes, TotpQR } from './TwoFactorStep'

const input = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5'

// The signed-in user's two-step verification: turn on (QR + code), new backup codes, off.
export default function TwoFactorSettings({ disabled }: { disabled?: boolean }) {
  const { t } = useLang()
  const [status, setStatus] = useState<{ enabled: boolean; required_by_company: boolean; backup_codes_left: number } | null>(null)
  const [setup, setSetup] = useState<{ secret: string; otpauth_url: string } | null>(null)
  const [codes, setCodes] = useState<string[] | null>(null)
  const [mode, setMode] = useState<'' | 'disable' | 'codes'>('')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => { api.twoFactor.status().then(setStatus).catch(() => setStatus(null)) }, [])
  useEffect(() => { load() }, [load])

  const run = async (fn: () => Promise<void>) => {
    setBusy(true); setError('')
    try { await fn() } catch (e) { setError(e instanceof Error ? e.message : 'Failed') } finally { setBusy(false); setCode(''); setPassword('') }
  }

  if (!status) return null

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6 space-y-4">
      <div>
        <h2 className="text-sm font-semibold text-gray-700">{t('التحقق بخطوتين', 'Two-step verification')}</h2>
        <p className="text-xs text-gray-500 mt-1">
          {status.enabled
            ? t(`مفعّل · ${status.backup_codes_left} من رموز الاسترداد متبقية`, `On · ${status.backup_codes_left} backup codes left`)
            : t('غير مفعّل. يحمي حسابك برمز من تطبيق مصادقة على هاتفك عند تسجيل الدخول.', 'Off. Protects your account with a code from an authenticator app on your phone when you sign in.')}
          {status.required_by_company && ' · ' + t('مطلوب من شركتك', 'required by your company')}
        </p>
      </div>
      {error && <p role="alert" className="text-xs text-red-500">{error}</p>}

      {codes && <BackupCodes codes={codes} onDone={() => { setCodes(null); load() }} />}

      {!codes && !status.enabled && !setup && (
        <button type="button" disabled={busy || disabled} onClick={() => run(async () => setSetup(await api.twoFactor.setup()))}
          className="w-full py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg disabled:opacity-50">
          {t('تفعيل', 'Turn on')}
        </button>
      )}

      {!codes && setup && (
        <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); run(async () => { const r = await api.twoFactor.enable(code); setSetup(null); setCodes(r.backup_codes) }) }}>
          <p className="text-xs text-gray-600">{t('امسح الرمز بتطبيق المصادقة ثم أدخل الرمز المكون من 6 أرقام.', 'Scan the code with your authenticator app, then enter the 6-digit code.')}</p>
          <TotpQR url={setup.otpauth_url} secret={setup.secret} />
          <input className={`${input} text-center font-mono tracking-widest`} dir="ltr" inputMode="numeric" autoComplete="one-time-code" value={code}
            onChange={(e) => setCode(e.target.value)} aria-label={t('الرمز', 'Code')} placeholder="123456" />
          <div className="flex gap-2">
            <button type="submit" disabled={busy || code.length < 6} className="flex-1 py-2.5 bg-brand-600 text-white text-sm rounded-lg disabled:opacity-50">{t('تأكيد', 'Confirm')}</button>
            <button type="button" onClick={() => setSetup(null)} className="flex-1 py-2.5 text-sm rounded-lg border border-gray-300">{t('إلغاء', 'Cancel')}</button>
          </div>
        </form>
      )}

      {!codes && status.enabled && (
        <div className="space-y-3">
          <div className="flex gap-2">
            <button type="button" onClick={() => setMode(mode === 'codes' ? '' : 'codes')} className="flex-1 py-2 text-sm rounded-lg border border-gray-300">{t('رموز استرداد جديدة', 'New backup codes')}</button>
            {!status.required_by_company && (
              <button type="button" onClick={() => setMode(mode === 'disable' ? '' : 'disable')} className="flex-1 py-2 text-sm rounded-lg border border-red-200 text-red-600">{t('إيقاف', 'Turn off')}</button>
            )}
          </div>
          {mode && (
            <form className="space-y-2" onSubmit={(e) => {
              e.preventDefault()
              run(async () => {
                if (mode === 'disable') { await api.twoFactor.disable(password, code); load() } else { setCodes((await api.twoFactor.newBackupCodes(code)).backup_codes) }
                setMode('')
              })
            }}>
              {mode === 'disable' && (
                <input type="password" className={input} value={password} onChange={(e) => setPassword(e.target.value)} placeholder={t('كلمة المرور', 'Password')} aria-label={t('كلمة المرور', 'Password')} required />
              )}
              <input className={`${input} font-mono`} dir="ltr" value={code} onChange={(e) => setCode(e.target.value)} placeholder={t('رمز التطبيق أو رمز استرداد', 'App code or backup code')} aria-label={t('الرمز', 'Code')} required />
              <button type="submit" disabled={busy} className="w-full py-2.5 text-sm rounded-lg bg-gray-900 text-white disabled:opacity-50">{t('متابعة', 'Continue')}</button>
            </form>
          )}
        </div>
      )}
    </div>
  )
}
