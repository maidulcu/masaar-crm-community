'use client'
import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Expense, ExpenseCategory, RentalProperty, PaginatedResult } from '@/types'

const PAYMENT_METHODS = ['cash', 'bank_transfer', 'credit_card', 'check', 'other']

const payStatusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  paid: 'bg-green-100 text-green-700',
  refunded: 'bg-gray-100 text-gray-600',
}

export default function ExpensesPage() {
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'agent' || isAdmin

  const [expenses, setExpenses] = useState<Expense[]>([])
  const [categories, setCategories] = useState<ExpenseCategory[]>([])
  const [properties, setProperties] = useState<RentalProperty[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const limit = 20

  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({
    category_id: '', property_id: '', amount: 0, expense_date: '',
    description: '', vendor_name: '', vendor_contact: '',
    payment_method: 'bank_transfer', receipt_url: '', notes: '',
  })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => { load() }, [page])

  const load = async () => {
    setLoading(true)
    try {
      const [eRes, cRes, pRes] = await Promise.all([
        api.expenses.list({ page, limit }) as Promise<PaginatedResult<Expense>>,
        api.expenses.listCategories() as Promise<{ data: ExpenseCategory[] }>,
        api.rentalProperties.list({ limit: 100 }) as Promise<PaginatedResult<RentalProperty>>,
      ])
      setExpenses(eRes.data ?? [])
      setTotal(eRes.total ?? 0)
      setCategories(cRes.data ?? [])
      setProperties(pRes.data ?? [])
    } catch { setExpenses([]) } finally { setLoading(false) }
  }

  const set = (k: string, v: unknown) => setForm(f => ({ ...f, [k]: v }))

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setSaving(true)
    try {
      await api.expenses.create(form)
      setOpen(false)
      setForm({ category_id: '', property_id: '', amount: 0, expense_date: '', description: '', vendor_name: '', vendor_contact: '', payment_method: 'bank_transfer', receipt_url: '', notes: '' })
      load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally { setSaving(false) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm(t('حذف المصروف؟', 'Delete this expense?'))) return
    try { await api.expenses.delete(id); load() } catch {}
  }

  const fmtDate = (d: string) => new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' })
  const fmtAmount = (v: number) => `AED ${v.toLocaleString()}`
  const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('المصروفات', 'Expenses')} />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">{t('المصروفات', 'Expenses')} ({total})</h2>
          {isAgent && (
            <button onClick={() => setOpen(true)}
              className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
              + {t('مصروف جديد', 'New Expense')}
            </button>
          )}
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : expenses.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12 text-center">
            <p className="text-gray-400 text-sm mb-3">{t('لا توجد مصروفات', 'No expenses yet')}</p>
            {isAgent && <button onClick={() => setOpen(true)} className="text-brand-600 text-sm font-medium hover:underline">+ {t('أضف أول مصروف', 'Add your first expense')}</button>}
          </div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  {[t('التاريخ', 'Date'), t('الوصف', 'Description'), t('الفئة', 'Category'), t('المورد', 'Vendor'), t('المبلغ', 'Amount'), t('الحالة', 'Status')].map(h => (
                    <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                  ))}
                  {isAdmin && <th className="px-4 py-3 text-right">{t('إجراءات', 'Actions')}</th>}
                </tr>
              </thead>
              <tbody>
                {expenses.map(ex => (
                  <tr key={ex.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <Link href={`/expenses/${ex.id}`} className="font-medium text-gray-900 hover:text-brand-600">
                        {fmtDate(ex.expense_date)}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gray-700 max-w-[200px] truncate">{ex.description}</td>
                    <td className="px-4 py-3 text-gray-500 text-xs">{ex.category?.category_name || '-'}</td>
                    <td className="px-4 py-3 text-gray-600">{ex.vendor_name || '-'}</td>
                    <td className="px-4 py-3 font-medium text-gray-900">{fmtAmount(ex.amount)}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${payStatusColor[ex.payment_status] || ''}`}>
                        {ex.payment_status}
                      </span>
                    </td>
                    {isAdmin && (
                      <td className="px-4 py-3 text-right">
                        <button onClick={() => handleDelete(ex.id)}
                          className="text-xs text-red-500 hover:text-red-700 font-medium px-2 py-1 rounded hover:bg-red-50">
                          {t('حذف', 'Delete')}
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {total > limit && (
          <div className="flex justify-center gap-2 mt-6">
            <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">{t('السابق', 'Prev')}</button>
            <span className="flex items-center px-4 text-sm text-gray-500">{page} / {Math.ceil(total / limit)}</span>
            <button onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}
              className="px-4 py-2 border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50 text-sm">{t('التالي', 'Next')}</button>
          </div>
        )}
      </div>

      {/* Create Modal */}
      {open && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => { setOpen(false); setError('') }}>
          <div className="bg-white rounded-xl p-6 w-full max-w-lg mx-4 max-h-[85vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('مصروف جديد', 'New Expense')}</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('التاريخ', 'Date')} *</label>
                  <input type="date" required className={inputCls} value={form.expense_date} onChange={e => set('expense_date', e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('المبلغ (AED)', 'Amount (AED)')} *</label>
                  <input type="number" min={0} step={0.01} required className={inputCls} value={form.amount} onChange={e => set('amount', +e.target.value)} />
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('الفئة', 'Category')} *</label>
                <select required className={inputCls} value={form.category_id} onChange={e => set('category_id', e.target.value)}>
                  <option value="">{t('اختر فئة...', 'Select category...')}</option>
                  {categories.map(c => <option key={c.id} value={c.id}>{c.category_name}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('الوصف', 'Description')} *</label>
                <textarea required rows={2} className={inputCls} value={form.description} onChange={e => set('description', e.target.value)} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('طريقة الدفع', 'Payment Method')}</label>
                  <select className={inputCls} value={form.payment_method} onChange={e => set('payment_method', e.target.value)}>
                    {PAYMENT_METHODS.map(m => <option key={m} value={m}>{m.replace('_', ' ')}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('العقار (اختياري)', 'Property (optional)')}</label>
                  <select className={inputCls} value={form.property_id} onChange={e => set('property_id', e.target.value)}>
                    <option value="">{t('بدون', 'None')}</option>
                    {properties.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('اسم المورد', 'Vendor Name')}</label>
                  <input className={inputCls} value={form.vendor_name} onChange={e => set('vendor_name', e.target.value)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('رقم المورد', 'Vendor Contact')}</label>
                  <input className={inputCls} value={form.vendor_contact} onChange={e => set('vendor_contact', e.target.value)} />
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('رابط الإيصال', 'Receipt URL')}</label>
                <input className={inputCls} value={form.receipt_url} onChange={e => set('receipt_url', e.target.value)} placeholder="https://..." />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('ملاحظات', 'Notes')}</label>
                <textarea rows={2} className={inputCls} value={form.notes} onChange={e => set('notes', e.target.value)} />
              </div>
              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              <div className="flex gap-2 pt-2">
                <button type="button" onClick={() => { setOpen(false); setError('') }}
                  className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={saving}
                  className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
                  {saving ? t('جاري الحفظ...', 'Saving...') : t('إنشاء', 'Create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
