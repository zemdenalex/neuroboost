import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { Layout } from '../../components/Layout'
import {
  ChevronLeft,
  ChevronRight,
  Calendar as CalendarIcon,
  Plus,
  Loader2,
} from 'lucide-react'
import { listEvents, createEvent, updateEvent, deleteEvent, moveEvent, NbEvent, toNbEvent } from '../../api/events'
import type { Event as ApiEvent } from '../../api/events'

// Constants
const MSK_OFFSET_MS = 3 * 60 * 60 * 1000
const DAY_MS = 24 * 60 * 60 * 1000
const HOUR_PX = 48
const MIN_SLOT_MIN = 15
const ALL_DAY_HEIGHT = 60
const DAY_HEADER_HEIGHT = 36

// Helper functions
function mondayUtcMidnightOfCurrentWeek(): number {
  const nowUtcMs = Date.now()
  const nowMsk = new Date(nowUtcMs + MSK_OFFSET_MS)
  const mondayIndex = (nowMsk.getUTCDay() + 6) % 7
  const todayMskMidnight = new Date(nowMsk)
  todayMskMidnight.setUTCHours(0, 0, 0, 0)
  const mondayMskMidnightMs = todayMskMidnight.getTime() - mondayIndex * DAY_MS
  return mondayMskMidnightMs - MSK_OFFSET_MS
}

function mskMidnightUtcMs(utcMs: number): number {
  const msk = new Date(utcMs + MSK_OFFSET_MS)
  msk.setUTCHours(0, 0, 0, 0)
  return msk.getTime() - MSK_OFFSET_MS
}

function minutesSinceMskMidnight(utcISO: string): number {
  const utcMs = new Date(utcISO).getTime()
  const baseUtc = mskMidnightUtcMs(utcMs)
  return Math.max(0, Math.min(1440, Math.round((utcMs - baseUtc) / 60000)))
}

const snapMin = (m: number) => Math.round(m / MIN_SLOT_MIN) * MIN_SLOT_MIN
const minsToTop = (m: number) => (m / 60) * HOUR_PX
const topToMins = (y: number) => (y / HOUR_PX) * 60
const clampMins = (m: number) => Math.max(0, Math.min(1439, m))

function formatTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`
}

export default function Calendar() {
  const [weekOffset, setWeekOffset] = useState(0)
  const [events, setEvents] = useState<NbEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedEvent, setSelectedEvent] = useState<NbEvent | null>(null)
  const [showEditor, setShowEditor] = useState(false)
  const [editingEvent, setEditingEvent] = useState<Partial<NbEvent> | null>(null)
  
  const scrollRef = useRef<HTMLDivElement>(null)
  const baseWeekUtc0 = useMemo(() => mondayUtcMidnightOfCurrentWeek(), [])
  const mondayUtc0 = baseWeekUtc0 + weekOffset * 7 * DAY_MS

  // Generate days array
  const days = useMemo(() => {
    return Array.from({ length: 7 }, (_, i) => {
      const dayUtc0 = mondayUtc0 + i * DAY_MS
      const dayMsk = new Date(dayUtc0 + MSK_OFFSET_MS)
      return {
        i,
        dayUtc0,
        dayMsk,
        isToday: mskMidnightUtcMs(Date.now()) === dayUtc0,
        label: dayMsk.toLocaleDateString('en-US', { weekday: 'short', day: 'numeric' }),
      }
    })
  }, [mondayUtc0])

  // Current time indicator
  const nowInfo = useMemo(() => {
    const nowUtc = Date.now()
    const nowDayUtc0 = mskMidnightUtcMs(nowUtc)
    const nowMsk = new Date(nowUtc + MSK_OFFSET_MS)
    const min = nowMsk.getUTCHours() * 60 + nowMsk.getUTCMinutes()
    return { dayUtc0: nowDayUtc0, min }
  }, [])

  // Fetch events when week changes
  useEffect(() => {
    const fetchEvents = async () => {
      setLoading(true)
      try {
        const start = new Date(mondayUtc0).toISOString()
        const end = new Date(mondayUtc0 + 7 * DAY_MS).toISOString()
        const data = await listEvents(start, end)
        setEvents(data.map(toNbEvent))
      } catch (error) {
        console.error('Failed to fetch events:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchEvents()
  }, [mondayUtc0])

  // Scroll to current time on mount
  useEffect(() => {
    if (scrollRef.current && weekOffset === 0) {
      const targetScroll = minsToTop(Math.max(0, nowInfo.min - 60))
      scrollRef.current.scrollTop = targetScroll
    }
  }, [loading])

  // Separate all-day and timed events
  const { allDayEvents, timedEvents } = useMemo(() => {
    const allDay: NbEvent[] = []
    const timed: (NbEvent & { top: number; height: number; dayUtc0: number })[] = []

    for (const e of events) {
      if (e.allDay) {
        allDay.push(e)
      } else {
        const startMin = minutesSinceMskMidnight(e.startUtc)
        const endMin = Math.max(startMin + MIN_SLOT_MIN, minutesSinceMskMidnight(e.endUtc))
        const top = minsToTop(startMin)
        const height = Math.max(minsToTop(endMin - startMin), minsToTop(MIN_SLOT_MIN))
        const bucketUtc0 = mskMidnightUtcMs(new Date(e.startUtc).getTime())
        timed.push({ ...e, top, height, dayUtc0: bucketUtc0 })
      }
    }

    return { allDayEvents: allDay, timedEvents: timed }
  }, [events])

  // Group timed events by day
  const timedPerDay = useMemo(() => {
    const map = new Map<number, typeof timedEvents>()
    for (const d of days) map.set(d.dayUtc0, [])
    for (const e of timedEvents) {
      if (map.has(e.dayUtc0)) {
        map.get(e.dayUtc0)!.push(e)
      }
    }
    return map
  }, [timedEvents, days])

  // Drag state
  const [drag, setDrag] = useState<{
    dayUtc0: number
    startMin: number
    endMin: number
  } | null>(null)

  const handleDayMouseDown = (dayUtc0: number, e: React.MouseEvent) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const y = e.clientY - rect.top + (scrollRef.current?.scrollTop ?? 0)
    const min = clampMins(snapMin(topToMins(y)))
    setDrag({ dayUtc0, startMin: min, endMin: min + 60 })
  }

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!drag) return
    const dayCol = document.querySelector(`[data-day="${drag.dayUtc0}"]`) as HTMLElement
    if (!dayCol) return
    
    const rect = dayCol.getBoundingClientRect()
    const y = e.clientY - rect.top + (scrollRef.current?.scrollTop ?? 0)
    const min = clampMins(snapMin(topToMins(y)))
    
    setDrag(prev => prev ? { ...prev, endMin: Math.max(min, prev.startMin + 15) } : null)
  }, [drag])

  const handleMouseUp = useCallback(async () => {
    if (!drag) return
    
    const { dayUtc0, startMin, endMin } = drag
    const actualStart = Math.min(startMin, endMin)
    const actualEnd = Math.max(startMin, endMin)
    
    const startUtc = new Date(dayUtc0 + actualStart * 60000).toISOString()
    const endUtc = new Date(dayUtc0 + actualEnd * 60000).toISOString()
    
    setDrag(null)
    setEditingEvent({
      title: '',
      startUtc,
      endUtc,
      allDay: false,
      tags: [],
    })
    setShowEditor(true)
  }, [drag])

  useEffect(() => {
    if (drag) {
      window.addEventListener('mousemove', handleMouseMove)
      window.addEventListener('mouseup', handleMouseUp)
      return () => {
        window.removeEventListener('mousemove', handleMouseMove)
        window.removeEventListener('mouseup', handleMouseUp)
      }
    }
  }, [drag, handleMouseMove, handleMouseUp])

  // Event handlers
  const handleEventClick = (event: NbEvent) => {
    setSelectedEvent(event)
    setEditingEvent(event)
    setShowEditor(true)
  }

  const handleSaveEvent = async () => {
    if (!editingEvent?.title) return

    try {
      if (editingEvent.id) {
        // Update existing event
        const updated = await updateEvent(editingEvent.id, {
          title: editingEvent.title,
          description: editingEvent.description,
          starts_at: editingEvent.startUtc,
          ends_at: editingEvent.endUtc,
          all_day: editingEvent.allDay,
          color: editingEvent.color,
        })
        setEvents(prev => prev.map(e => e.id === updated.id ? toNbEvent(updated) : e))
      } else {
        // Create new event
        const created = await createEvent({
          title: editingEvent.title!,
          description: editingEvent.description,
          starts_at: editingEvent.startUtc!,
          ends_at: editingEvent.endUtc!,
          all_day: editingEvent.allDay,
          color: editingEvent.color,
        })
        setEvents(prev => [...prev, toNbEvent(created)])
      }
      setShowEditor(false)
      setEditingEvent(null)
      setSelectedEvent(null)
    } catch (error) {
      console.error('Failed to save event:', error)
    }
  }

  const handleDeleteEvent = async () => {
    if (!editingEvent?.id) return
    
    if (confirm(`Delete "${editingEvent.title}"?`)) {
      try {
        await deleteEvent(editingEvent.id)
        setEvents(prev => prev.filter(e => e.id !== editingEvent.id))
        setShowEditor(false)
        setEditingEvent(null)
        setSelectedEvent(null)
      } catch (error) {
        console.error('Failed to delete event:', error)
      }
    }
  }

  const weekLabel = useMemo(() => {
    const start = days[0].dayMsk
    const end = days[6].dayMsk
    const startMonth = start.toLocaleDateString('en-US', { month: 'short' })
    const endMonth = end.toLocaleDateString('en-US', { month: 'short' })
    const year = start.getFullYear()
    
    if (startMonth === endMonth) {
      return `${startMonth} ${start.getDate()} - ${end.getDate()}, ${year}`
    }
    return `${startMonth} ${start.getDate()} - ${endMonth} ${end.getDate()}, ${year}`
  }, [days])

  return (
    <Layout>
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800 bg-zinc-900">
          <div className="flex items-center gap-4">
            <CalendarIcon className="w-5 h-5 text-zinc-400" />
            <h1 className="text-lg font-mono font-semibold text-white">{weekLabel}</h1>
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => setWeekOffset(prev => prev - 1)}
              className="p-2 hover:bg-zinc-800 rounded-lg transition-colors"
            >
              <ChevronLeft className="w-5 h-5 text-zinc-400" />
            </button>
            <button
              onClick={() => setWeekOffset(0)}
              className="px-3 py-1 text-sm font-mono text-zinc-300 hover:bg-zinc-800 rounded-lg transition-colors"
            >
              Today
            </button>
            <button
              onClick={() => setWeekOffset(prev => prev + 1)}
              className="p-2 hover:bg-zinc-800 rounded-lg transition-colors"
            >
              <ChevronRight className="w-5 h-5 text-zinc-400" />
            </button>
          </div>

          <button
            onClick={() => {
              const now = new Date()
              const startUtc = now.toISOString()
              const endUtc = new Date(now.getTime() + 60 * 60000).toISOString()
              setEditingEvent({ title: '', startUtc, endUtc, allDay: false, tags: [] })
              setShowEditor(true)
            }}
            className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-mono rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            New Event
          </button>
        </div>

        {/* Calendar Grid */}
        <div className="flex-1 overflow-hidden">
          {loading ? (
            <div className="flex items-center justify-center h-full">
              <Loader2 className="w-8 h-8 text-zinc-400 animate-spin" />
            </div>
          ) : (
            <div className="h-full flex flex-col">
              {/* All-day section + Day headers */}
              <div className="flex border-b border-zinc-700" style={{ height: ALL_DAY_HEIGHT }}>
                {/* Time gutter */}
                <div className="w-16 shrink-0 bg-zinc-900 border-r border-zinc-700" />
                
                {/* Day columns */}
                {days.map(day => (
                  <div
                    key={day.i}
                    className={`flex-1 border-r border-zinc-700 last:border-r-0 ${
                      day.isToday ? 'bg-blue-900/20' : 'bg-zinc-900'
                    }`}
                  >
                    <div className="px-2 py-1 text-center">
                      <div className={`text-sm font-mono ${day.isToday ? 'text-blue-400' : 'text-zinc-300'}`}>
                        {day.label}
                      </div>
                    </div>
                  </div>
                ))}
              </div>

              {/* Scrollable time grid */}
              <div ref={scrollRef} className="flex-1 overflow-y-auto">
                <div className="flex" style={{ minHeight: HOUR_PX * 24 }}>
                  {/* Time gutter */}
                  <div className="w-16 shrink-0 bg-zinc-900 border-r border-zinc-700">
                    {Array.from({ length: 24 }, (_, h) => (
                      <div
                        key={h}
                        className="relative border-t border-zinc-800"
                        style={{ height: HOUR_PX }}
                      >
                        <span className="absolute -top-2 left-2 text-xs text-zinc-500 font-mono bg-zinc-900 px-1">
                          {h.toString().padStart(2, '0')}:00
                        </span>
                      </div>
                    ))}
                  </div>

                  {/* Day columns */}
                  {days.map(day => {
                    const dayEvents = timedPerDay.get(day.dayUtc0) ?? []
                    
                    return (
                      <div
                        key={day.i}
                        data-day={day.dayUtc0}
                        className={`flex-1 border-r border-zinc-700 last:border-r-0 relative ${
                          day.isToday ? 'bg-blue-900/10' : ''
                        }`}
                        onMouseDown={(e) => handleDayMouseDown(day.dayUtc0, e)}
                      >
                        {/* Hour lines */}
                        {Array.from({ length: 24 }, (_, h) => (
                          <div
                            key={h}
                            className="absolute left-0 right-0 border-t border-zinc-800"
                            style={{ top: h * HOUR_PX }}
                          />
                        ))}

                        {/* Current time indicator */}
                        {day.dayUtc0 === nowInfo.dayUtc0 && (
                          <div
                            className="absolute left-0 right-0 h-0.5 bg-red-500 z-20"
                            style={{ top: minsToTop(nowInfo.min) }}
                          >
                            <div className="absolute -left-1 -top-1 w-2 h-2 rounded-full bg-red-500" />
                          </div>
                        )}

                        {/* Events */}
                        {dayEvents.map(event => (
                          <div
                            key={event.id}
                            className={`absolute left-1 right-1 px-2 py-1 rounded border cursor-pointer transition-all ${
                              selectedEvent?.id === event.id
                                ? 'bg-blue-600/90 border-blue-400 ring-1 ring-blue-400 z-20'
                                : 'bg-zinc-800/95 border-zinc-600 hover:bg-zinc-700/95 z-10'
                            }`}
                            style={{
                              top: event.top,
                              height: Math.max(event.height, 24),
                              backgroundColor: event.color ? `${event.color}dd` : undefined,
                            }}
                            onClick={(e) => {
                              e.stopPropagation()
                              handleEventClick(event)
                            }}
                          >
                            <div className="text-xs font-mono text-white truncate font-medium">
                              {event.title || 'Untitled'}
                            </div>
                            {event.height > 40 && (
                              <div className="text-[10px] text-zinc-300 font-mono">
                                {formatTime(minutesSinceMskMidnight(event.startUtc))} - {formatTime(minutesSinceMskMidnight(event.endUtc))}
                              </div>
                            )}
                          </div>
                        ))}

                        {/* Drag preview */}
                        {drag && drag.dayUtc0 === day.dayUtc0 && (
                          <div
                            className="absolute left-1 right-1 bg-blue-500/40 border border-blue-400 rounded pointer-events-none z-30"
                            style={{
                              top: minsToTop(Math.min(drag.startMin, drag.endMin)),
                              height: minsToTop(Math.abs(drag.endMin - drag.startMin)),
                            }}
                          >
                            <div className="px-2 py-1 text-xs font-mono text-blue-200">
                              {formatTime(Math.min(drag.startMin, drag.endMin))} - {formatTime(Math.max(drag.startMin, drag.endMin))}
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Event Editor Modal */}
        {showEditor && editingEvent && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
            <div className="bg-zinc-900 border border-zinc-700 rounded-lg w-full max-w-md p-6 space-y-4">
              <h2 className="text-lg font-mono font-semibold text-white">
                {editingEvent.id ? 'Edit Event' : 'New Event'}
              </h2>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">Title</label>
                <input
                  type="text"
                  value={editingEvent.title || ''}
                  onChange={(e) => setEditingEvent(prev => ({ ...prev!, title: e.target.value }))}
                  placeholder="Event title"
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  autoFocus
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">Start</label>
                  <input
                    type="datetime-local"
                    value={editingEvent.startUtc ? new Date(editingEvent.startUtc).toISOString().slice(0, 16) : ''}
                    onChange={(e) => setEditingEvent(prev => ({ ...prev!, startUtc: new Date(e.target.value).toISOString() }))}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm text-zinc-400 mb-1">End</label>
                  <input
                    type="datetime-local"
                    value={editingEvent.endUtc ? new Date(editingEvent.endUtc).toISOString().slice(0, 16) : ''}
                    onChange={(e) => setEditingEvent(prev => ({ ...prev!, endUtc: new Date(e.target.value).toISOString() }))}
                    className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">Description</label>
                <textarea
                  value={editingEvent.description || ''}
                  onChange={(e) => setEditingEvent(prev => ({ ...prev!, description: e.target.value }))}
                  placeholder="Optional description"
                  rows={3}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500 resize-none"
                />
              </div>

              <div>
                <label className="block text-sm text-zinc-400 mb-1">Color</label>
                <div className="flex gap-2">
                  {['#3b82f6', '#22c55e', '#eab308', '#f97316', '#ef4444', '#a855f7', '#ec4899'].map(color => (
                    <button
                      key={color}
                      onClick={() => setEditingEvent(prev => ({ ...prev!, color }))}
                      className={`w-8 h-8 rounded-full border-2 transition-all ${
                        editingEvent.color === color ? 'border-white scale-110' : 'border-transparent'
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                  <button
                    onClick={() => setEditingEvent(prev => ({ ...prev!, color: undefined }))}
                    className={`w-8 h-8 rounded-full border-2 bg-zinc-700 transition-all ${
                      !editingEvent.color ? 'border-white scale-110' : 'border-transparent'
                    }`}
                  />
                </div>
              </div>

              <div className="flex justify-between pt-4">
                {editingEvent.id && (
                  <button
                    onClick={handleDeleteEvent}
                    className="px-4 py-2 text-red-400 hover:bg-red-900/30 rounded-lg transition-colors"
                  >
                    Delete
                  </button>
                )}
                <div className="flex gap-2 ml-auto">
                  <button
                    onClick={() => {
                      setShowEditor(false)
                      setEditingEvent(null)
                      setSelectedEvent(null)
                    }}
                    className="px-4 py-2 text-zinc-400 hover:bg-zinc-800 rounded-lg transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSaveEvent}
                    disabled={!editingEvent.title}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-zinc-700 disabled:text-zinc-500 text-white rounded-lg transition-colors"
                  >
                    Save
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </Layout>
  )
}