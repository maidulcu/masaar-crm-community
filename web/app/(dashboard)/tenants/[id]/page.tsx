'use client'
import { useEffect, useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { Tenant } from '@/types'
import clsx from 'clsx'

const statusColor: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  inactive: 'bg-gray-100 text-gray-600',
  blacklisted: 'bg-red-100 text-red-700',
}

const verificationColor: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  verified: 'bg-green-100 text-green-700',
  rejected: 'bg-red-100 text-red-700',
}

export default function TenantDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [tenant, setTenant] = useState<Tenant | null>(null)
  const [loading, setLoading] = useState(true)
  const [verifyOpen, setVerifyOpen] = useState(false)
  const [verifyNotes, setVerifyNotes] = useState('')
  const [actionLoading, setActionLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = async () => {
    if (!id) return; setLoading(true)
    try { setTenant(await api.tenants.get(id) as Tenant) }
    catch {} finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const handleVerify = async (status: string) => {
    setError(''); setSuccess(''); setActionLoading(true)
    try {
      await api.tenants.verify(id, verifyNotes)
      setSuccess(t(status === 'verified' ? 'تم التحقق' : 'تم الرفض', status === 'verified' ? 'Verified' : 'Rejected'))
      setVerifyOpen(false); load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed')
    } finally { setActionLoading(false) }
  }

  const fmtDate = (d: string | null) => d ? new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' }) : '—'
  const fmtAmount = (v: number) => v ? `AED ${v.toLocaleString()}` : '—'

  if (loading) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!tenant) return <div className="flex-1 flex items-center justify-center text-gray-400 text-sm">{t('غير موجود', 'Tenant not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={tenant.full_name_en} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <Link href="/tenants" className="text-xs text-gray-500 hover:text-gray-700 font-medium flex items-center gap-1">← {t('العودة', 'Back to Tenants')}</Link>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        <div className="flex items-center gap-3">
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', statusColor[tenant.status])}>{tenant.status}</span>
          <span className={clsx('px-3 py-1 rounded-full text-sm font-medium capitalize', verificationColor[tenant.verification_status])}>{tenant.verification_status}</span>
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div><p className="text-xs text-gray-400 mb-1">{t('الاسم', 'Name')}</p><p className="font-semibold text-gray-900">{tenant.full_name_en}</p>{tenant.full_name_ar && <p className="text-sm text-gray-500">{tenant.full_name_ar}</p>}</div>
          <div><p className="text-xs text-gray-400 mb-1">{t('البريد', 'Email')}</p><p className="font-medium text-gray-700">{tenant.email || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('الهاتف', 'Phone')}</p><p className="font-medium text-gray-700">{tenant.phone_wa || tenant.phone || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('الجنسية', 'Nationality')}</p><p className="font-medium text-gray-700">{tenant.nationality || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('نوع الهوية', 'ID Type')}</p><p className="font-medium text-gray-700 capitalize">{tenant.id_type?.replace('_', ' ') || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('رقم الهوية', 'ID Number')}</p><p className="font-medium text-gray-700">{tenant.id_number || '—'}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('انتهاء الهوية', 'ID Expiry')}</p><p className="font-medium text-gray-700">{fmtDate(tenant.id_expiry_date)}</p></div>
          <div><p className="text-xs text-gray-400 mb-1">{t('الحالة الوظيفية', 'Employment')}</p><p className="font-medium text-gray-700 capitalize">{tenant.employment_status?.replace('_', ' ') || '—'}</p></div>
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('المعلومات المالية', 'Financial')}</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-5">
            <div><p className="text-xs text-gray-400 mb-1">{t('جهة العمل', 'Employer')}</p><p className="font-medium text-gray-700">{tenant.employer_name || '—'}</p></div>
            <div><p className="text-xs text-gray-400 mb-1">{t('الدخل السنوي', 'Annual Income')}</p><p className="font-semibold text-gray-900">{fmtAmount(tenant.annual_income)}</p></div>
            <div><p className="text-xs text-gray-400 mb-1">{t('بلد المنشأ', 'Country of Origin')}</p><p className="font-medium text-gray-700">{tenant.country_of_origin || '—'}</p></div>
          </div>
        </div>

        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('معلومات الاتصال في الطوارئ', 'Emergency Contact')}</h3>
          <div className="grid grid-cols-2 gap-5">
            <div><p className="text-xs text-gray-400 mb-1">{t('الاسم', 'Name')}</p><p className="font-medium text-gray-700">{tenant.emergency_contact_name || '—'}</p></div>
            <div><p className="text-xs text-gray-400 mb-1">{t('الهاتف', 'Phone')}</p><p className="font-medium text-gray-700">{tenant.emergency_contact_phone || '—'}</p></div>
          </div>
        </div>

        {tenant.notes && <div className="bg-white rounded-xl border border-gray-100 p-5"><h3 className="text-sm font-semibold text-gray-700 mb-2">{t('ملاحظات', 'Notes')}</h3><p className="text-sm text-gray-700 whitespace-pre-wrap">{tenant.notes}</p></div>}

        <div className="flex flex-wrap gap-3">
          {isAdmin && tenant.verification_status === 'pending' && (
            <>
              <button onClick={() => setVerifyOpen(true)} className="px-5 py-2.5 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700">{t('تحقق', 'Verify')}</button>
              <button onClick={() => handleVerify('rejected')} disabled={actionLoading} className="px-5 py-2.5 bg-red-500 text-white text-sm font-medium rounded-lg hover:bg-red-600">{t('رفض', 'Reject')}</button>
            </>
          )}
          {tenant.id_document_url && (
            <a href={tenant.id_document_url} target="_blank" rel="noopener noreferrer" className="px-5 py-2.5 border border-gray-200 text-sm font-medium rounded-lg hover:bg-gray-50">{t('عرض المستند', 'View Document')}</a>
          )}
          {tenant.salary_certificate_url && (
            <a href={tenant.salary_certificate_url} target="_blank" rel="noopener noreferrer" className="px-5 py-2.5 border border-gray-200 text-sm font-medium rounded-lg hover:bg-gray-50">{t('شهادة الراتب', 'Salary Certificate')}</a>
          )}
        </div>
      </div>

      {verifyOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setVerifyOpen(false)}>
          <div className="bg-white rounded-xl p-6 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-gray-700 mb-4">{t('التحقق من المستأجر', 'Verify Tenant')}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('ملاحظات (اختياري)', 'Notes (optional)')}</label>
                <textarea rows={3} className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-500" value={verifyNotes} onChange={e => setVerifyNotes(e.target.value)} />
              </div>
              <div className="flex gap-2">
                <button onClick={() => setVerifyOpen(false)} className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">{t('إلغاء', 'Cancel')}</button>
                <button onClick={() => handleVerify('verified')} disabled={actionLoading} className="flex-1 py-2.5 bg-green-600 text-white rounded-lg text-sm font-medium hover:bg-green-700">{t('تأكيد التحقق', 'Confirm Verify')}</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
