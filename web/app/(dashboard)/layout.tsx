'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { Sidebar } from '@/components/layout/Sidebar'
import OfflineIndicator from '@/components/OfflineIndicator'
import DemoBanner from '@/components/layout/DemoBanner'
import DemoUpgradeModal from '@/components/layout/DemoUpgradeModal'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { token, ready, init } = useAuthStore()
  const router = useRouter()

  useEffect(() => { init() }, [init])
  useEffect(() => {
    // Redirect only once the session restore has finished; before that the token is null on
    // every reload because it lives in memory.
    if (ready && token === null) router.replace('/login')
  }, [ready, token, router])

  // Don't mount pages (and their data fetches) until we know who the user is.
  if (!ready || token === null) {
    return <div className="min-h-screen bg-surface-50" aria-busy="true" />
  }

  return (
    <div className="flex min-h-screen bg-surface-50">
      <Sidebar />
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <DemoBanner />
        {children}
      </div>
      <OfflineIndicator />
      <DemoUpgradeModal />
    </div>
  )
}
