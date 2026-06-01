import type { WSEvent } from '@/types'

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'

type Handler = (event: WSEvent) => void

export function createNotificationSocket(userId: string, onEvent: Handler): () => void {
  const url = `${WS_BASE}/ws/notifications?user=${userId}`
  const ws = new WebSocket(url)

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
