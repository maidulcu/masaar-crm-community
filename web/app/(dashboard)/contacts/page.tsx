'use client'
import { useEffect, useState, useCallback } from 'react'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { Pagination } from '@/components/ui/Pagination'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import type { Contact, PaginatedResult } from '@/types'
import clsx from 'clsx'

export default function ContactsPage() {
  const [result, setResult] = useState<PaginatedResult<Contact> | null>(null)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [showAddModal, setShowAddModal] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const { lang, t } = useLang()

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.contacts.list({ search, page, limit: 20 }) as PaginatedResult<Contact>
      setResult(data)
    } catch {
      setResult(null)
    } finally {
      setLoading(false)
    }
  }, [search, page])

  useEffect(() => { load() }, [load])

  const handleSearch = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setPage(1)
    load()
  }

  const handleAddContact = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setSubmitError('')
    setSubmitting(true)
    
    const formData = new FormData(e.currentTarget)
    const data = {
      phone_wa: formData.get('phone_wa'),
      full_name: formData.get('full_name'),
      email: formData.get('email') || undefined,
      language: formData.get('language') || 'en',
    }

    try {
      await api.contacts.create(data)
      setShowAddModal(false)
      load()
    } catch (err: any) {
      setSubmitError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally {
      setSubmitting(false)
    }
  }

  const scoreColor = (score: number) => {
    if (score >= 80) return 'text-emerald-700 bg-emerald-50 ring-1 ring-emerald-100'
    if (score >= 50) return 'text-gold-700 bg-gold-50 ring-1 ring-gold-100'
    return 'text-surface-600 bg-surface-100 ring-1 ring-surface-200'
  }

  const contacts = result?.data ?? []
  const total = result?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('جهات الاتصال', 'Contacts')} />

      <div className="flex-1 overflow-auto bg-surface-50">
        <div className="max-w-6xl mx-auto px-6 py-8">

          {/* Search bar + Add button */}
          <div className="flex gap-2 mb-6">
            <form onSubmit={handleSearch} className="flex-1 flex gap-2">
              <div className="relative flex-1">
                <span className="pointer-events-none absolute inset-y-0 start-0 flex items-center ps-3.5 text-surface-400">
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.75} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </span>
                <input
                  type="search"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t('ابحث بالاسم أو الهاتف أو البريد...', 'Search by name, phone, or email...')}
                  className="w-full ps-10 pe-4 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
                />
              </div>
              <button
                type="submit"
                className="px-4 py-2.5 bg-white border border-surface-200 text-surface-700 text-sm font-medium rounded-xl hover:bg-surface-50 transition-colors"
              >
                {t('بحث', 'Search')}
              </button>
            </form>
            <button
              onClick={() => setShowAddModal(true)}
              className="inline-flex items-center gap-1.5 px-4 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all duration-200 ease-soft"
            >
              <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fillRule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clipRule="evenodd" />
              </svg>
              {t('إضافة', 'Add')}
            </button>
          </div>

          {/* Stats */}
          <p className="text-xs text-surface-500 mb-4 font-medium">
            {total.toLocaleString()} {t('جهة اتصال', 'contacts')}
          </p>

          {/* Table */}
          {loading ? (
            <div className="text-center py-20 text-sm text-surface-400">
              {t('جاري التحميل...', 'Loading...')}
            </div>
          ) : contacts.length === 0 ? (
            <div className="text-center py-20 text-sm text-surface-400 bg-white border border-surface-200/70 rounded-2xl shadow-card">
              {t('لا توجد نتائج', 'No results found')}
            </div>
          ) : (
            <div className="bg-white rounded-2xl border border-surface-200/70 shadow-card overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-surface-200/70 bg-surface-50/60 text-surface-500 text-[11px] font-semibold uppercase tracking-wide">
                    <th className="text-start px-5 py-3.5">{t('الاسم', 'Name')}</th>
                    <th className="text-start px-5 py-3.5">{t('واتساب', 'WhatsApp')}</th>
                    <th className="text-start px-5 py-3.5">{t('البريد', 'Email')}</th>
                    <th className="text-start px-5 py-3.5">{t('اللغة', 'Lang')}</th>
                    <th className="text-start px-5 py-3.5">{t('النقاط', 'Score')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-surface-100">
                  {contacts.map((c) => (
                    <tr key={c.id} className="hover:bg-surface-50/60 transition-colors cursor-pointer" onClick={() => window.location.href = `/contacts/${c.id}`}>
                      <td className="px-5 py-3.5">
                        <Link href={`/contacts/${c.id}`} className="flex items-center gap-2.5">
                          <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary-100 to-primary-200 text-primary-700 text-xs font-semibold flex items-center justify-center shrink-0 ring-1 ring-primary-200/50">
                            {c.full_name[0]?.toUpperCase()}
                          </div>
                          <span className="font-medium text-surface-900 truncate max-w-[180px]">{c.full_name}</span>
                        </Link>
                      </td>
                      <td className="px-5 py-3.5 text-surface-600 font-mono text-xs tabular-nums">{c.phone_wa}</td>
                      <td className="px-5 py-3.5 text-surface-600 truncate max-w-[200px]">{c.email || '—'}</td>
                      <td className="px-5 py-3.5">
                        <span className="text-[11px] font-medium px-2 py-0.5 bg-surface-100 text-surface-600 rounded-md">
                          {c.language.toUpperCase()}
                        </span>
                      </td>
                      <td className="px-5 py-3.5">
                        <span className={clsx('text-[11px] font-semibold px-2 py-0.5 rounded-full tabular-nums', scoreColor(c.lead_score))}>
                          {c.lead_score}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        </div>
      </div>

      {/* Add Contact Modal */}
      <Modal
        open={showAddModal}
        onClose={() => setShowAddModal(false)}
        title={t('إضافة جهة اتصال', 'Add Contact')}
      >
        <form onSubmit={handleAddContact} className="space-y-4">
          <FormField label={t('رقم الواتساب *', 'WhatsApp Number *')}>
            <input
              type="tel"
              name="phone_wa"
              required
              placeholder={t('971501234567', '971501234567')}
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            />
          </FormField>
          
          <FormField label={t('الاسم الكامل *', 'Full Name *')}>
            <input
              type="text"
              name="full_name"
              required
              placeholder={t('أدخل الاسم الكامل', 'Enter full name')}
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            />
          </FormField>
          
          <FormField label={t('البريد الإلكتروني', 'Email')}>
            <input
              type="email"
              name="email"
              placeholder={t('example@email.com', 'example@email.com')}
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            />
          </FormField>
          
          <FormField label={t('اللغة', 'Language')}>
            <select
              name="language"
              defaultValue="en"
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            >
              <option value="en">English</option>
              <option value="ar">العربية</option>
            </select>
          </FormField>

          {submitError && <FormError error={submitError} />}

          <div className="flex gap-2 pt-2">
            <button
              type="button"
              onClick={() => setShowAddModal(false)}
              className="flex-1 px-4 py-2.5 border border-surface-200 text-surface-700 text-sm font-medium rounded-xl bg-white hover:bg-surface-50 transition-colors"
            >
              {t('إلغاء', 'Cancel')}
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="flex-1 px-4 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all disabled:opacity-60"
            >
              {submitting ? t('جاري الإضافة...', 'Adding...') : t('إضافة', 'Add')}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
