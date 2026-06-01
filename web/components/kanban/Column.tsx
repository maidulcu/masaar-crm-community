'use client'
import { useDroppable } from '@dnd-kit/core'
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable'
import type { Lead, LeadStage } from '@/types'
import { KanbanCard } from './Card'
import { useLang } from '@/context/LangContext'
import clsx from 'clsx'

const stageConfig: Record<LeadStage, { label: { en: string; ar: string }; color: string; dot: string }> = {
  new:       { label: { en: 'New',       ar: 'جديد'      }, color: 'bg-surface-100 text-surface-700',   dot: 'bg-surface-400' },
  contacted: { label: { en: 'Contacted', ar: 'تم التواصل' }, color: 'bg-sky-50 text-sky-700',            dot: 'bg-sky-500' },
  qualified: { label: { en: 'Qualified', ar: 'مؤهل'      }, color: 'bg-primary-50 text-primary-700',    dot: 'bg-primary-500' },
  proposal:  { label: { en: 'Proposal',  ar: 'عرض'       }, color: 'bg-gold-50 text-gold-700',          dot: 'bg-gold-500' },
  won:       { label: { en: 'Won',       ar: 'مكسب'      }, color: 'bg-emerald-50 text-emerald-700',    dot: 'bg-emerald-500' },
  lost:      { label: { en: 'Lost',      ar: 'خسارة'     }, color: 'bg-red-50 text-red-600',            dot: 'bg-red-500' },
}

interface Props {
  stage: string
  leads: Lead[]
  onOpenLead?: (lead: Lead) => void
  stageName?: string
  stageColor?: string
}

export function KanbanColumn({ stage, leads, onOpenLead, stageName, stageColor }: Props) {
  const { setNodeRef, isOver } = useDroppable({ id: stage })
  const { lang, t } = useLang()
  const config = stageConfig[stage as LeadStage]

  const totalValue = leads.reduce((sum, l) => sum + l.deal_value, 0)
  const currency = leads[0]?.currency ?? 'AED'

  return (
    <div className="flex flex-col w-72 shrink-0">
      {/* Column header */}
      <div className="flex items-center justify-between mb-3 px-1.5">
        <div className="flex items-center gap-2">
          {config ? (
            <span className={clsx('inline-flex items-center gap-1.5 text-[11px] font-semibold px-2 py-1 rounded-full', config.color)}>
              <span className={clsx('w-1.5 h-1.5 rounded-full', config.dot)} aria-hidden="true" />
              {lang === 'ar' ? config.label.ar : config.label.en}
            </span>
          ) : (
            <span
              className="inline-flex items-center gap-1.5 text-[11px] font-semibold px-2 py-1 rounded-full"
              style={{ backgroundColor: stageColor ? `${stageColor}20` : '#f1f5f9', color: stageColor || '#64748b' }}
            >
              <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: stageColor || '#94a3b8' }} aria-hidden="true" />
              {stageName || stage}
            </span>
          )}
          <span className="inline-flex items-center justify-center min-w-[20px] h-5 px-1.5 rounded-md text-[11px] text-surface-500 font-medium bg-surface-100">
            {leads.length}
          </span>
        </div>
        {totalValue > 0 && (
          <span className="text-[11px] text-surface-500 tabular-nums font-medium">{currency} {totalValue.toLocaleString()}</span>
        )}
      </div>

      {/* Drop zone */}
      <div
        ref={setNodeRef}
        className={clsx(
          'flex-1 min-h-32 rounded-2xl p-2 space-y-2 transition-all duration-200 ease-soft border',
          isOver
            ? 'bg-primary-50/60 border-primary-200 ring-1 ring-primary-200'
            : 'bg-surface-100/70 border-surface-200/50'
        )}
      >
        <SortableContext items={leads.map((l) => l.id)} strategy={verticalListSortingStrategy}>
          {leads.map((lead) => (
            <KanbanCard key={lead.id} lead={lead} onOpen={onOpenLead} />
          ))}
        </SortableContext>

        {leads.length === 0 && (
          <div className="flex items-center justify-center h-24 text-xs text-surface-400 font-medium">
            {t('أفلت هنا', 'Drop here')}
          </div>
        )}
      </div>
    </div>
  )
}
