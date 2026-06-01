'use client'
import { useEffect, useRef, useState } from 'react'
import { createNotificationSocket } from '@/lib/ws'
import type { Notification, WSEvent } from '@/types'
import { api } from '@/lib/api'

export function useNotifications(userId: string | null) {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [unread, setUnread] = useState(0)
  const closeRef = useRef<(() => void) | null>(null)

  // Load initial notifications
  useEffect(() => {
    if (!userId) return
    api.notifications.list({ page: 1 }).then((res: any) => {
      const items: Notification[] = res?.data ?? []
      setNotifications(items)
      setUnread(items.filter((n) => !n.read).length)
    }).catch(() => {})
  }, [userId])

  // WebSocket for real-time
  useEffect(() => {
    if (!userId) return

    const close = createNotificationSocket(userId, (event: WSEvent) => {
      if (event.type === 'notification') {
        const n = event.payload as Notification
        setNotifications((prev) => [n, ...prev])
        setUnread((c) => c + 1)
      }
    })
    closeRef.current = close
    return () => close()
  }, [userId])

  const markRead = async (id: string) => {
    await api.notifications.markRead(id).catch(() => {})
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read: true } : n))
    )
    setUnread((c) => Math.max(0, c - 1))
  }

  const markAllRead = async () => {
    await api.notifications.markAllRead().catch(() => {})
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
    setUnread(0)
  }

  const loadMore = async (page: number) => {
    const res: any = await api.notifications.list({ page }).catch(() => ({ data: [] }))
    return res?.data ?? []
  }

  return { notifications, unread, markRead, markAllRead, loadMore }
}
