'use client'

import Link from 'next/link'
import { useAuthStore } from '@/store/auth'

export default function DemoBanner() {
  const { company } = useAuthStore()

  if (!company?.is_demo) return null

  return (
    <div className="w-full bg-amber-400 text-amber-950 px-4 py-2 flex items-center justify-between gap-4 text-sm font-medium z-50 shrink-0">
      <div className="flex items-center gap-2">
        {/* warning icon */}
        <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
        </svg>
        <span>
          <strong>Demo account</strong> — data is read-only and resets daily.
          {' '}Any changes you try to make will show you the upgrade prompt.
        </span>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <a
          href="https://cal.com/masaar/demo"
          target="_blank"
          rel="noopener noreferrer"
          className="bg-amber-200 hover:bg-amber-300 text-amber-900 transition rounded px-3 py-1 text-xs font-semibold whitespace-nowrap"
        >
          Book a call
        </a>
        <Link
          href="/signup"
          className="bg-amber-950 text-amber-100 hover:bg-amber-800 transition rounded px-3 py-1 text-xs font-semibold whitespace-nowrap"
        >
          Start free trial →
        </Link>
      </div>
    </div>
  )
}
