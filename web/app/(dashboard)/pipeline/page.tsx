'use client'
import { useEffect, useState, useCallback } from 'react'
import {
  DndContext, DragEndEvent, DragOverlay, DragStartEvent,
  PointerSensor, useSensor, useSensors, closestCorners,
} from '@dnd-kit/core'
import { Header } from '@/components/layout/Header'
import { KanbanColumn } from '@/components/kanban/Column'
import { KanbanCard } from '@/components/kanban/Card'
import { Modal, FormField, FormError } from '@/components/ui/Modal'
import { CommunicationHistoryComponent } from '@/components/communication/CommunicationHistory'
import EmailComposeModal from '@/components/communication/EmailComposeModal'
import { AgentAssist } from '@/components/agent/AgentAssist'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import type { KanbanBoard, Lead, LeadStage, Contact, PaginatedResult, CommunicationHistory, User, PipelineStage } from '@/types'

export default function PipelinePage() {
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'
  const isAgent = user?.role === 'agent' || isAdmin

  const [stages, setStages] = useState<PipelineStage[]>([])
  const [board, setBoard] = useState<KanbanBoard>({})
  const [activeCard, setActiveCard] = useState<Lead | null>(null)
  const [loading, setLoading] = useState(true)
  const [showAddModal, setShowAddModal] = useState(false)
  const [contacts, setContacts] = useState<Contact[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState('')
  const { t } = useLang()

  // Lead detail / notes modal
  const [selectedLead, setSelectedLead] = useState<Lead | null>(null)
  const [notes, setNotes] = useState('')
  const [notesSaving, setNotesSaving] = useState(false)
  const [notesSaved, setNotesSaved] = useState(false)
  const [notesError, setNotesError] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [scoring, setScoring] = useState(false)
  const [logCallOpen, setLogCallOpen] = useState(false)
  const [callNotes, setCallNotes] = useState('')
  const [emailOpen, setEmailOpen] = useState(false)
  const [tags, setTags] = useState<string[]>([])
  const [newTag, setNewTag] = useState('')
  const [addingTag, setAddingTag] = useState(false)

  // Assignment
  const [selectedAssignee, setSelectedAssignee] = useState('')
  const [assigning, setAssigning] = useState(false)

  // Communication history
  const [communications, setCommunications] = useState<CommunicationHistory[]>([])
  const [commLoading, setCommLoading] = useState(false)
  const [activeTab, setActiveTab] = useState<'notes' | 'history' | 'assist'>('history')

  // Agent assist (Phase 8)
  const [threadId, setThreadId] = useState<string>('')
  const [threadConversation, setThreadConversation] = useState('')

  const handleOpenLead = async (lead: Lead) => {
    setSelectedLead(lead)
    setNotes(lead.notes ?? '')
    setNotesSaved(false)
    setNotesError('')
    setActiveTab('history')
    setSelectedAssignee(lead.assigned_to || '')
    setDeleteConfirm(false)
    setNewTag('')

    // Fetch tags
    try {
      const data = await api.leads.getTags(lead.id) as string[]
      setTags(Array.isArray(data) ? data : [])
    } catch { setTags([]) }

    // Fetch communication history
    setCommLoading(true)
    try {
      const response = (await api.leads.communications(lead.id, 100)) as CommunicationHistory[]
      setCommunications(Array.isArray(response) ? response : [])
    } catch (err) {
      console.error('Failed to fetch communications:', err)
      setCommunications([])
    } finally {
      setCommLoading(false)
    }

    // Fetch thread data for agent assist using contact_id filter
    setThreadId('')
    setThreadConversation('')
    try {
      const threads = await api.threads.list({ contact_id: lead.contact_id, limit: 5 }) as any
      const threadsList = threads?.data || []
      if (threadsList.length > 0) {
        const match = threadsList.find((t: any) => t.contact_id === lead.contact_id) || threadsList[0]
        setThreadId(match.id)
        const messages = await api.threads.messages(match.id)
        const messagesList = Array.isArray(messages) ? messages : (messages as any).data || []
        const conversation = messagesList.map((m: any) => m.body).join('\n')
        setThreadConversation(conversation)
      }
    } catch (err) {
      console.error('Failed to fetch thread data:', err)
    }

    // Load users for assignment dropdown
    try {
      const usersData = await api.users.list() as User[]
      setUsers(Array.isArray(usersData) ? usersData : [])
    } catch { setUsers([]) }
  }

  const handleSaveNotes = async () => {
    if (!selectedLead) return
    setNotesSaving(true)
    setNotesSaved(false)
    setNotesError('')
    try {
      await api.leads.updateNotes(selectedLead.id, notes)
      setBoard((prev) => {
        const next = { ...prev }
        for (const stage of Object.keys(next)) {
          next[stage] = (next[stage] ?? []).map((l) =>
            l.id === selectedLead.id ? { ...l, notes } : l
          )
        }
        return next
      })
      setSelectedLead((prev) => prev ? { ...prev, notes } : null)
      setNotesSaved(true)
    } catch (err: unknown) {
      setNotesError(err instanceof Error ? err.message : t('حدث خطأ', 'Something went wrong'))
    } finally {
      setNotesSaving(false)
    }
  }

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
  )

  const load = useCallback(async () => {
    try {
      const [boardData, stageData] = await Promise.all([
        api.leads.kanban() as Promise<KanbanBoard>,
        api.pipelineStages.list('lead') as Promise<PipelineStage[]>,
      ])
      setBoard(boardData ?? {})
      setStages(stageData ?? [])
    } catch {
      // handle error silently
    } finally {
      setLoading(false)
    }
  }, [])

  const loadContacts = useCallback(async () => {
    try {
      const data = await api.contacts.list({ limit: 100 }) as PaginatedResult<Contact>
      setContacts(data.data ?? [])
    } catch {
      setContacts([])
    }
  }, [])

  const handleOpenAddModal = () => {
    loadContacts()
    api.users.list().then((data) => setUsers(Array.isArray(data) ? data : [])).catch(() => {})
    setShowAddModal(true)
  }

  const handleAddLead = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setSubmitError('')
    setSubmitting(true)
    
    const formData = new FormData(e.currentTarget)
    const data: Record<string, unknown> = {
      contact_id: formData.get('contact_id'),
      source: formData.get('source') || 'whatsapp',
      deal_value: parseFloat(formData.get('deal_value') as string) || 0,
      currency: (formData.get('currency') as string) || 'AED',
      notes: formData.get('notes') as string || '',
      assigned_to: (formData.get('assigned_to') as string) || null,
    }

    if (!data.contact_id) {
      setSubmitError(t('يرجى اختيار جهة اتصال', 'Please select a contact'))
      setSubmitting(false)
      return
    }

    try {
      await api.leads.create(data)
      setShowAddModal(false)
      load()
    } catch (err: any) {
      setSubmitError(err.message || t('حدث خطأ', 'Something went wrong'))
    } finally {
      setSubmitting(false)
    }
  }

  useEffect(() => { load() }, [load])

  const findCard = (id: string): Lead | null => {
    for (const leads of Object.values(board)) {
      const found = leads?.find((l) => l.id === id)
      if (found) return found
    }
    return null
  }

  const findStage = (id: string): string | null => {
    for (const [stage, leads] of Object.entries(board)) {
      if (leads?.some((l) => l.id === id)) return stage
    }
    return null
  }

  const handleDragStart = (e: DragStartEvent) => {
    setActiveCard(findCard(String(e.active.id)))
  }

  const handleDragEnd = async (e: DragEndEvent) => {
    setActiveCard(null)
    const { active, over } = e
    if (!over) return

    const leadId = String(active.id)
    const overStage = stages.find(s => s.id === over.id)
    const targetStage = overStage
      ? overStage.name
      : (stages.find(s => s.name === over.id)?.name || findStage(String(over.id)))

    const currentStage = findStage(leadId)
    if (!targetStage || targetStage === currentStage) return

    const card = findCard(leadId)
    const wonStage = stages.find(s => s.is_won)?.name
    const lostStage = stages.find(s => s.is_lost)?.name
    let closedReason = ''
    if ((targetStage === wonStage || targetStage === lostStage) && card) {
      closedReason = prompt(t('سبب الإغلاق (اختياري):', 'Closing reason (optional):')) || ''
    }

    // Optimistic update
    setBoard((prev) => {
      const next: Record<string, Lead[]> = { ...prev }
      const existingCard = (next[currentStage!] ?? []).find((l) => l.id === leadId)
      if (!existingCard) return prev
      next[currentStage!] = (next[currentStage!] ?? []).filter((l) => l.id !== leadId)
      next[targetStage] = [{ ...existingCard, stage: targetStage as LeadStage, closed_reason: closedReason }, ...(next[targetStage!] ?? [])]
      return next
    })

    await api.leads.updateStage(leadId, targetStage, closedReason || undefined).catch(() => load())
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-surface-400 text-sm">
        {t('جاري التحميل...', 'Loading...')}
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <Header title={t('خط الأنابيب', 'Sales Pipeline')} />

      <div className="flex-1 overflow-x-auto p-6 bg-surface-50">
        <div className="flex justify-end mb-5">
          <button
            onClick={handleOpenAddModal}
            className="inline-flex items-center gap-1.5 px-4 py-2 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 hover:shadow-card-hover transition-all duration-200 ease-soft"
          >
            <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path fillRule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clipRule="evenodd" />
            </svg>
            {t('Lead جديد', 'New Lead')}
          </button>
        </div>

        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <div className="flex gap-4 min-w-max pb-4">
            {stages.map((s) => (
              <KanbanColumn
                key={s.id}
                stage={s.name}
                stageName={s.name}
                stageColor={s.color}
                leads={board[s.name] ?? []}
                onOpenLead={handleOpenLead}
              />
            ))}
          </div>

          <DragOverlay>
            {activeCard && <KanbanCard lead={activeCard} />}
          </DragOverlay>
        </DndContext>
      </div>

      {/* Add Lead Modal */}
      <Modal
        open={showAddModal}
        onClose={() => setShowAddModal(false)}
        title={t('إضافة Lead جديد', 'Add New Lead')}
      >
        <form onSubmit={handleAddLead} className="space-y-4">
          <FormField label={t('جهة الاتصال *', 'Contact *')}>
            <select
              name="contact_id"
              required
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            >
              <option value="">{t('اختر جهة اتصال', 'Select a contact')}</option>
              {contacts.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.full_name} ({c.phone_wa})
                </option>
              ))}
            </select>
          </FormField>
          
          <FormField label={t('المصدر', 'Source')}>
            <select
              name="source"
              defaultValue="whatsapp"
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            >
              <option value="whatsapp">{t('واتساب', 'WhatsApp')}</option>
              <option value="web">{t('الموقع', 'Website')}</option>
              <option value="referral">{t('إحالة', 'Referral')}</option>
              <option value="event">{t('فعالية', 'Event')}</option>
            </select>
          </FormField>
          
          <FormField label={t('قيمة الصفقة', 'Deal Value')}>
            <div className="flex gap-2">
              <input
                type="number"
                name="deal_value"
                min="0"
                step="1"
                placeholder="0"
                className="flex-1 px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
              />
              <select
                name="currency"
                defaultValue="AED"
                className="w-24 px-2 py-2.5 border border-surface-200 rounded-xl text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40"
              >
                <option value="AED">AED</option>
                <option value="USD">USD</option>
              </select>
            </div>
          </FormField>

          <FormField label={t('ملاحظات', 'Notes')}>
            <textarea
              name="notes"
              rows={2}
              placeholder={t('أضف ملاحظات...', 'Add notes...')}
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 resize-none transition-shadow"
            />
          </FormField>

          <FormField label={t('مسؤول', 'Assigned To')}>
            <select
              name="assigned_to"
              className="w-full px-3.5 py-2.5 border border-surface-200 rounded-xl text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 transition-shadow"
            >
              <option value="">{t('غير معين', 'Unassigned')}</option>
              {users.map((u) => (
                <option key={u.id} value={u.id}>{u.name} ({u.role})</option>
              ))}
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

      {/* Lead Detail / Notes Modal */}
      <Modal
        open={!!selectedLead}
        onClose={() => setSelectedLead(null)}
        title={selectedLead?.contact?.full_name ?? t('تفاصيل Lead', 'Lead Details')}
      >
        {selectedLead && (
          <div className="space-y-4">
            {/* Lead meta */}
            <div className="grid grid-cols-2 gap-2.5 text-sm">
              <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3">
                <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('المرحلة', 'Stage')}</p>
                <p className="font-semibold text-surface-900 capitalize">{selectedLead.stage}</p>
              </div>
              <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3">
                <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('القيمة', 'Value')}</p>
                <p className="font-semibold text-surface-900 tabular-nums">
                  {selectedLead.currency} {selectedLead.deal_value.toLocaleString()}
                </p>
              </div>
              {selectedLead.contact?.phone_wa && (
                <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3 col-span-2">
                  <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('واتساب', 'WhatsApp')}</p>
                  <p className="font-semibold text-surface-900 font-mono">{selectedLead.contact.phone_wa}</p>
                </div>
              )}
              {selectedLead.contact?.email && (
                <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3 col-span-2">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('البريد', 'Email')}</p>
                      <p className="font-semibold text-surface-900">{selectedLead.contact.email}</p>
                    </div>
                    <button
                      onClick={() => setEmailOpen(true)}
                      className="px-3 py-1.5 bg-primary-600 text-white text-xs font-medium rounded-lg hover:bg-primary-700 transition-colors"
                    >
                      {t('إرسال بريد', 'Send Email')}
                    </button>
                  </div>
                </div>
              )}
              {selectedLead.source && (
                <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3">
                  <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('المصدر', 'Source')}</p>
                  <p className="font-semibold text-surface-900 capitalize">{selectedLead.source}</p>
                </div>
              )}
              {selectedLead.closed_reason && (
                <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3 col-span-2">
                  <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('سبب الإغلاق', 'Closed Reason')}</p>
                  <p className="font-semibold text-surface-900">{selectedLead.closed_reason}</p>
                </div>
              )}
              {selectedLead.lead_score != null && (
                <div className="bg-surface-50 border border-surface-200/70 rounded-xl p-3">
                  <p className="text-[11px] uppercase tracking-wide text-surface-500 mb-1 font-medium">{t('نقاط العميل', 'Lead Score')}</p>
                  <p className="font-semibold text-surface-900 tabular-nums">{selectedLead.lead_score}/100</p>
                </div>
              )}
            </div>

            {/* Assignment */}
            <div className="flex items-center gap-2">
              <select
                value={selectedAssignee}
                onChange={(e) => setSelectedAssignee(e.target.value)}
                className="flex-1 px-3 py-2 border border-surface-200 rounded-xl text-sm bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40"
              >
                <option value="">{t('غير معين', 'Unassigned')}</option>
                {users.map((u) => (
                  <option key={u.id} value={u.id}>{u.name}</option>
                ))}
              </select>
              <button
                disabled={assigning}
                onClick={async () => {
                  setAssigning(true)
                  try {
                    await api.leads.assign(selectedLead.id, selectedAssignee || null)
                    setSelectedLead((prev) => prev ? { ...prev, assigned_to: selectedAssignee || null } : null)
                    load()
                  } catch {} finally { setAssigning(false) }
                }}
                className="px-3 py-2 bg-primary-600 text-white text-sm font-medium rounded-xl hover:bg-primary-700 disabled:opacity-60 transition-colors"
              >
                {assigning ? t('...', '...') : t('تعيين', 'Assign')}
              </button>
              <button
                disabled={scoring}
                onClick={async () => {
                  setScoring(true)
                  try {
                    await api.leads.reScore(selectedLead.id)
                    await load()
                    const refreshed = findCard(selectedLead.id)
                    if (refreshed) setSelectedLead(refreshed)
                  } catch {} finally { setScoring(false) }
                }}
                className="px-3 py-2 bg-gold-500 text-white text-sm font-medium rounded-xl hover:bg-gold-600 disabled:opacity-60 transition-colors"
                title={t('إعادة تقييم بواسطة AI', 'Re-score with AI')}
              >
                {scoring ? '...' : '🤖'}
              </button>
            </div>

            {/* Tags */}
            <div className="flex flex-wrap gap-1.5">
              {tags.map((tag) => (
                <span key={tag} className="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 bg-primary-50 text-primary-700 rounded-full font-medium">
                  {tag}
                  {isAgent && (
                    <button
                      onClick={async () => {
                        await api.leads.removeTag(selectedLead.id, tag)
                        const data = await api.leads.getTags(selectedLead.id) as string[]
                        setTags(Array.isArray(data) ? data : [])
                      }}
                      className="hover:text-red-600 leading-none"
                      title={t('إزالة', 'Remove')}
                    >×</button>
                  )}
                </span>
              ))}
              {isAgent && (
                <form
                  onSubmit={async (e) => {
                    e.preventDefault()
                    if (!newTag.trim()) return
                    setAddingTag(true)
                    try {
                      await api.leads.addTag(selectedLead.id, newTag.trim())
                      setNewTag('')
                      const data = await api.leads.getTags(selectedLead.id) as string[]
                      setTags(Array.isArray(data) ? data : [])
                    } catch {} finally { setAddingTag(false) }
                  }}
                  className="inline-flex items-center gap-1"
                >
                  <input
                    value={newTag}
                    onChange={e => setNewTag(e.target.value)}
                    placeholder={t('إضافة وسمة', 'Add tag')}
                    className="text-[11px] px-2 py-0.5 border border-dashed border-primary-300 rounded-full bg-transparent text-primary-600 placeholder-primary-300 focus:outline-none focus:ring-1 focus:ring-primary-400 w-24"
                    disabled={addingTag}
                  />
                </form>
              )}
            </div>

            {/* Tabs */}
            <div className="border-b border-surface-200 flex gap-5">
              <button
                onClick={() => setActiveTab('history')}
                className={`py-2 px-0.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
                  activeTab === 'history'
                    ? 'text-primary-600 border-primary-600'
                    : 'text-surface-500 border-transparent hover:text-surface-900'
                }`}
              >
                {t('السجل', 'History')}
              </button>
              <button
                onClick={() => setActiveTab('assist')}
                className={`py-2 px-0.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
                  activeTab === 'assist'
                    ? 'text-primary-600 border-primary-600'
                    : 'text-surface-500 border-transparent hover:text-surface-900'
                }`}
              >
                💡 {t('المساعد', 'Assist')}
              </button>
              <button
                onClick={() => setActiveTab('notes')}
                className={`py-2 px-0.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
                  activeTab === 'notes'
                    ? 'text-primary-600 border-primary-600'
                    : 'text-surface-500 border-transparent hover:text-surface-900'
                }`}
              >
                {t('الملاحظات', 'Notes')}
              </button>
            </div>

            {/* Tab content */}
            {activeTab === 'history' && (
              <div className="max-h-96 overflow-y-auto space-y-3">
                <div className="flex gap-2">
                  <button
                    onClick={() => setLogCallOpen(true)}
                    className="flex-1 px-3 py-2 border border-dashed border-surface-300 text-surface-500 text-xs font-medium rounded-xl hover:bg-surface-50 hover:border-surface-400 transition-colors"
                  >
                    + {t('تسجيل مكالمة', 'Log a Call')}
                  </button>
                  {selectedLead.contact?.email && (
                    <button
                      onClick={() => setEmailOpen(true)}
                      className="flex-1 px-3 py-2 border border-dashed border-primary-300 text-primary-600 text-xs font-medium rounded-xl hover:bg-primary-50 hover:border-primary-400 transition-colors"
                    >
                      ✉ {t('إرسال بريد', 'Send Email')}
                    </button>
                  )}
                </div>
                <CommunicationHistoryComponent
                  communications={communications}
                  loading={commLoading}
                />
              </div>
            )}

            {activeTab === 'assist' && threadId && (
              <div className="max-h-96 overflow-y-auto space-y-3">
                <AgentAssist
                  leadId={selectedLead.id}
                  threadId={threadId}
                  contactName={selectedLead.contact?.full_name || 'Unknown'}
                  conversation={threadConversation}
                  onActionClick={async (action, message) => {
                    if (message && threadId) {
                      try {
                        await api.whatsapp.sendMessage(threadId, message)
                      } catch {}
                    }
                  }}
                />
                <button
                  onClick={async () => {
                    try {
                      const msg = threadConversation.split('\n').filter(Boolean).slice(-5).join('\n')
                      await api.messages.autoCreateLead({ thread_id: threadId, message: msg })
                      load()
                    } catch {}
                  }}
                  className="w-full px-3 py-2 bg-emerald-600 text-white text-xs font-medium rounded hover:bg-emerald-700 transition-colors"
                >
                  {t('تحويل المحادثة إلى Lead', 'Create Lead from Conversation')}
                </button>
              </div>
            )}

            {activeTab === 'assist' && !threadId && (
              <div className="py-8 text-center text-surface-400 text-sm">
                {t('لا توجد محادثة لهذا العميل', 'No conversation for this lead')}
              </div>
            )}

            {activeTab === 'notes' && (
              <div>
                <label className="block text-[13px] font-medium text-surface-700 mb-1.5">
                  {t('الملاحظات', 'Notes')}
                </label>
                <textarea
                  value={notes}
                  onChange={(e) => { setNotes(e.target.value); setNotesSaved(false) }}
                  rows={5}
                  placeholder={t('أضف ملاحظات حول هذا العميل المحتمل...', 'Add notes about this lead...')}
                  className="w-full text-sm border border-surface-200 rounded-xl px-3.5 py-2.5 bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 focus:border-primary-400 resize-none transition-shadow"
                />
              </div>
            )}

            {notesError && (
              <p className="text-xs text-red-600 bg-red-50 border border-red-100 px-3 py-2 rounded-lg">{notesError}</p>
            )}
            {notesSaved && (
              <p className="text-xs text-emerald-600 font-medium">{t('تم الحفظ', 'Saved')}</p>
            )}

            <div className="flex gap-2 pt-1">
              {deleteConfirm ? (
                <>
                  <button
                    type="button"
                    onClick={() => setDeleteConfirm(false)}
                    className="flex-1 px-4 py-2.5 border border-surface-200 text-surface-700 text-sm font-medium rounded-xl bg-white hover:bg-surface-50 transition-colors"
                  >
                    {t('إلغاء', 'Cancel')}
                  </button>
                  <button
                    type="button"
                    disabled={deleting}
                    onClick={async () => {
                      setDeleting(true)
                      try {
                        await api.leads.delete(selectedLead!.id)
                        setSelectedLead(null)
                        load()
                      } catch {} finally { setDeleting(false) }
                    }}
                    className="flex-1 px-4 py-2.5 bg-red-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-red-700 disabled:opacity-60 transition-all"
                  >
                    {deleting ? t('جاري الحذف...', 'Deleting...') : t('تأكيد الحذف', 'Confirm Delete')}
                  </button>
                </>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={() => setSelectedLead(null)}
                    className="flex-1 px-4 py-2.5 border border-surface-200 text-surface-700 text-sm font-medium rounded-xl bg-white hover:bg-surface-50 transition-colors"
                  >
                    {t('إغلاق', 'Close')}
                  </button>
                  <button
                    type="button"
                    onClick={() => setDeleteConfirm(true)}
                    className="px-4 py-2.5 border border-red-200 text-red-600 text-sm font-medium rounded-xl bg-white hover:bg-red-50 transition-colors"
                  >
                    {t('حذف', 'Delete')}
                  </button>
                  {activeTab === 'notes' && (
                    <button
                      type="button"
                      onClick={handleSaveNotes}
                      disabled={notesSaving}
                      className="flex-1 px-4 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 hover:shadow-card-hover disabled:opacity-60 transition-all"
                    >
                      {notesSaving ? t('جاري الحفظ...', 'Saving...') : t('حفظ الملاحظات', 'Save Notes')}
                    </button>
                  )}
                </>
              )}
            </div>
          </div>
        )}
      </Modal>

      {/* Log Call Modal */}
      <Modal
        open={logCallOpen}
        onClose={() => { setLogCallOpen(false); setCallNotes('') }}
        title={t('تسجيل مكالمة', 'Log a Call')}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-surface-700 mb-1">{t('ملاحظات المكالمة', 'Call Notes')}</label>
            <textarea
              value={callNotes}
              onChange={(e) => setCallNotes(e.target.value)}
              rows={4}
              placeholder={t('أضف ملاحظات حول المكالمة...', 'Add call notes...')}
              className="w-full text-sm border border-surface-200 rounded-xl px-3.5 py-2.5 bg-white placeholder:text-surface-400 focus:outline-none focus:ring-2 focus:ring-primary-500/40 resize-none transition-shadow"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => { setLogCallOpen(false); setCallNotes('') }}
              className="flex-1 px-4 py-2.5 border border-surface-200 text-surface-700 text-sm font-medium rounded-xl bg-white hover:bg-surface-50 transition-colors"
            >
              {t('إلغاء', 'Cancel')}
            </button>
            <button
              onClick={async () => {
                setLogCallOpen(false)
                setCallNotes('')
                // Refresh communications to show audit entry
                if (selectedLead) {
                  setCommLoading(true)
                  try {
                    const response = await api.leads.communications(selectedLead.id, 100) as CommunicationHistory[]
                    setCommunications(Array.isArray(response) ? response : [])
                  } catch {} finally { setCommLoading(false) }
                }
              }}
              className="flex-1 px-4 py-2.5 bg-primary-600 text-white text-sm font-medium rounded-xl shadow-card hover:bg-primary-700 transition-all"
            >
              {t('تسجيل', 'Log')}
            </button>
          </div>
        </div>
      </Modal>

      {/* Email Compose */}
      {selectedLead && (
        <EmailComposeModal
          open={emailOpen}
          onClose={() => setEmailOpen(false)}
          toEmail={selectedLead.contact?.email}
          toName={selectedLead.contact?.full_name}
          relatedTo="lead"
          relatedId={selectedLead.id}
          onSent={() => {
            if (selectedLead) {
              setCommLoading(true)
              api.leads.communications(selectedLead.id, 100).then(response => {
                setCommunications(Array.isArray(response) ? response : [])
              }).catch(() => {}).finally(() => setCommLoading(false))
            }
          }}
        />
      )}
    </div>
  )
}
