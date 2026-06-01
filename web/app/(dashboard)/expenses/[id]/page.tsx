'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Expense, ExpenseApproval } from '@/types'
import clsx from 'clsx'

const payStatusColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  paid: 'bg-green-100 text-green-700',
  refunded: 'bg-gray-100 text-gray-600',
}

const approvalColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  approved: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
}

export default function ExpenseDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [expense, setExpense] = useState<Expense | null>(null)
  const [approval, setApproval] = useState<ExpenseApproval | null>(null)
  const [loading, setLoading] = useState(true)

  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const [editOpen, setEditOpen] = useState(false)
  const [editPaymentStatus, setEditPaymentStatus] = useState('pending')
  const [editDesc, setEditDesc] = useState('')
  const [editAmount, setEditAmount] = useState('')

  const [approveOpen, setApproveOpen] = useState(false)
  const [approveComments, setApproveComments] = useState('')

  const load = async () => {
    if (!id) return
    setLoading(true)
    try {
      const r = await api.expenses.get(id) as Expense
      setExpense(r)
      setEditPaymentStatus(r.payment_status)
      setEditDesc(r.description)
      setEditAmount(String(r.amount))
    } catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const doAction = async (label: string, fn: () => Promise<unknown>) => {
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      await fn()
      setSuccess(label)
      load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('فشلت العملية', 'Action failed'))
    } finally { setActionLoading(false) }
  }

  const handleUpdate = (e: React.FormEvent) => {
    e.preventDefault()
    const amount = parseFloat(editAmount)
    if (isNaN(amount) || amount <= 0) return
    doAction(t('تم التحديث', 'Updated'), () =>
      api.expenses.update(id, { amount, description: editDesc, payment_status: editPaymentStatus }))
    setEditOpen(false)
  }

  const handleApprove = (e: React.FormEvent) => {
    e.preventDefault()
    doAction(t('تم الاعتماد', 'Approved'), () =>
      api.expenses.approve(id, approveComments ? { comments: approveComments } : undefined))
    setApproveOpen(false)
  }

  const fmtDate = (d: string) => new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' })
  const fmtAmount = (v: number) => `AED ${v.toLocaleString()}`

  const labelCls = 'block text-xs font-medium text-gray-600 mb-1'
  const inputCls = 'w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 focus:border-transparent'

  if (loading) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  }
  if (!expense) {
    return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Expense not found')}</div>
  }

  const needsApproval = isAdmin && !approval

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('تفاصيل المصروف', 'Expense Details')} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/expenses" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">
          ← {t('العودة', 'Back to Expenses')}
        </Link>

        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}

        {/* Status badges */}
        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', payStatusColor[expense.payment_status])}>
            {t('الدفع:', 'Payment:')} {expense.payment_status}
          </span>
          {approval && (
            <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', approvalColor[approval.approval_status])}>
              {t('الاعتماد:', 'Approval:')} {approval.approval_status}
            </span>
          )}
        </div>

        {/* Key details */}
        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('التاريخ', 'Date')}</p>
            <p className="font-semibold text-gray-900">{fmtDate(expense.expense_date)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المبلغ', 'Amount')}</p>
            <p className="font-semibold text-gray-900">{fmtAmount(expense.amount)}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('الفئة', 'Category')}</p>
            <p className="font-medium text-gray-700">{expense.category?.category_name || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('طريقة الدفع', 'Payment Method')}</p>
            <p className="font-medium text-gray-700 capitalize">{expense.payment_method.replace('_', ' ')}</p>
          </div>
          <div className="col-span-2">
            <p className="text-xs text-gray-400 mb-1">{t('الوصف', 'Description')}</p>
            <p className="text-gray-700">{expense.description || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('المورد', 'Vendor')}</p>
            <p className="font-medium text-gray-700">{expense.vendor_name || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-1">{t('رقم المورد', 'Vendor Contact')}</p>
            <p className="font-medium text-gray-700">{expense.vendor_contact || '-'}</p>
          </div>
        </div>

        {/* Notes & Receipt */}
        {expense.notes && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-2">{t('ملاحظات', 'Notes')}</h3>
            <p className="text-sm text-gray-700 whitespace-pre-wrap">{expense.notes}</p>
          </div>
        )}
        {expense.receipt_url && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-2">{t('الإيصال', 'Receipt')}</h3>
            <a href={expense.receipt_url} target="_blank" rel="noopener noreferrer"
              className="text-sm text-brand-600 hover:underline font-medium">
              {t('عرض الإيصال', 'View Receipt →')}
            </a>
          </div>
        )}

        {/* Approval info */}
        {approval && (
          <div className="bg-white rounded-xl border border-gray-100 p-5">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('الاعتماد', 'Approval')}</h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-xs text-gray-400">{t('الحالة', 'Status')}</p>
                <span className={clsx('inline-block mt-1 px-2 py-0.5 rounded text-xs font-medium capitalize', approvalColor[approval.approval_status])}>{approval.approval_status}</span>
              </div>
              {approval.approval_date && (
                <div>
                  <p className="text-xs text-gray-400">{t('تاريخ الاعتماد', 'Approval Date')}</p>
                  <p className="font-medium text-gray-700 mt-1">{fmtDate(approval.approval_date)}</p>
                </div>
              )}
              {approval.approval_comments && (
                <div className="col-span-2">
                  <p className="text-xs text-gray-400">{t('ملاحظات', 'Comments')}</p>
                  <p className="mt-1 text-gray-700">{approval.approval_comments}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-3">
          {isAdmin && (
            <>
              <button onClick={() => setEditOpen(true)}
                className="px-5 py-2.5 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
                {t('تحديث المصروف', 'Edit Expense')}
              </button>
              {needsApproval && (
                <button onClick={() => setApproveOpen(true)}
                  className="px-5 py-2.5 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 transition-colors">
                  {t('اعتماد', 'Approve')}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {editOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setEditOpen(false)}>
          <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('تحديث المصروف', 'Edit Expense')}</h3>
            <form onSubmit={handleUpdate} className="space-y-4">
              <div>
                <label className={labelCls}>{t('المبلغ (AED)', 'Amount (AED)')}</label>
                <input type="number" min={0} step={0.01} required className={inputCls} value={editAmount} onChange={e => setEditAmount(e.target.value)} />
              </div>
              <div>
                <label className={labelCls}>{t('الوصف', 'Description')}</label>
                <textarea rows={2} className={inputCls} value={editDesc} onChange={e => setEditDesc(e.target.value)} />
              </div>
              <div>
                <label className={labelCls}>{t('حالة الدفع', 'Payment Status')}</label>
                <select className={inputCls} value={editPaymentStatus} onChange={e => setEditPaymentStatus(e.target.value)}>
                  <option value="pending">{t('معلق', 'Pending')}</option>
                  <option value="paid">{t('مدفوع', 'Paid')}</option>
                  <option value="refunded">{t('مسترجع', 'Refunded')}</option>
                </select>
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setEditOpen(false)}
                  className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={actionLoading}
                  className="flex-1 py-2.5 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700">
                  {actionLoading ? '...' : t('حفظ', 'Save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Approve Modal */}
      {approveOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setApproveOpen(false)}>
          <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('اعتماد المصروف', 'Approve Expense')}</h3>
            <p className="text-xs text-gray-500 mb-4">{t('تأكيد اعتماد هذا المصروف؟', 'Confirm approval for this expense?')}</p>
            <form onSubmit={handleApprove} className="space-y-4">
              <div>
                <label className={labelCls}>{t('ملاحظات (اختياري)', 'Comments (optional)')}</label>
                <textarea rows={2} className={inputCls} value={approveComments} onChange={e => setApproveComments(e.target.value)} />
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => setApproveOpen(false)}
                  className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={actionLoading}
                  className="flex-1 py-2.5 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700">
                  {actionLoading ? '...' : t('اعتماد', 'Approve')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
