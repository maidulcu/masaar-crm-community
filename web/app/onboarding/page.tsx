'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'

const STEPS = ['Company', 'Team', 'WhatsApp', 'Done']

export default function OnboardingPage() {
  const { company } = useAuthStore()
  const router = useRouter()
  const [step, setStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [trn, setTrn] = useState('')
  const [address, setAddress] = useState('')
  const [inviteEmails, setInviteEmails] = useState('')

  const handleNext = async () => {
    if (step === 0) {
      setLoading(true)
      try {
        await api.settings.updateCompany({ vat_number: trn, business_address: address })
      } catch { /* non-critical */ }
      setLoading(false)
    }
    if (step === 1) {
      const emails = inviteEmails.split('\n').map(e => e.trim()).filter(Boolean)
      for (const email of emails) {
        try {
          await api.users.invite({ name: email, email, role: 'agent' })
        } catch { /* best-effort */ }
      }
    }
    if (step === 2) {
      setStep(3)
      return
    }
    setStep(s => Math.min(s + 1, STEPS.length - 1))
  }

  const handleSkip = () => {
    router.push('/pipeline')
  }

  const handleFinish = () => {
    router.push('/pipeline')
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-blue-50 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Progress */}
        <div className="flex items-center justify-center gap-2 mb-8">
          {STEPS.map((s, i) => (
            <div key={s} className="flex items-center gap-2">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold ${
                i <= step ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-400'
              }`}>
                {i + 1}
              </div>
              <span className={`text-xs font-medium ${i <= step ? 'text-blue-600' : 'text-gray-400'}`}>
                {s}
              </span>
              {i < STEPS.length - 1 && <div className={`w-8 h-0.5 ${i < step ? 'bg-blue-600' : 'bg-gray-200'}`} />}
            </div>
          ))}
        </div>

        <div className="bg-white rounded-2xl shadow-xl border border-gray-100 p-8">
          {step === 0 && (
            <div className="space-y-4">
              <h2 className="text-xl font-bold text-gray-900">Company Details</h2>
              <p className="text-sm text-gray-500">Set up your company profile on {company?.name}.</p>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">TRN / VAT Number</label>
                <input type="text" value={trn} onChange={e => setTrn(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g. 100123456700003" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Business Address</label>
                <input type="text" value={address} onChange={e => setAddress(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Dubai, United Arab Emirates" />
              </div>
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              <h2 className="text-xl font-bold text-gray-900">Invite Your Team</h2>
              <p className="text-sm text-gray-500">Add your agents by email. They'll receive an invite link to set their password.</p>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Agent Emails</label>
                <textarea value={inviteEmails} onChange={e => setInviteEmails(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-blue-500"
                  rows={4} placeholder="ahmed@example.com&#10;sara@example.com" />
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="space-y-4">
              <h2 className="text-xl font-bold text-gray-900">Connect WhatsApp</h2>
              <p className="text-sm text-gray-500">Set up your WhatsApp Business integration to start receiving and sending messages.</p>
              <ol className="list-decimal list-inside text-sm text-gray-700 space-y-2">
                <li>Go to the <strong>Meta Business Suite</strong>.</li>
                <li>Set the webhook URL to: <code className="bg-gray-100 px-2 py-0.5 rounded text-xs">{typeof window !== 'undefined' && window.location.origin}/webhooks/whatsapp</code></li>
                <li>Configure your verify token in Settings &rarr; Integrations.</li>
              </ol>
              <p className="text-xs text-gray-400">You can always do this later in Settings &rarr; Integrations.</p>
            </div>
          )}

          {step === 3 && (
            <div className="text-center space-y-4">
              <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
                <svg className="w-8 h-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h2 className="text-xl font-bold text-gray-900">You're all set!</h2>
              <p className="text-sm text-gray-500">Your Masaar CRM workspace is ready. Start managing your pipeline and WhatsApp conversations.</p>
            </div>
          )}

          <div className="flex justify-between mt-8 pt-4 border-t border-gray-100">
            {step > 0 && step < 3 ? (
              <button onClick={() => setStep(s => s - 1)}
                className="text-sm text-gray-500 hover:text-gray-700 font-medium">
                Back
              </button>
            ) : (
              <div />
            )}
            <div className="flex gap-3">
              <button onClick={handleSkip}
                className="text-sm text-gray-400 hover:text-gray-600">
                Skip
              </button>
              <button onClick={step === 3 ? handleFinish : handleNext}
                disabled={loading}
                className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-6 py-2 transition">
                {loading ? 'Saving…' : step === 3 ? 'Go to Dashboard' : 'Continue'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
