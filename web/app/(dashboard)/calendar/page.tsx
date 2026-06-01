'use client'
import { useState, useEffect, useCallback } from 'react'
import { Header } from '@/components/layout/Header'
import { api } from '@/lib/api'
import { useLang } from '@/context/LangContext'
import { useAuthStore } from '@/store/auth'
import Link from 'next/link'

// ── Types ─────────────────────────────────────────────────────────────────────
interface Viewing {
  id: string
  listing_id: string | null
  listing_title: string
  contact: { id: string; full_name: string; phone_wa: string }
  agent_id: string | null
  agent_name: string
  scheduled_at: string
  duration_min: number
  status: string
  address: string
  notes: string
  checked_in_at: string | null
  checked_out_at: string | null
}

// ── Status colours ────────────────────────────────────────────────────────────
const STATUS_BG: Record<string, string> = {
  scheduled:  'bg-blue-100 text-blue-800 border-blue-200',
  confirmed:  'bg-green-100 text-green-800 border-green-200',
  checked_in: 'bg-purple-100 text-purple-800 border-purple-200',
  completed:  'bg-gray-100 text-gray-600 border-gray-200',
  cancelled:  'bg-red-50 text-red-600 border-red-200 line-through opacity-60',
  no_show:    'bg-amber-50 text-amber-700 border-amber-200',
}
const STATUS_DOT: Record<string, string> = {
  scheduled: 'bg-blue-400', confirmed: 'bg-green-400', checked_in: 'bg-purple-400',
  completed: 'bg-gray-400', cancelled: 'bg-red-400', no_show: 'bg-amber-400',
}
const STATUS_LABEL: Record<string, string> = {
  scheduled: 'Scheduled', confirmed: 'Confirmed', checked_in: 'Checked In',
  completed: 'Completed', cancelled: 'Cancelled', no_show: 'No Show',
}

// ── Calendar helpers ──────────────────────────────────────────────────────────
function getDaysInMonth(year: number, month: number) {
  return new Date(year, month + 1, 0).getDate()
}
function getFirstDayOfMonth(year: number, month: number) {
  return new Date(year, month, 1).getDay()
}
function isoDateStr(y: number, m: number, d: number) {
  return `${y}-${String(m+1).padStart(2,'0')}-${String(d).padStart(2,'0')}`
}

