'use client'
import { useEffect, useState } from 'react'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { api } from '@/lib/api'
import type { AuditLog, PaginatedResult } from '@/types'
import clsx from 'clsx'

const entityLabels: Record<string, { en: string; ar: string }> = {
  contact: { en: 'Contact', ar: 'جهة اتصال' },
  lead: { en: 'Lead', ar: 'عميل محتمل' },
  deal: { en: 'Deal', ar: 'صفقة' },
  invoice: { en: 'Invoice', ar: 'فاتورة' },
  user: { en: 'User', ar: 'مستخدم' },
  document: { en: 'Document', ar: 'مستند' },
  document_template: { en: 'Template', ar: 'قالب' },
}

const actionLabels: Record<string, { en: string; ar: string }> = {
  create: { en: 'Create', ar: 'إنشاء' },
  update: { en: 'Update', ar: 'تحديث' },
  delete: { en: 'Delete', ar: 'حذف' },
  login: { en: 'Login', ar: 'تسجيل دخول' },
  logout: { en: 'Logout', ar: 'تسجيل خروج' },
  password_change: { en: 'Password Change', ar: 'تغيير كلمة المرور' },
  signature_request: { en: 'Signature Request', ar: 'طلب توقيع' },
}

const actionColors: Record<string, string> = {
  create: 'bg-green-100 text-green-700',
  update: 'bg-blue-100 text-blue-700',
  delete: 'bg-red-100 text-red-700',
  login: 'bg-gray-100 text-gray-700',
  logout: 'bg-gray-100 text-gray-600',
  password_change: 'bg-yellow-100 text-yellow-700',
  signature_request: 'bg-purple-100 text-purple-700',
}

export default function AuditLogPage() {
  const { t, lang } = useLang()
  const [items, setItems] = useState<AuditLog[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [entityFilter, setEntityFilter] = useState('')
  const [actionFilter, setActionFilter] = useState('')
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const limit = 50

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.auditLog.list({
        entity_type: entityFilter || undefined,
        action: actionFilter || undefined,
        page,
        limit,
      }) as PaginatedResult<AuditLog>
      setItems(res.data ?? [])
      setTotal(res.total ?? 0)
    } catch {
      setItems([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [page, entityFilter, actionFilter])

  const totalPages = Math.ceil(total / limit)

  const toggleExpand = (id: number) => {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const fmtDate = (d: string) => {
    const date = new Date(d)
    return date.toLocaleString(lang === 'ar' ? 'ar-AE' : 'en-AE', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    })
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('سجل التدقيق', 'Audit Log')} />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('سجل التدقيق', 'Audit Log')}
            <span className="ml-2 text-sm font-normal text-gray-500">({total})</span>
          </h2>
          <div className="flex gap-2">
            <select
              value={entityFilter}
              onChange={e => { setEntityFilter(e.target.value); setPage(1) }}
              className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <option value="">{t('كل الكيانات', 'All Entities')}</option>
              {Object.entries(entityLabels).map(([k, v]) => (
                <option key={k} value={k}>{lang === 'ar' ? v.ar : v.en}</option>
              ))}
            </select>
            <select
              value={actionFilter}
              onChange={e => { setActionFilter(e.target.value); setPage(1) }}
              className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-brand-500"
            >
              <option value="">{t('كل الإجراءات', 'All Actions')}</option>
              {Object.entries(actionLabels).map(([k, v]) => (
                <option key={k} value={k}>{lang === 'ar' ? v.ar : v.en}</option>
              ))}
            </select>
          </div>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : items.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
            <p className="text-gray-400 text-sm">{t('لا توجد سجلات', 'No records')}</p>
          </div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  <th className="px-4 py-3">{t('الكيان', 'Entity')}</th>
                  <th className="px-4 py-3">{t('الإجراء', 'Action')}</th>
                  <th className="px-4 py-3">{t('معرف الكيان', 'Entity ID')}</th>
                  <th className="px-4 py-3">{t('الفاعل', 'Actor')}</th>
                  <th className="px-4 py-3">{t('التاريخ', 'Date')}</th>
                  <th className="px-4 py-3">{t('التفاصيل', 'Details')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {items.map(item => {
                  const entityName = entityLabels[item.entity_type]
                  const actionName = actionLabels[item.action]
                  return (
                    <tr key={item.id} className="hover:bg-gray-50/50">
                      <td className="px-4 py-3 text-gray-700">
                        {lang === 'ar' ? (entityName?.ar ?? item.entity_type) : (entityName?.en ?? item.entity_type)}
                      </td>
                      <td className="px-4 py-3">
                        <span className={clsx('px-2 py-0.5 rounded text-[11px] font-medium', actionColors[item.action] || 'bg-gray-100 text-gray-600')}>
                          {lang === 'ar' ? (actionName?.ar ?? item.action) : (actionName?.en ?? item.action)}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <code className="text-[11px] text-gray-500 font-mono">{item.entity_id.slice(0, 8)}…</code>
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {item.actor_id ? (
                          <code className="text-[11px] font-mono">{item.actor_id.slice(0, 8)}…</code>
                        ) : (
                          <span className="text-gray-400">&mdash;</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-xs whitespace-nowrap">{fmtDate(item.ts)}</td>
                      <td className="px-4 py-3">
                        {item.diff ? (
                          <button
                            onClick={() => toggleExpand(item.id)}
                            className="text-brand-600 hover:text-brand-700 text-xs font-medium"
                          >
                            {expanded.has(item.id) ? t('إخفاء', 'Hide') : t('عرض', 'View')}
                          </button>
                        ) : (
                          <span className="text-gray-400">&mdash;</span>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}

        {totalPages > 1 && (
          <div className="flex items-center justify-center gap-2 mt-6">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50 disabled:opacity-40"
            >
              {t('السابق', 'Prev')}
            </button>
            <span className="text-sm text-gray-500">
              {t(`صفحة ${page} من ${totalPages}`, `Page ${page} of ${totalPages}`)}
            </span>
            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="px-3 py-1.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50 disabled:opacity-40"
            >
              {t('التالي', 'Next')}
            </button>
          </div>
        )}
      </div>

      {expanded.size > 0 && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={() => setExpanded(new Set())}>
          <div className="bg-white rounded-xl shadow-xl max-w-lg w-full mx-4 max-h-[70vh] overflow-auto" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between px-5 py-4 border-b border-gray-200">
              <h3 className="font-semibold text-gray-900">{t('تفاصيل التغيير', 'Change Details')}</h3>
              <button onClick={() => setExpanded(new Set())} className="text-gray-400 hover:text-gray-600">&times;</button>
            </div>
            <pre className="p-5 text-xs font-mono text-gray-700 whitespace-pre-wrap break-words">
              {JSON.stringify(items.find(i => expanded.has(i.id))?.diff, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}
