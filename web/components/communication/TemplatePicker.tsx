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
        <button
          type="button"
          onClick={onClose}
          aria-label={t('إغلاق', 'Close')}
          className="inline-flex items-center justify-center w-6 h-6 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors focus:outline-none focus:ring-2 focus:ring-brand-500"
        >
          <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        </button>
      </div>

      <input
        value={search}
        onChange={e => setSearch(e.target.value)}
        placeholder={t('بحث في القوالب...', 'Search templates...')}
        aria-label={t('بحث في القوالب', 'Search templates')}
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
                    type="button"
                    onClick={() => onSelect(tpl.body)}
                    className="w-full text-start px-3 py-2 rounded-lg bg-white border border-gray-200 hover:border-brand-300 hover:bg-brand-50 transition-colors focus:outline-none focus:ring-2 focus:ring-brand-500"
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
