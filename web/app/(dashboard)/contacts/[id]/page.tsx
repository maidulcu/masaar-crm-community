'use client'
import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Header } from '@/components/layout/Header'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import EmailComposeModal from '@/components/communication/EmailComposeModal'
import type { Contact, Lead, WhatsAppThread, User } from '@/types'
import clsx from 'clsx'

const stageLabels: Record<string, { en: string; ar: string }> = {
  new: { en: 'New', ar: 'جديد' },
  contacted: { en: 'Contacted', ar: 'تم التواصل' },
  qualified: { en: 'Qualified', ar: 'مؤهل' },
  proposal: { en: 'Proposal', ar: 'عرض' },
  won: { en: 'Won', ar: 'فائز' },
  lost: { en: 'Lost', ar: 'خاسر' },
}

const stageColors: Record<string, string> = {
  new: 'bg-blue-100 text-blue-700',
  contacted: 'bg-yellow-100 text-yellow-700',
  qualified: 'bg-purple-100 text-purple-700',
  proposal: 'bg-indigo-100 text-indigo-700',
  won: 'bg-green-100 text-green-700',
  lost: 'bg-red-100 text-red-700',
}

const threadStatusColors: Record<string, string> = {
  open: 'bg-green-100 text-green-700',
  pending: 'bg-yellow-100 text-yellow-700',
  closed: 'bg-gray-100 text-gray-600',
}

