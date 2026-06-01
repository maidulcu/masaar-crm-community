'use client'
import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import { Sidebar } from '@/components/layout/Sidebar'
import OfflineIndicator from '@/components/OfflineIndicator'
import DemoBanner from '@/components/layout/DemoBanner'
import DemoUpgradeModal from '@/components/layout/DemoUpgradeModal'

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { token, init } = useAuthStore()
  const router = useRouter()

  useEffect(() => { init() }, [init])
  useEffect(() => {
    if (token === null && typeof window !== 'undefined') {
      // Only redirect after init has run (token could be null before init)
      const stored = localStorage.getItem('masaar_access_token')
      if (!stored) router.replace('/login')
    }
  }, [token, router])

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
