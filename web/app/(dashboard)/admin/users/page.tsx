'use client'
import { useEffect, useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { Header } from '@/components/layout/Header'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import { api } from '@/lib/api'
import type { User } from '@/types'

const ROLES = ['agent', 'viewer', 'admin'] as const
const inputCls = 'w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-500'

const roleBadge = (role: string) => {
  const cls = {
    admin: 'bg-purple-100 text-purple-700',
    agent: 'bg-blue-100 text-blue-700',
    viewer: 'bg-gray-100 text-gray-600',
  }[role] ?? 'bg-gray-100 text-gray-600'
  return <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${cls}`}>{role}</span>
}

export default function AdminUsersPage() {
  const { t } = useLang()
  const { user: me } = useAuthStore()
  const router = useRouter()

  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  const [showInvite, setShowInvite] = useState(false)
  const [inviteForm, setInviteForm] = useState({ name: '', email: '', role: 'agent' })
  const [inviteError, setInviteError] = useState('')
  const [inviting, setInviting] = useState(false)

  const [editTarget, setEditTarget] = useState<User | null>(null)
  const [editForm, setEditForm] = useState({ name: '', role: '' })
  const [editError, setEditError] = useState('')
  const [saving, setSaving] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (me && me.role !== 'admin') router.replace('/dashboard')
  }, [me, router])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.users.list() as User[]
      setUsers(data ?? [])
    } catch { setUsers([]) } finally { setLoading(false) }
  }, [])

  useEffect(() => { load() }, [load])

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    setInviteError('')
    setInviting(true)
    try {
      await api.users.invite(inviteForm)
      setShowInvite(false)
      setInviteForm({ name: '', email: '', role: 'agent' })
      load()
    } catch (err: any) {
      setInviteError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally { setInviting(false) }
  }

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editTarget) return
    setEditError('')
    setSaving(true)
    try {
      await api.users.update(editTarget.id, { name: editForm.name, role: editForm.role })
      setEditTarget(null)
      load()
    } catch (err: any) {
      setEditError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally { setSaving(false) }
  }

  const handleToggleActive = async (user: User) => {
    try {
      await api.users.setActive(user.id, !user.is_active)
      load()
    } catch { /* silent */ }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await api.users.delete(deleteTarget.id)
      setDeleteTarget(null)
      load()
    } catch { /* silent */ } finally { setDeleting(false) }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('إدارة الفريق', 'Team Management')} />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold text-gray-800">
            {t('المستخدمون', 'Users')} ({users.length})
          </h2>
          <button onClick={() => setShowInvite(true)} className="px-4 py-2 bg-brand-600 text-white text-sm font-medium rounded-lg hover:bg-brand-700 transition-colors">
            + {t('دعوة مستخدم', 'Invite User')}
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-400 text-sm">{t('جاري التحميل...', 'Loading...')}</div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            {users.length === 0 ? (
              <div className="p-12 text-center">
                <p className="text-gray-400 text-sm mb-3">{t('لا يوجد مستخدمون', 'No users found')}</p>
                <button onClick={() => setShowInvite(true)} className="text-brand-600 text-sm font-medium hover:underline">
                  + {t('دعوة أول مستخدم', 'Invite your first user')}
                </button>
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    {[t('الاسم', 'Name'), t('البريد الإلكتروني', 'Email'), t('الدور', 'Role'), t('الحالة', 'Status'), t('تاريخ الانضمام', 'Joined'), t('الإجراءات', 'Actions')].map(h => (
                      <th key={h} className="px-4 py-3 text-left font-medium text-gray-700">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {users.map(u => {
                    const isSelf = u.id === me?.id
                    return (
                      <tr key={u.id} className="border-b border-gray-100 hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium text-gray-900">
                          {u.name}
                          {isSelf && <span className="ml-2 text-xs text-gray-400">({t('أنت', 'you')})</span>}
                        </td>
                        <td className="px-4 py-3 text-gray-500 text-xs">{u.email}</td>
                        <td className="px-4 py-3">{roleBadge(u.role)}</td>
                        <td className="px-4 py-3">
                          <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${u.is_active ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600'}`}>
                            {u.is_active ? t('نشط', 'Active') : t('معطل', 'Inactive')}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-400 text-xs">
                          {new Date(u.created_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3">
                          {!isSelf && (
                            <div className="flex items-center gap-3">
                              <button
                                onClick={() => { setEditTarget(u); setEditForm({ name: u.name, role: u.role }) }}
                                className="text-xs text-brand-600 hover:underline"
                              >
                                {t('تعديل', 'Edit')}
                              </button>
                              <button
                                onClick={() => handleToggleActive(u)}
                                className={`text-xs ${u.is_active ? 'text-yellow-600' : 'text-green-600'} hover:underline`}
                              >
                                {u.is_active ? t('تعطيل', 'Deactivate') : t('تفعيل', 'Activate')}
                              </button>
                              <button
                                onClick={() => setDeleteTarget(u)}
                                className="text-xs text-red-500 hover:underline"
                              >
                                {t('حذف', 'Delete')}
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Invite Modal */}
      <Modal open={showInvite} onClose={() => { setShowInvite(false); setInviteError('') }} title={t('دعوة مستخدم جديد', 'Invite New User')}>
        <form onSubmit={handleInvite} className="space-y-4">
          <FormField label={t('الاسم', 'Name')}>
            <input required className={inputCls} value={inviteForm.name} onChange={e => setInviteForm(f => ({ ...f, name: e.target.value }))} placeholder={t('أحمد المنصوري', 'Ahmed Al Mansouri')} />
          </FormField>
          <FormField label={t('البريد الإلكتروني', 'Email')}>
            <input type="email" required className={inputCls} value={inviteForm.email} onChange={e => setInviteForm(f => ({ ...f, email: e.target.value }))} placeholder="ahmed@company.ae" />
          </FormField>
          <FormField label={t('الدور', 'Role')}>
            <select className={inputCls} value={inviteForm.role} onChange={e => setInviteForm(f => ({ ...f, role: e.target.value }))}>
              <option value="agent">{t('وكيل', 'Agent')}</option>
              <option value="viewer">{t('مشاهد', 'Viewer')}</option>
              <option value="admin">{t('مسؤول', 'Admin')}</option>
            </select>
          </FormField>
          <p className="text-xs text-gray-400">
            {t('سيتلقى المستخدم رابطاً لتفعيل حسابه عبر البريد الإلكتروني.', 'The user will receive an account setup link by email.')}
          </p>
          {inviteError && <FormError error={inviteError} />}
          <div className="flex gap-3 pt-1">
            <button type="button" onClick={() => setShowInvite(false)} className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
              {t('إلغاء', 'Cancel')}
            </button>
            <button type="submit" disabled={inviting} className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
              {inviting ? t('جارٍ الإرسال...', 'Sending...') : t('إرسال الدعوة', 'Send Invite')}
            </button>
          </div>
        </form>
      </Modal>

      {/* Edit Modal */}
      <Modal open={!!editTarget} onClose={() => { setEditTarget(null); setEditError('') }} title={t('تعديل المستخدم', 'Edit User')}>
        <form onSubmit={handleEdit} className="space-y-4">
          <FormField label={t('الاسم', 'Name')}>
            <input required className={inputCls} value={editForm.name} onChange={e => setEditForm(f => ({ ...f, name: e.target.value }))} />
          </FormField>
          <FormField label={t('الدور', 'Role')}>
            <select className={inputCls} value={editForm.role} onChange={e => setEditForm(f => ({ ...f, role: e.target.value }))}>
              {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
          </FormField>
          {editError && <FormError error={editError} />}
          <div className="flex gap-3 pt-1">
            <button type="button" onClick={() => setEditTarget(null)} className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
              {t('إلغاء', 'Cancel')}
            </button>
            <button type="submit" disabled={saving} className="flex-1 py-2 bg-brand-600 text-white rounded-lg text-sm font-medium hover:bg-brand-700 disabled:opacity-60">
              {saving ? t('جارٍ الحفظ...', 'Saving...') : t('حفظ', 'Save')}
            </button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirm Modal */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('تأكيد الحذف', 'Confirm Delete')}>
        <p className="text-sm text-gray-600 mb-5">
          {t(
            `هل أنت متأكد من حذف "${deleteTarget?.name}"؟ لا يمكن التراجع عن هذا الإجراء.`,
            `Are you sure you want to delete "${deleteTarget?.name}"? This cannot be undone.`
          )}
        </p>
        <div className="flex gap-3">
          <button onClick={() => setDeleteTarget(null)} className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
            {t('إلغاء', 'Cancel')}
          </button>
          <button onClick={handleDelete} disabled={deleting} className="flex-1 py-2 bg-red-600 text-white rounded-lg text-sm font-medium hover:bg-red-700 disabled:opacity-60">
            {deleting ? t('جارٍ الحذف...', 'Deleting...') : t('حذف', 'Delete')}
          </button>
        </div>
      </Modal>
    </div>
  )
}
