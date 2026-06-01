'use client'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { Lead } from '@/types'
import { useLang } from '@/context/LangContext'
import clsx from 'clsx'

const sourceColors: Record<string, string> = {
  whatsapp: 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-100',
  web:      'bg-sky-50 text-sky-700 ring-1 ring-sky-100',
  referral: 'bg-violet-50 text-violet-700 ring-1 ring-violet-100',
  event:    'bg-gold-50 text-gold-700 ring-1 ring-gold-100',
}

interface Props {
  lead: Lead
  onOpen?: (lead: Lead) => void
}

export function KanbanCard({ lead, onOpen }: Props) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: lead.id })
  const { t } = useLang()

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      className={clsx(
        'bg-white rounded-xl border border-surface-200/70 shadow-card select-none transition-all duration-200 ease-soft hover:shadow-card-hover hover:border-surface-300',
        isDragging && 'opacity-60 shadow-pop ring-2 ring-primary-300'
      )}
    >
      {/* Clickable body — opens notes modal */}
      <div
        className="p-3.5 cursor-pointer"
        onClick={() => onOpen?.(lead)}
      >
        {/* Contact name */}
        <p className="font-semibold text-[13.5px] text-surface-900 truncate tracking-tight">
          {lead.contact?.full_name ?? t('جهة اتصال غير معروفة', 'Unknown contact')}
        </p>

        {/* Phone */}
        {lead.contact?.phone_wa && (
          <p className="text-[11.5px] text-surface-500 mt-0.5 truncate font-mono">{lead.contact.phone_wa}</p>
        )}

        {/* Deal value */}
        <div className="flex items-center justify-between mt-2.5">
          <span className="text-[13px] font-semibold text-surface-900 tabular-nums">
            {lead.currency} {lead.deal_value.toLocaleString()}
          </span>
          {lead.source && (
            <span className={clsx('text-[10px] font-semibold px-2 py-0.5 rounded-full capitalize', sourceColors[lead.source] ?? 'bg-surface-100 text-surface-600 ring-1 ring-surface-200')}>
              {lead.source}
            </span>
          )}
        </div>

        {/* Assigned agent */}
        {lead.assigned_user && (
          <p className="mt-2 text-[10px] text-surface-500 font-medium truncate">
            👤 {lead.assigned_user.name}
          </p>
        )}

        {/* Tags */}
        {lead.tags && lead.tags.length > 0 && (
          <div className="mt-1.5 flex flex-wrap gap-1">
            {lead.tags.slice(0, 3).map((tag) => (
              <span key={tag} className="text-[9px] px-1.5 py-0.5 bg-primary-50 text-primary-700 rounded-full font-medium">
                {tag}
              </span>
            ))}
            {lead.tags.length > 3 && (
              <span className="text-[9px] text-surface-400">+{lead.tags.length - 3}</span>
            )}
          </div>
        )}

        {/* Lead score */}
        {lead.lead_score != null && lead.lead_score > 0 && (
          <div className="mt-2 flex items-center gap-1.5">
            <div className="flex-1 h-1 bg-surface-100 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-primary-400 to-primary-600 rounded-full"
                style={{ width: `${lead.lead_score}%` }}
              />
            </div>
            <span className="text-[10px] text-surface-500 tabular-nums font-medium">{lead.lead_score}</span>
          </div>
        )}

        {/* Notes preview */}
        {lead.notes && (
          <p className="mt-2 text-[11px] text-surface-500 truncate italic">
            {lead.notes}
          </p>
        )}
      </div>

      {/* Drag handle — only this triggers DnD */}
      <div
        {...listeners}
        className="flex items-center justify-center py-1.5 border-t border-surface-100 cursor-grab active:cursor-grabbing text-surface-300 hover:text-surface-500 transition-colors"
        title={t('اسحب للنقل', 'Drag to move')}
      >
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
          <path d="M8 6a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm8 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3zM8 13.5a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm8 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3zM8 21a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm8 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3z"/>
        </svg>
      </div>
    </div>
  )
}
