'use client'
import { useEffect, useState } from 'react'
import QRCode from 'qrcode'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { LoginResponse, TwoFactorPending } from '@/types'

const input = 'w-full px-3 py-2.5 border border-gray-300 rounded-xl text-center tracking-widest text-lg font-mono'

// Shows the authenticator QR code for a secret.
export function TotpQR({ url, secret }: { url: string; secret: string }) {
  const { t } = useLang()
  const [img, setImg] = useState('')
  useEffect(() => { QRCode.toDataURL(url, { margin: 1, width: 200 }).then(setImg).catch(() => setImg('')) }, [url])
  return (
    <div className="text-center space-y-2">
      {img && <img src={img} alt={t('رمز QR لتطبيق المصادقة', 'QR code for the authenticator app')} className="mx-auto w-48 h-48" />}
      <p className="text-xs text-gray-500">{t('أو أدخل هذا المفتاح يدوياً:', 'Or enter this key by hand:')}</p>
      <p dir="ltr" className="font-mono text-sm break-all select-all bg-gray-50 rounded p-2">{secret.replace(/(.{4})/g, '$1 ').trim()}</p>
    </div>
  )
}

// Lists backup codes once, with a way to copy them.
export function BackupCodes({ codes, onDone }: { codes: string[]; onDone: () => void }) {
  const { t } = useLang()
  return (
    <div className="space-y-3">
      <p className="text-sm text-gray-700">
        {t('احفظ رموز الاسترداد في مكان آمن. يمكن استخدام كل رمز مرة واحدة إذا فقدت هاتفك. لن تظهر مرة أخرى.',
          'Keep these backup codes somewhere safe. Each works once if you lose your phone. They will not be shown again.')}
      </p>
      <ul dir="ltr" className="grid grid-cols-2 gap-1 font-mono text-sm bg-gray-50 rounded-lg p-3">
        {codes.map((c) => <li key={c}>{c}</li>)}
      </ul>
      <div className="flex gap-2">
        <button type="button" onClick={() => navigator.clipboard?.writeText(codes.join('\n'))} className="flex-1 py-2 text-sm rounded-xl border border-gray-300">
          {t('نسخ', 'Copy')}
        </button>
        <button type="button" onClick={onDone} className="flex-1 py-2 text-sm rounded-xl bg-brand-600 text-white">
          {t('حفظتها', 'I have saved them')}
        </button>
      </div>
    </div>
  )
}

// The second sign-in step: a code from the authenticator app (or a backup code), or, when
// the company requires two-factor and the user has none yet, setting it up.
export default function TwoFactorStep({ pending, onSuccess, onCancel }: {
  pending: TwoFactorPending; onSuccess: (res: LoginResponse) => void; onCancel: () => void
}) {
  const { t } = useLang()
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [setup, setSetup] = useState<{ secret: string; otpauth_url: string } | null>(null)
  const [done, setDone] = useState<LoginResponse | null>(null)

  useEffect(() => {
    if (!pending.setup_required) return
    api.auth.enrollTOTP(pending.pending_token).then(setSetup).catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed'))
  }, [pending])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true); setError('')
    try {
      if (pending.setup_required) {
        const res = await api.auth.confirmEnrollTOTP(pending.pending_token, code)
        if (res.backup_codes?.length) setDone(res)
        else onSuccess(res)
      } else {
        onSuccess(await api.auth.verifyTOTP(pending.pending_token, code))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed')
      setCode('')
    } finally {
      setBusy(false)
    }
  }

  if (done) return <BackupCodes codes={done.backup_codes ?? []} onDone={() => onSuccess(done)} />

  return (
    <form onSubmit={submit} className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold text-gray-900">
          {pending.setup_required ? t('إعداد التحقق بخطوتين', 'Set up two-step verification') : t('التحقق بخطوتين', 'Two-step verification')}
        </h2>
        <p className="text-sm text-gray-600 mt-1">
          {pending.setup_required
            ? t('تتطلب شركتك التحقق بخطوتين. امسح الرمز بتطبيق مصادقة (مثل Google Authenticator أو Microsoft Authenticator) ثم أدخل الرمز المكون من 6 أرقام.',
              'Your company requires two-step verification. Scan the code with an authenticator app (such as Google or Microsoft Authenticator), then enter the 6-digit code.')
            : t('أدخل الرمز المكون من 6 أرقام من تطبيق المصادقة، أو أحد رموز الاسترداد.', 'Enter the 6-digit code from your authenticator app, or one of your backup codes.')}
        </p>
      </div>
      {setup && <TotpQR url={setup.otpauth_url} secret={setup.secret} />}
      <input
        className={input}
        value={code}
        onChange={(e) => setCode(e.target.value)}
        autoComplete="one-time-code"
        inputMode={pending.setup_required ? 'numeric' : 'text'}
        autoFocus
        aria-label={t('الرمز', 'Code')}
        placeholder="123456"
        dir="ltr"
      />
      {error && <p role="alert" className="text-sm text-red-600 bg-red-50 p-2 rounded">{error}</p>}
      <button type="submit" disabled={busy || code.trim().length < 6} className="w-full py-2.5 rounded-xl bg-brand-600 text-white font-medium disabled:opacity-50">
        {t('تحقق', 'Verify')}
      </button>
      <button type="button" onClick={onCancel} className="w-full text-sm text-gray-500 hover:text-gray-700">{t('رجوع', 'Back')}</button>
    </form>
  )
}
