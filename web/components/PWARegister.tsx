'use client'

import { useEffect } from 'react'
import { setupPWAListeners } from '@/lib/pwa'

export default function PWARegister() {
  useEffect(() => {
    if (typeof window === 'undefined') return

    const cleanup: (() => void)[] = []

    if ('serviceWorker' in navigator) {
      navigator.serviceWorker
        .register('/sw.js')
        .then((reg) => {
          reg.update()

          if (reg.waiting) {
            reg.waiting.postMessage({ type: 'SKIP_WAITING' })
          }

          reg.addEventListener('updatefound', () => {
            const newSW = reg.installing
            if (newSW) {
              newSW.addEventListener('statechange', () => {
                if (newSW.state === 'installed' && navigator.serviceWorker.controller) {
                  newSW.postMessage({ type: 'SKIP_WAITING' })
                  window.location.reload()
                }
              })
            }
          })

          // Register periodic background sync (supported in Chromium)
          if ('periodicSync' in reg) {
            // @ts-expect-error periodicSync is not in TS types yet
            reg.periodicSync
              .register('masaar-update-check', {
                minInterval: 24 * 60 * 60 * 1000,
              })
              .catch(() => {})
          }
        })
        .catch(() => {})
    }

    cleanup.push(setupPWAListeners())

    return () => cleanup.forEach((fn) => fn())
  }, [])

  return null
}