export default function ContactDetailPage() {
  const { id } = useParams<{ id: string }>()
  const router = useRouter()
  const { t, lang } = useLang()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'agent' || isAdmin

  const [contact, setContact] = useState<Contact | null>(null)
  const [leads, setLeads] = useState<Lead[]>([])
  const [threads, setThreads] = useState<WhatsAppThread[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [editOpen, setEditOpen] = useState(false)
  const [form, setForm] = useState({ full_name: '', email: '', language: 'en' as 'ar' | 'en', lead_score: 0, assigned_to: '' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [emailOpen, setEmailOpen] = useState(false)
  const [scoring, setScoring] = useState(false)
  const [scoreResult, setScoreResult] = useState<{ score: number; reasoning: string } | null>(null)

  const load = async () => {
    if (!id) return; setLoading(true)
    try {
      const [c, l, t, u] = await Promise.all([
        api.contacts.get(id) as Promise<Contact>,
        api.leads.search({ contact_id: id }) as Promise<Lead[]>,
        api.threads.list({ contact_id: id }) as Promise<WhatsAppThread[]>,
        api.users.list() as Promise<User[]>,
      ])
      setContact(c)
      setLeads(l ?? [])
      setThreads(t ?? [])
      setUsers(u ?? [])
    } catch {
      setContact(null)
    } finally { setLoading(false) }
  }

  useEffect(() => { load() }, [id])

  const openEdit = () => {
    if (!contact) return
    setForm({
      full_name: contact.full_name,
      email: contact.email || '',
      language: contact.language,
      lead_score: contact.lead_score,
      assigned_to: contact.assigned_to || '',
    })
    setEditOpen(true)
    setError('')
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setSaving(true)
    try {
      await api.contacts.update(id, {
        full_name: form.full_name,
        email: form.email || undefined,
        language: form.language,
        lead_score: form.lead_score,
        assigned_to: form.assigned_to || null,
      })
      setSuccess(t('تم الحفظ', 'Saved'))
      setEditOpen(false); load()
      setTimeout(() => setSuccess(''), 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('حدث خطأ', 'Error'))
    } finally { setSaving(false) }
  }

  const handleDelete = async () => {
    if (!confirm(t('حذف جهة الاتصال؟', 'Delete this contact?'))) return
    try { await api.contacts.delete(id); router.push('/contacts') }
    catch { setError(t('فشل الحذف', 'Delete failed')) }
  }

  const assignedUser = contact?.assigned_to ? users.find(u => u.id === contact.assigned_to) : null

  const fmtDate = (d: string) => new Date(d).toLocaleDateString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric' })
  const fmtDateTime = (d: string) => new Date(d).toLocaleString(lang === 'ar' ? 'ar-AE' : 'en-AE', { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  if (loading) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
  if (!contact) return <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">{t('غير موجود', 'Contact not found')}</div>

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={contact.full_name} />
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <div className="flex items-center justify-between">
          <Link href="/contacts" className="text-xs text-surface-500 hover:text-surface-700 font-medium flex items-center gap-1">← {t('العودة إلى جهات الاتصال', 'Back to Contacts')}</Link>
          <div className="flex gap-2">
            {isAgent && <button onClick={openEdit} className="px-4 py-2 border border-surface-200 text-surface-700 text-sm font-medium rounded-lg hover:bg-surface-50">{t('تعديل', 'Edit')}</button>}
            {isAdmin && <button onClick={handleDelete} className="px-4 py-2 bg-red-500 text-white text-sm font-medium rounded-lg hover:bg-red-600">{t('حذف', 'Delete')}</button>}
          </div>
        </div>
        {success && <p className="text-xs text-green-600 bg-green-50 p-3 rounded-lg">{success}</p>}
        {error && <p className="text-xs text-red-500 bg-red-50 p-3 rounded-lg">{error}</p>}

        {/* Contact Info */}
        <div className="bg-white rounded-xl border border-surface-200/70 p-5 grid grid-cols-2 md:grid-cols-4 gap-5">
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('الاسم', 'Name')}</p>
            <p className="font-semibold text-surface-900">{contact.full_name}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('واتساب', 'WhatsApp')}</p>
            <p className="font-medium text-surface-700 font-mono text-sm">{contact.phone_wa}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('البريد', 'Email')}</p>
            <div className="flex items-center gap-2">
              <p className="font-medium text-surface-700">{contact.email || '—'}</p>
              {contact.email && (
                <button onClick={() => setEmailOpen(true)}
                  className="text-[11px] px-2 py-0.5 bg-primary-600 text-white rounded-md hover:bg-primary-700 transition-colors font-medium">
                  {t('إرسال', 'Send')}
                </button>
              )}
            </div>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('اللغة', 'Language')}</p>
            <span className="text-[11px] font-medium px-2 py-0.5 bg-surface-100 text-surface-600 rounded-md">{contact.language.toUpperCase()}</span>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('نقاط العميل', 'Lead Score')}</p>
            <div className="flex items-center gap-2">
              <span className={clsx('text-xs font-semibold px-2 py-0.5 rounded-full', contact.lead_score >= 80 ? 'bg-green-100 text-green-700' : contact.lead_score >= 50 ? 'bg-yellow-100 text-yellow-700' : 'bg-surface-100 text-surface-600')}>
                {contact.lead_score}
              </span>
              {isAgent && leads.length > 0 && (
                <button
                  disabled={scoring}
                  onClick={async () => {
                    setScoring(true)
                    setScoreResult(null)
                    try {
                      const res: any = await api.ai.scoreContact(id)
                      if (res?.score != null) {
                        setScoreResult({ score: res.score, reasoning: res.reasoning })
                        setContact(c => c ? { ...c, lead_score: res.score } : c)
                      }
                    } catch {} finally { setScoring(false) }
                  }}
                  className="text-[11px] px-2 py-0.5 bg-gold-100 text-gold-700 rounded-full font-medium hover:bg-gold-200 disabled:opacity-40 transition-colors"
                  title={t('تقييم بواسطة AI', 'Score with AI')}
                >
                  {scoring ? '...' : '🤖 AI'}
                </button>
              )}
            </div>
            {scoreResult && (
              <p className="text-[11px] text-surface-500 mt-1.5 leading-relaxed max-w-xs">{scoreResult.reasoning}</p>
            )}
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('المسؤول', 'Assigned To')}</p>
            <p className="font-medium text-surface-700">{assignedUser?.name || '—'}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('تاريخ الإضافة', 'Created')}</p>
            <p className="font-medium text-surface-700 text-sm">{fmtDateTime(contact.created_at)}</p>
          </div>
          <div>
            <p className="text-xs text-surface-400 mb-1">{t('آخر تحديث', 'Updated')}</p>
            <p className="font-medium text-surface-700 text-sm">{fmtDateTime(contact.updated_at)}</p>
          </div>
        </div>

        {/* Leads */}
        <div className="bg-white rounded-xl border border-surface-200/70 overflow-hidden">
          <div className="px-5 py-3.5 border-b border-surface-200/70 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-surface-900">{t('العملاء المحتملون', 'Leads')}</h3>
            <span className="text-xs text-surface-500">{leads.length}</span>
          </div>
          {leads.length === 0 ? (
            <p className="px-5 py-8 text-sm text-surface-400 text-center">{t('لا يوجد عملاء محتملون', 'No leads')}</p>
          ) : (
            <div className="divide-y divide-surface-100">
              {leads.map(l => (
                <Link key={l.id} href={`/pipeline?lead=${l.id}`} className="flex items-center gap-3 px-5 py-3 hover:bg-surface-50 transition-colors">
                  <span className={clsx('px-2 py-0.5 rounded text-[10px] font-medium', stageColors[l.stage])}>
                    {lang === 'ar' ? (stageLabels[l.stage]?.ar || l.stage) : (stageLabels[l.stage]?.en || l.stage)}
                  </span>
                  <span className="flex-1 text-sm text-surface-700">{l.notes ? l.notes.substring(0, 80) + (l.notes.length > 80 ? '...' : '') : t('بدون ملاحظات', 'No notes')}</span>
                  {l.deal_value > 0 && <span className="text-sm font-medium text-surface-900">AED {l.deal_value.toLocaleString()}</span>}
                  <span className="text-[10px] text-surface-400">{l.created_at ? fmtDate(l.created_at) : ''}</span>
                </Link>
              ))}
            </div>
          )}
        </div>

        {/* WhatsApp Threads */}
        <div className="bg-white rounded-xl border border-surface-200/70 overflow-hidden">
          <div className="px-5 py-3.5 border-b border-surface-200/70 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-surface-900">{t('محادثات واتساب', 'WhatsApp Threads')}</h3>
            <span className="text-xs text-surface-500">{threads.length}</span>
          </div>
          {threads.length === 0 ? (
            <p className="px-5 py-8 text-sm text-surface-400 text-center">{t('لا توجد محادثات', 'No threads')}</p>
          ) : (
            <div className="divide-y divide-surface-100">
              {threads.map(th => (
                <Link key={th.id} href={`/inbox/${th.id}`} className="flex items-center gap-3 px-5 py-3 hover:bg-surface-50 transition-colors">
                  <span className={clsx('px-2 py-0.5 rounded text-[10px] font-medium', threadStatusColors[th.thread_status])}>
                    {th.thread_status}
                  </span>
                  <span className="flex-1 text-sm text-surface-700">
                    {th.message_count} {t('رسالة', 'messages')}
                    {th.ai_summary && <span className="text-surface-400 ml-2">· {th.ai_summary.substring(0, 60)}</span>}
                  </span>
                  <span className="text-[10px] text-surface-400">{th.last_message_at ? fmtDateTime(th.last_message_at) : ''}</span>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {editOpen && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => { setEditOpen(false); setError('') }}>
          <div className="bg-white rounded-xl p-6 w-full max-w-md mx-4" onClick={e => e.stopPropagation()}>
            <h3 className="text-sm font-semibold text-surface-900 mb-4">{t('تعديل جهة الاتصال', 'Edit Contact')}</h3>
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('الاسم', 'Name')} *</label>
                <input required className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={form.full_name} onChange={e => setForm(f => ({ ...f, full_name: e.target.value }))} />
              </div>
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('البريد', 'Email')}</label>
                <input type="email" className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={form.email} onChange={e => setForm(f => ({ ...f, email: e.target.value }))} />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-surface-600 mb-1">{t('اللغة', 'Language')}</label>
                  <select className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={form.language} onChange={e => setForm(f => ({ ...f, language: e.target.value as 'ar' | 'en' }))}>
                    <option value="en">English</option>
                    <option value="ar">العربية</option>
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-600 mb-1">{t('النقاط', 'Score')}</label>
                  <input type="number" min={0} max={100} className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={form.lead_score} onChange={e => setForm(f => ({ ...f, lead_score: +e.target.value }))} />
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-surface-600 mb-1">{t('المسؤول', 'Assigned To')}</label>
                <select className="w-full px-3 py-2 border border-surface-200 rounded-lg text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40" value={form.assigned_to} onChange={e => setForm(f => ({ ...f, assigned_to: e.target.value }))}>
                  <option value="">{t('بدون', 'None')}</option>
                  {users.map(u => <option key={u.id} value={u.id}>{u.name}</option>)}
                </select>
              </div>
              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              <div className="flex gap-2 pt-2">
                <button type="button" onClick={() => { setEditOpen(false); setError('') }} className="flex-1 py-2 border border-surface-200 rounded-lg text-sm text-surface-600 hover:bg-surface-50">{t('إلغاء', 'Cancel')}</button>
                <button type="submit" disabled={saving} className="flex-1 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:opacity-60">
                  {saving ? t('جاري الحفظ...', 'Saving...') : t('حفظ', 'Save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <EmailComposeModal
        open={emailOpen}
        onClose={() => setEmailOpen(false)}
        toEmail={contact?.email}
        toName={contact?.full_name}
        relatedTo="contact"
        relatedId={contact?.id}
      />
    </div>
  )
}
