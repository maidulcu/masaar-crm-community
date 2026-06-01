'use client'

import { useState, useEffect } from 'react'
import { getPendingCount } from '@/lib/sync-queue'
import { isOnline, onOnlineChange } from '@/lib/pwa'

export default function OfflineIndicator() {
  const [offline, setOffline] = useState(false)
  const [pendingCount, setPendingCount] = useState(0)
  const [syncing, setSyncing] = useState(false)

  useEffect(() => {
    setOffline(!isOnline())

    // Query SW for any queued items on page load
    getPendingCount().then(setPendingCount)

    const unsub = onOnlineChange((online) => {
      setOffline(!online)
      if (online) {
        setSyncing(true)
        // Clear pending count — SW will replay and send SYNC_RESULT
        setPendingCount(0)
      }
    })

    const handleSWMessage = (event: MessageEvent) => {
      if (event.data?.type === 'SYNC_RESULT') {
        setSyncing(false)
        if (event.data.synced > 0 || event.data.failed > 0) {
          setPendingCount(0)
        }
      }
      if (event.data?.type === 'QUEUED') {
        setPendingCount((c) => c + 1)
      }
    }

    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('message', handleSWMessage)
      // If SW already activated but we missed the ready event, ask for count
      if (navigator.serviceWorker.controller) {
        getPendingCount().then(setPendingCount)
      }
    }

    return () => {
      unsub()
      if ('serviceWorker' in navigator) {
        navigator.serviceWorker.removeEventListener('message', handleSWMessage)
      }
    }
  }, [])

  if (!offline && pendingCount === 0 && !syncing) return null

  return (
    <div className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 max-w-sm w-full px-4">
      {offline ? (
        <div className="bg-amber-100 border border-amber-200 text-amber-800 text-xs font-medium rounded-xl px-4 py-2.5 shadow-lg flex items-center justify-between">
          <span className="flex items-center gap-1.5">
            <svg className="w-3.5 h-3.5 shrink-0" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.083 9h1.946c.089 0 .176.036.24.1a.25.25 0 01.064.25L5.045 14h7.91l.208-1.452a.75.75 0 111.482.245l-.243 1.698c-.07.496-.496.854-.994.854H5.173a1.5 1.5 0 01-1.492-1.652l.536-5.363a.5.5 0 01.496-.445h.865L3.66 5.66a.75.75 0 111.06-1.06l13 13a.75.75 0 11-1.06 1.06L10 13.06l-3.21 3.21a.75.75 0 11-1.06-1.06l2.91-2.91-.348-2.3H4.083a1.5 1.5 0 01-1.493-1.544L2.7 7.2a.75.75 0 111.498-.05l.11 1.1c.006.06.05.1.106.1h.419l-.73-.73a.75.75 0 111.06-1.06l3.33 3.33-.088.73H4.083L3.544 9.83a.5.5 0 00.497.544h.042z" clipRule="evenodd" />
            </svg>
            You are offline — changes will sync when connected
          </span>
          <span className="text-amber-700 font-semibold">Offline</span>
        </div>
      ) : syncing || pendingCount > 0 ? (
        <div className="bg-blue-100 border border-blue-200 text-blue-800 text-xs font-medium rounded-xl px-4 py-2.5 shadow-lg flex items-center justify-between">
          <span className="flex items-center gap-1.5">
            <svg className="w-3.5 h-3.5 shrink-0 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {syncing && pendingCount === 0
              ? 'Syncing pending changes…'
              : `Syncing ${pendingCount} pending change${pendingCount !== 1 ? 's' : ''}…`}
          </span>
        </div>
      ) : null}
    </div>
  )
}
