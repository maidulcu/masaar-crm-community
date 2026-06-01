'use client'
import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { MessageTemplate } from '@/types'

interface Props {
  onSelect: (body: string) => void
  onClose: () => void
}

export default function TemplatePicker({ onSelect, onClose }: Props) {
  const { lang, t } = useLang()
  const [templates, setTemplates] = useState<MessageTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')

  useEffect(() => {
    setLoading(true)
    api.messageTemplates.list({ limit: 100 })
      .then((res: any) => {
        const data: MessageTemplate[] = res?.data ?? res ?? []
        setTemplates(Array.isArray(data) ? data.filter(t => t.is_active) : [])
      })
      .catch(() => setTemplates([]))
      .finally(() => setLoading(false))
  }, [])

  const filtered = templates.filter(t =>
    t.name.toLowerCase().includes(search.toLowerCase()) ||
    t.body.toLowerCase().includes(search.toLowerCase()) ||
    t.category.toLowerCase().includes(search.toLowerCase())
  )

  const defaultCategory = t('عام', 'General')
  const grouped: Record<string, MessageTemplate[]> = {}
  filtered.forEach(tpl => {
    const cat = tpl.category || defaultCategory
    if (!grouped[cat]) grouped[cat] = []
    grouped[cat].push(tpl)
  })

  return (
    <div className="mt-3 p-4 bg-gray-50 rounded-xl border border-gray-200">
      <div className="flex items-center justify-between mb-3">
        <p className="text-xs font-semibold text-gray-600">
          {t('قوالب الرد', 'Reply Templates')}
        </p>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-sm">✕</button>
      </div>

      <input
        value={search}
        onChange={e => setSearch(e.target.value)}
        placeholder={t('بحث في القوالب...', 'Search templates...')}
        className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-brand-500"
      />

      {loading ? (
        <p className="text-xs text-gray-400 text-center py-4">{t('جاري التحميل...', 'Loading...')}</p>
      ) : Object.keys(grouped).length === 0 ? (
        <p className="text-xs text-gray-400 text-center py-4">
          {t('لا توجد قوالب', 'No templates found')}
        </p>
      ) : (
        <div className="max-h-60 overflow-y-auto space-y-3">
          {Object.entries(grouped).map(([category, items]) => (
            <div key={category}>
              <p className="text-[10px] uppercase tracking-wide text-gray-400 font-semibold mb-1 px-1">
                {category}
              </p>
              <div className="space-y-1">
                {items.map(tpl => (
                  <button
                    key={tpl.id}
                    onClick={() => onSelect(tpl.body)}
                    className="w-full text-left px-3 py-2 rounded-lg bg-white border border-gray-200 hover:border-brand-300 hover:bg-brand-50 transition-colors"
                  >
                    <p className="text-sm font-medium text-gray-900">{tpl.name}</p>
                    <p className="text-xs text-gray-400 mt-0.5 line-clamp-2">{tpl.body}</p>
                    {tpl.variables && tpl.variables.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {tpl.variables.map(v => (
                          <span key={v} className="text-[10px] px-1.5 py-0.5 bg-gray-100 text-gray-500 rounded">
                            {'{{' + v + '}}'}
                          </span>
                        ))}
                      </div>
                    )}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
