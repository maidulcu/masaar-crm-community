import type { WSEvent } from '@/types'
import { getToken } from '@/lib/auth'

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'

type Handler = (event: WSEvent) => void

/**
 * Opens the notification socket. The server identifies the user from the JWT,
 * so only the (short-lived) access token is sent — browsers cannot set an
 * Authorization header on WebSocket connections, hence the query parameter.
 */
export function createNotificationSocket(onEvent: Handler): () => void {
  const token = getToken()
  if (!token) return () => {}
  const ws = new WebSocket(`${WS_BASE}/ws/notifications?token=${encodeURIComponent(token)}`)

  ws.onmessage = (e) => {
    try {
      const event: WSEvent = JSON.parse(e.data)
      onEvent(event)
    } catch {
      // ignore malformed frames
    }
  }

  ws.onerror = () => {
    // silently reconnect — handled by hook
  }

  return () => ws.close()
}
