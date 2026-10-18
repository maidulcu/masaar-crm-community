'use client'
import { useState } from 'react'
import clsx from 'clsx'
import { Header } from '@/components/layout/Header'
import TaskList from '@/components/task/TaskList'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'

type Tab = 'mine' | 'overdue' | 'done' | 'all'

export default function TasksPage() {
  const { t } = useLang()
  const { user } = useAuthStore()
  const canEdit = user?.role === 'admin' || user?.role === 'agent'
  const [tab, setTab] = useState<Tab>('mine')

  const filters: Record<Tab, { mine?: boolean; status?: string; overdue?: boolean }> = {
    mine: { mine: true, status: 'open' },
    overdue: { mine: true, overdue: true },
    done: { mine: true, status: 'done' },
    all: { status: 'open' },
  }
  const tabs: { key: Tab; label: string }[] = [
    { key: 'mine', label: t('مهامي', 'My open tasks') },
    { key: 'overdue', label: t('المتأخرة', 'Overdue') },
    { key: 'done', label: t('المنجزة', 'Done') },
    { key: 'all', label: t('كل المهام المفتوحة', 'All open tasks') },
  ]

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('المهام', 'Tasks')} />
      <main className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl">
          <div role="tablist" className="flex gap-2 mb-4">
            {tabs.map(tb => (
              <button key={tb.key} role="tab" aria-selected={tab === tb.key} onClick={() => setTab(tb.key)}
                className={clsx('px-3 py-1.5 text-xs font-medium rounded-full',
                  tab === tb.key ? 'bg-primary-600 text-white' : 'bg-surface-100 text-surface-600 hover:bg-surface-200')}>
                {tb.label}
              </button>
            ))}
          </div>
          <div className="bg-white rounded-2xl border border-surface-200/70 p-5">
            <TaskList key={tab} filter={filters[tab]} canEdit={canEdit} />
          </div>
        </div>
      </main>
    </div>
  )
}
