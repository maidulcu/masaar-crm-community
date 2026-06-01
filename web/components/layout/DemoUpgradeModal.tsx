'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'

/**
 * DemoUpgradeModal — shown whenever a demo user tries a write operation.
 * Triggered by the 'demo:blocked' CustomEvent dispatched from api.ts.
 */
export default function DemoUpgradeModal() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const handler = () => setOpen(true)
    window.addEventListener('demo:blocked', handler)
    return () => window.removeEventListener('demo:blocked', handler)
  }, [])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-surface-900/40 backdrop-blur-sm"
        onClick={() => setOpen(false)}
      />

      {/* Card */}
      <div className="relative w-full max-w-sm bg-white rounded-2xl shadow-xl border border-surface-200 p-6 animate-scale-in">
        {/* Icon */}
        <div className="mx-auto mb-4 w-12 h-12 rounded-xl bg-amber-100 flex items-center justify-center">
          <svg className="w-6 h-6 text-amber-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
          </svg>
        </div>

        <h2 className="text-center text-base font-semibold text-surface-900 mb-1">
          This is a read-only demo
        </h2>
        <p className="text-center text-sm text-surface-500 mb-6">
          You&apos;re exploring a live preview. Sign up for a free trial to create leads, send WhatsApp messages, manage leases, and more.
        </p>

        <div className="flex flex-col gap-2">
          <Link
            href="/signup"
            className="w-full py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-semibold rounded-xl text-center transition-colors shadow-card"
            onClick={() => setOpen(false)}
          >
            Start free trial →
          </Link>
          <button
            onClick={() => setOpen(false)}
            className="w-full py-2.5 text-sm text-surface-500 hover:text-surface-700 transition-colors"
          >
            Keep exploring
          </button>
        </div>

        {/* Close button */}
        <button
          onClick={() => setOpen(false)}
          className="absolute top-4 right-4 text-surface-400 hover:text-surface-600 transition-colors"
          aria-label="Close"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  )
}
