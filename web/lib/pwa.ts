import { triggerSync } from './sync-queue'

let online = typeof navigator !== 'undefined' ? navigator.onLine : true
const listeners: Set<(online: boolean) => void> = new Set()

export function isOnline(): boolean {
  return online
}

export function onOnlineChange(cb: (online: boolean) => void) {
  listeners.add(cb)
  return () => listeners.delete(cb)
}

export function setupPWAListeners() {
  if (typeof window === 'undefined') return () => {}

  const handleOnline = () => {
    online = true
    listeners.forEach((cb) => cb(true))
    triggerSync()
  }

  const handleOffline = () => {
    online = false
    listeners.forEach((cb) => cb(false))
  }

  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)

  return () => {
    window.removeEventListener('online', handleOnline)
    window.removeEventListener('offline', handleOffline)
    listeners.clear()
  }
}