// ── Component ─────────────────────────────────────────────────────────────────
export default function CalendarPage() {
  const { t } = useLang()
  const { user, company } = useAuthStore()
  const isAgent = user?.role === 'admin' || user?.role === 'agent'
  const isDemo = !!company?.is_demo

  const today = new Date()
  const [year, setYear] = useState(today.getFullYear())
  const [month, setMonth] = useState(today.getMonth())
  const [view, setView] = useState<'month' | 'week' | 'list'>('month')
  const [viewings, setViewings] = useState<Viewing[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<Viewing | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [createDate, setCreateDate] = useState('')
  const [acting, setActing] = useState<string | null>(null)

  // Create form
  const [form, setForm] = useState({
    contact_phone: '', scheduled_at: '', duration_min: '60',
    address: '', notes: '', listing_id: '',
  })
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    // Fetch entire month (+ a week buffer each side for week view)
    const from = new Date(year, month - 1, 24).toISOString()
    const to   = new Date(year, month + 1, 8).toISOString()
    try {
      const r = await api.viewings.list({ from, to, limit: 500 }) as any
      setViewings(r.data ?? [])
    } catch { setViewings([]) }
    finally { setLoading(false) }
  }, [year, month])

  useEffect(() => { load() }, [load])

  // ── Helpers ────────────────────────────────────────────────────────────────
  const viewingsOnDay = (dateStr: string) =>
    viewings.filter(v => v.scheduled_at.startsWith(dateStr))

  const handleStatusChange = async (id: string, status: string) => {
    if (isDemo) return
    setActing(id)
    try {
      await api.viewings.updateStatus(id, status)
      await load()
      if (selected?.id === id) setSelected(null)
    } catch {}
    finally { setActing(null) }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isDemo) return
    setCreating(true); setCreateError('')
    try {
      // Look up contact by phone
      const contacts = await api.contacts.list({ search: form.contact_phone, page: 1, limit: 1 }) as any
      const contactId = contacts?.data?.[0]?.id
      if (!contactId) { setCreateError('Contact not found by that phone number.'); return }

      await api.viewings.create({
        contact_id: contactId,
        scheduled_at: new Date(form.scheduled_at).toISOString(),
        duration_min: parseInt(form.duration_min) || 60,
        address: form.address,
        notes: form.notes,
        listing_id: form.listing_id || undefined,
      })
      setShowCreate(false)
      setForm({ contact_phone: '', scheduled_at: '', duration_min: '60', address: '', notes: '', listing_id: '' })
      await load()
    } catch (err: any) { setCreateError(err.message || 'Failed to schedule') }
    finally { setCreating(false) }
  }

  const prevMonth = () => { if (month === 0) { setYear(y => y-1); setMonth(11) } else setMonth(m => m-1) }
  const nextMonth = () => { if (month === 11) { setYear(y => y+1); setMonth(0) } else setMonth(m => m+1) }

  const MONTH_NAMES = ['January','February','March','April','May','June','July','August','September','October','November','December']
  const DAY_NAMES   = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat']

  const daysInMonth  = getDaysInMonth(year, month)
  const firstDay     = getFirstDayOfMonth(year, month)
  const todayStr     = isoDateStr(today.getFullYear(), today.getMonth(), today.getDate())

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="flex flex-col flex-1 min-h-0">
      <Header title={t('التقويم', 'Calendar')} />
      <div className="flex flex-1 min-h-0 overflow-hidden">

        {/* ── Sidebar ──────────────────────────────────────────────────── */}
        <aside className="w-72 shrink-0 border-r border-gray-200 bg-white flex flex-col overflow-y-auto">
          <div className="p-4 border-b border-gray-100">
            {isAgent && !isDemo && (
              <button onClick={() => { setShowCreate(true); setForm(f => ({ ...f, scheduled_at: `${todayStr}T10:00` })) }}
                className="w-full py-2.5 bg-brand-600 text-white text-sm font-semibold rounded-lg hover:bg-brand-700 flex items-center justify-center gap-2">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
                </svg>
                {t('جدولة معاينة', 'Schedule Viewing')}
              </button>
            )}
            {isDemo && (
              <div className="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg p-2.5">
                🔒 <strong>Demo</strong> — read-only. <a href="/signup" className="underline">Sign up</a> to schedule viewings.
              </div>
            )}
          </div>

          {/* Upcoming viewings list */}
          <div className="flex-1 overflow-y-auto p-3">
            <p className="text-[10px] text-gray-400 uppercase tracking-wide font-semibold mb-2 px-1">
              {t('القادمة', 'Upcoming')}
            </p>
            {viewings
              .filter(v => new Date(v.scheduled_at) >= today && v.status !== 'cancelled')
              .slice(0, 10)
              .map(v => (
                <button key={v.id} onClick={() => setSelected(v)}
                  className={`w-full text-left p-2.5 rounded-lg mb-1 border text-xs transition-colors hover:bg-gray-50 ${
                    selected?.id === v.id ? 'bg-brand-50 border-brand-200' : 'border-gray-100'
                  }`}>
                  <div className="flex items-center gap-1.5 mb-0.5">
                    <span className={`w-2 h-2 rounded-full shrink-0 ${STATUS_DOT[v.status]}`} />
                    <span className="font-semibold text-gray-800 truncate">{v.contact?.full_name}</span>
                  </div>
                  <p className="text-gray-500 pl-3.5">{new Date(v.scheduled_at).toLocaleDateString('en-AE', { weekday:'short', month:'short', day:'numeric' })} · {new Date(v.scheduled_at).toLocaleTimeString('en-AE', { hour:'2-digit', minute:'2-digit' })}</p>
                  {v.listing_title && <p className="text-gray-400 pl-3.5 truncate">{v.listing_title}</p>}
                </button>
              ))}
          </div>
        </aside>

        {/* ── Main calendar area ───────────────────────────────────────── */}
        <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
          {/* Toolbar */}
          <div className="flex items-center gap-3 px-4 py-3 border-b border-gray-200 bg-white shrink-0">
            <button onClick={prevMonth} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-600">‹</button>
            <h2 className="text-sm font-semibold text-gray-800 min-w-[140px] text-center">
              {MONTH_NAMES[month]} {year}
            </h2>
            <button onClick={nextMonth} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-600">›</button>
            <button onClick={() => { setYear(today.getFullYear()); setMonth(today.getMonth()) }}
              className="ml-1 px-3 py-1.5 text-xs border border-gray-200 rounded-lg hover:bg-gray-50 text-gray-600">
              {t('اليوم', 'Today')}
            </button>
            <div className="ml-auto flex items-center gap-1">
              {(['month','week','list'] as const).map(v => (
                <button key={v} onClick={() => setView(v)}
                  className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-colors ${view === v ? 'bg-brand-600 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>
                  {v.charAt(0).toUpperCase() + v.slice(1)}
                </button>
              ))}
            </div>
          </div>

          <div className="flex-1 overflow-y-auto">
            {/* ── Month view ──────────────────────────────────────────── */}
            {view === 'month' && (
              <div className="h-full flex flex-col">
                {/* Day headers */}
                <div className="grid grid-cols-7 border-b border-gray-200 shrink-0">
                  {DAY_NAMES.map(d => (
                    <div key={d} className="py-2 text-center text-[11px] font-semibold text-gray-400 uppercase tracking-wide">
                      {d}
                    </div>
                  ))}
                </div>
                {/* Calendar grid */}
                <div className="grid grid-cols-7 flex-1" style={{ gridAutoRows: 'minmax(80px, 1fr)' }}>
                  {/* Empty cells before month starts */}
                  {Array.from({ length: firstDay }).map((_, i) => (
                    <div key={`pre-${i}`} className="border-r border-b border-gray-100 bg-gray-50/50" />
                  ))}
                  {/* Day cells */}
                  {Array.from({ length: daysInMonth }).map((_, i) => {
                    const day  = i + 1
                    const dateStr = isoDateStr(year, month, day)
                    const dayViewings = viewingsOnDay(dateStr)
                    const isToday = dateStr === todayStr

                    return (
                      <div key={day} onClick={() => { setCreateDate(dateStr); if (isAgent && !isDemo) { setShowCreate(true); setForm(f => ({ ...f, scheduled_at: `${dateStr}T10:00` })) } }}
                        className={`border-r border-b border-gray-100 p-1.5 cursor-pointer hover:bg-gray-50/80 transition-colors ${isToday ? 'bg-blue-50/40' : ''}`}>
                        <span className={`inline-flex w-6 h-6 items-center justify-center rounded-full text-xs font-medium mb-1 ${
                          isToday ? 'bg-brand-600 text-white' : 'text-gray-700'
                        }`}>
                          {day}
                        </span>
                        <div className="space-y-0.5">
                          {dayViewings.slice(0, 3).map(v => (
                            <div key={v.id} onClick={e => { e.stopPropagation(); setSelected(v) }}
                              className={`px-1.5 py-0.5 rounded text-[10px] font-medium truncate border cursor-pointer ${STATUS_BG[v.status]}`}>
                              {new Date(v.scheduled_at).toLocaleTimeString('en-AE', { hour:'2-digit', minute:'2-digit' })} {v.contact?.full_name}
                            </div>
                          ))}
                          {dayViewings.length > 3 && (
                            <p className="text-[10px] text-gray-400 pl-1">+{dayViewings.length - 3} more</p>
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {/* ── List view ───────────────────────────────────────────── */}
            {view === 'list' && (
              <div className="p-4 space-y-2">
                {loading ? (
                  <div className="text-center py-12 text-sm text-gray-400">Loading...</div>
                ) : viewings.length === 0 ? (
                  <div className="text-center py-12 text-sm text-gray-400">{t('لا توجد معاينات هذا الشهر', 'No viewings this month')}</div>
                ) : viewings.map(v => (
                  <div key={v.id} onClick={() => setSelected(v)}
                    className={`p-3 bg-white rounded-xl border cursor-pointer hover:shadow-sm transition-all ${
                      selected?.id === v.id ? 'border-brand-300 shadow-sm' : 'border-gray-200'
                    }`}>
                    <div className="flex items-center gap-2">
                      <span className={`w-2 h-2 rounded-full shrink-0 ${STATUS_DOT[v.status]}`} />
                      <span className="text-sm font-semibold text-gray-800">{v.contact?.full_name}</span>
                      <span className={`ml-auto text-[10px] font-semibold px-2 py-0.5 rounded-full border ${STATUS_BG[v.status]}`}>
                        {STATUS_LABEL[v.status]}
                      </span>
                    </div>
                    <div className="ml-3.5 mt-1 text-xs text-gray-500 space-y-0.5">
                      <p>{new Date(v.scheduled_at).toLocaleString('en-AE', { weekday:'short', month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' })} · {v.duration_min}min</p>
                      {v.address && <p>{v.address}</p>}
                      {v.listing_title && <p className="text-brand-600">{v.listing_title}</p>}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* ── Week view ───────────────────────────────────────────── */}
            {view === 'week' && (
              <WeekView viewings={viewings} today={today} onSelect={setSelected} />
            )}
          </div>
        </div>

        {/* ── Detail panel ────────────────────────────────────────────── */}
        {selected && (
          <div className="w-80 shrink-0 border-l border-gray-200 bg-white flex flex-col overflow-y-auto">
            <div className="flex items-center justify-between p-4 border-b border-gray-100">
              <h3 className="text-sm font-semibold text-gray-800">{t('تفاصيل المعاينة', 'Viewing Details')}</h3>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600">✕</button>
            </div>
            <div className="flex-1 p-4 space-y-4">

              {/* Status badge */}
              <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border ${STATUS_BG[selected.status]}`}>
                <span className={`w-1.5 h-1.5 rounded-full ${STATUS_DOT[selected.status]}`} />
                {STATUS_LABEL[selected.status]}
              </span>

              {/* Info */}
              <div className="space-y-2 text-sm">
                <Row label="Contact" value={selected.contact?.full_name} />
                <Row label="Phone" value={selected.contact?.phone_wa} />
                <Row label="When" value={new Date(selected.scheduled_at).toLocaleString('en-AE', { weekday:'long', month:'long', day:'numeric', hour:'2-digit', minute:'2-digit' })} />
                <Row label="Duration" value={`${selected.duration_min} min`} />
                {selected.address && <Row label="Address" value={selected.address} />}
                {selected.listing_title && <Row label="Listing" value={selected.listing_title} />}
                {selected.agent_name && <Row label="Agent" value={selected.agent_name} />}
                {selected.notes && <Row label="Notes" value={selected.notes} />}
                {selected.checked_in_at && <Row label="Checked in" value={new Date(selected.checked_in_at).toLocaleTimeString()} />}
                {selected.checked_out_at && <Row label="Checked out" value={new Date(selected.checked_out_at).toLocaleTimeString()} />}
              </div>

              {/* Actions */}
              {isAgent && !isDemo && selected.status !== 'completed' && selected.status !== 'cancelled' && (
                <div className="space-y-1.5">
                  {selected.status === 'scheduled' && (
                    <button onClick={() => handleStatusChange(selected.id, 'confirmed')} disabled={acting === selected.id}
                      className="w-full py-2 bg-green-600 text-white text-xs font-semibold rounded-lg hover:bg-green-700 disabled:opacity-50">
                      ✓ Confirm
                    </button>
                  )}
                  {(selected.status === 'scheduled' || selected.status === 'confirmed') && (
                    <button onClick={() => handleStatusChange(selected.id, 'checked_in')} disabled={acting === selected.id}
                      className="w-full py-2 bg-purple-600 text-white text-xs font-semibold rounded-lg hover:bg-purple-700 disabled:opacity-50">
                      📍 Check In
                    </button>
                  )}
                  {selected.status === 'checked_in' && (
                    <button onClick={() => handleStatusChange(selected.id, 'completed')} disabled={acting === selected.id}
                      className="w-full py-2 bg-brand-600 text-white text-xs font-semibold rounded-lg hover:bg-brand-700 disabled:opacity-50">
                      ✓ Complete
                    </button>
                  )}
                  <div className="flex gap-1.5">
                    <button onClick={() => handleStatusChange(selected.id, 'no_show')} disabled={acting === selected.id}
                      className="flex-1 py-2 border border-amber-300 text-amber-700 text-xs font-semibold rounded-lg hover:bg-amber-50 disabled:opacity-50">
                      No Show
                    </button>
                    <button onClick={() => handleStatusChange(selected.id, 'cancelled')} disabled={acting === selected.id}
                      className="flex-1 py-2 border border-red-200 text-red-600 text-xs font-semibold rounded-lg hover:bg-red-50 disabled:opacity-50">
                      Cancel
                    </button>
                  </div>
                </div>
              )}

              {selected.contact && (
                <Link href={`/contacts/${selected.contact.id}`}
                  className="block text-center text-xs text-brand-600 hover:underline">
                  View contact →
                </Link>
              )}
            </div>
          </div>
        )}
      </div>

      {/* ── Create viewing modal ─────────────────────────────────────── */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4"
          onClick={() => setShowCreate(false)}>
          <div className="bg-white rounded-2xl shadow-xl max-w-md w-full p-6" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-base font-semibold text-gray-800">{t('جدولة معاينة', 'Schedule Viewing')}</h2>
              <button onClick={() => setShowCreate(false)} className="text-gray-400 hover:text-gray-600">✕</button>
            </div>
            <form onSubmit={handleCreate} className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('رقم هاتف العميل', 'Buyer Phone')}</label>
                <input value={form.contact_phone} onChange={e => setForm(p => ({ ...p, contact_phone: e.target.value }))}
                  placeholder="+971501234567" required
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('الوقت', 'Date & Time')}</label>
                <input type="datetime-local" value={form.scheduled_at} onChange={e => setForm(p => ({ ...p, scheduled_at: e.target.value }))}
                  required
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('المدة (دقيقة)', 'Duration (min)')}</label>
                  <select value={form.duration_min} onChange={e => setForm(p => ({ ...p, duration_min: e.target.value }))}
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 bg-white focus:outline-none focus:ring-2 focus:ring-brand-500">
                    {[15,30,45,60,90,120].map(d => <option key={d} value={d}>{d} min</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">{t('ID الإعلان', 'Listing ID (opt.)')}</label>
                  <input value={form.listing_id} onChange={e => setForm(p => ({ ...p, listing_id: e.target.value }))}
                    placeholder="uuid"
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500" />
                </div>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('العنوان', 'Address')}</label>
                <input value={form.address} onChange={e => setForm(p => ({ ...p, address: e.target.value }))}
                  placeholder="Property address"
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 mb-1">{t('ملاحظات', 'Notes')}</label>
                <textarea value={form.notes} onChange={e => setForm(p => ({ ...p, notes: e.target.value }))}
                  rows={2} placeholder="Any special notes..."
                  className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-brand-500 resize-none" />
              </div>
              {createError && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{createError}</p>}
              <div className="flex gap-2 pt-1">
                <button type="button" onClick={() => setShowCreate(false)}
                  className="flex-1 py-2.5 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50">
                  {t('إلغاء', 'Cancel')}
                </button>
                <button type="submit" disabled={creating}
                  className="flex-1 py-2.5 bg-brand-600 text-white text-sm font-semibold rounded-lg hover:bg-brand-700 disabled:opacity-50">
                  {creating ? t('جارٍ الجدولة...', 'Scheduling...') : t('جدولة', 'Schedule')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Sub-components ────────────────────────────────────────────────────────────

function Row({ label, value }: { label: string; value?: string }) {
  if (!value) return null
  return (
    <div className="flex gap-2">
      <span className="text-gray-400 w-20 shrink-0 text-xs">{label}</span>
      <span className="text-gray-800 text-xs">{value}</span>
    </div>
  )
}

function WeekView({ viewings, today, onSelect }: {
  viewings: Viewing[]; today: Date; onSelect: (v: Viewing) => void
}) {
  // Find the Monday of this week
  const weekStart = new Date(today)
  weekStart.setDate(today.getDate() - ((today.getDay() + 6) % 7))

  const days = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(weekStart)
    d.setDate(weekStart.getDate() + i)
    return d
  })

  const hours = Array.from({ length: 13 }, (_, i) => i + 8) // 8am–8pm

  return (
    <div className="overflow-x-auto h-full">
      <div className="min-w-[600px] h-full flex flex-col">
        {/* Day headers */}
        <div className="grid grid-cols-8 border-b border-gray-200 shrink-0 bg-white sticky top-0 z-10">
          <div className="border-r border-gray-100" />
          {days.map(d => {
            const isToday = d.toDateString() === today.toDateString()
            return (
              <div key={d.toISOString()} className={`py-2 text-center border-r border-gray-100 last:border-0 ${isToday ? 'bg-blue-50' : ''}`}>
                <p className="text-[10px] text-gray-400 uppercase">{['Mon','Tue','Wed','Thu','Fri','Sat','Sun'][d.getDay() === 0 ? 6 : d.getDay()-1]}</p>
                <p className={`text-sm font-semibold ${isToday ? 'text-brand-600' : 'text-gray-700'}`}>{d.getDate()}</p>
              </div>
            )
          })}
        </div>
        {/* Time grid */}
        <div className="flex-1 overflow-y-auto">
          {hours.map(hour => (
            <div key={hour} className="grid grid-cols-8 border-b border-gray-100" style={{ minHeight: 52 }}>
              <div className="border-r border-gray-100 px-2 py-1 text-[10px] text-gray-400 text-right">{hour}:00</div>
              {days.map(d => {
                const dateStr = `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
                const slotViewings = viewings.filter(v => {
                  const vDate = new Date(v.scheduled_at)
                  return vDate.toDateString() === d.toDateString() && vDate.getHours() === hour
                })
                return (
                  <div key={dateStr} className="border-r border-gray-100 last:border-0 p-0.5 space-y-0.5">
                    {slotViewings.map(v => (
                      <div key={v.id} onClick={() => onSelect(v)}
                        className={`px-1 py-0.5 rounded text-[10px] font-medium truncate border cursor-pointer ${STATUS_BG[v.status]}`}>
                        {new Date(v.scheduled_at).toLocaleTimeString('en-AE', { hour:'2-digit', minute:'2-digit' })} {v.contact?.full_name}
                      </div>
                    ))}
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
