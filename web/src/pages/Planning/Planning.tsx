import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  ChevronLeft,
  ChevronRight,
  CalendarDays,
  ListTodo,
  AlertCircle,
  Loader2,
  CheckCircle2,
  Clock,
  ExternalLink,
} from 'lucide-react'
import { getWeekPlan, WeekPlan, PlanningTask } from '../../api/planning'
import { PRIORITY_COLORS, PRIORITY_LABELS } from '../../api/tasks'

// Compute the Monday of the week that contains `date`
function getMondayOf(date: Date): Date {
  const d = new Date(date)
  const day = d.getDay() // 0 = Sun, 1 = Mon, ...
  const daysToMonday = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + daysToMonday)
  d.setHours(0, 0, 0, 0)
  return d
}

// Format a Date to YYYY-MM-DD for API
function toDateString(date: Date): string {
  return date.toISOString().slice(0, 10)
}

// Priority emoji map (priority 1 = highest = Emergency)
const PRIORITY_EMOJI: Record<number, string> = {
  0: '⬜',
  1: '🔴',
  2: '🟠',
  3: '🟡',
  4: '🟢',
  5: '🔵',
}

const DAY_KEYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const

export default function Planning() {
  const { t, i18n } = useTranslation('planning')
  const navigate = useNavigate()

  // weekOffset: 0 = current week, -1 = last week, etc.
  const [weekOffset, setWeekOffset] = useState(0)
  const [plan, setPlan] = useState<WeekPlan | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Compute the Monday for this week offset
  const monday = getMondayOf(new Date())
  monday.setDate(monday.getDate() + weekOffset * 7)
  const sunday = new Date(monday)
  sunday.setDate(monday.getDate() + 6)

  const fetchPlan = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await getWeekPlan(toDateString(monday))
      setPlan(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('error.load'))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [weekOffset])

  useEffect(() => {
    fetchPlan()
  }, [fetchPlan])

  // Format week range label: "Apr 7 – Apr 13, 2026"
  const weekLabel = (() => {
    const locale = i18n.language
    const startLabel = monday.toLocaleDateString(locale, { month: 'short', day: 'numeric' })
    const endLabel = sunday.toLocaleDateString(locale, { month: 'short', day: 'numeric', year: 'numeric' })
    return `${startLabel} – ${endLabel}`
  })()

  const isCurrentWeek = weekOffset === 0

  // Build per-day event counts for the week overview
  const dayEvents = (() => {
    if (!plan) return []
    const days: Array<{ label: string; date: Date; events: number; hours: number }> = []
    for (let i = 0; i < 7; i++) {
      const day = new Date(monday)
      day.setDate(monday.getDate() + i)
      const dayStart = day.getTime()
      const dayEnd = dayStart + 24 * 60 * 60 * 1000

      let eventCount = 0
      let hours = 0
      for (const ev of plan.weekEvents) {
        const evStart = new Date(ev.starts_at).getTime()
        if (evStart >= dayStart && evStart < dayEnd) {
          eventCount++
          if (!ev.all_day) {
            const evEnd = new Date(ev.ends_at).getTime()
            hours += (evEnd - evStart) / (1000 * 60 * 60)
          }
        }
      }

      days.push({
        label: t(`day.${DAY_KEYS[i]}`),
        date: day,
        events: eventCount,
        hours,
      })
    }
    return days
  })()

  const scheduledPct = plan ? Math.min(100, Math.round((plan.scheduledHours / plan.availableHours) * 100)) : 0

  const capacityColor =
    scheduledPct >= 90
      ? 'bg-red-500'
      : scheduledPct >= 70
      ? 'bg-yellow-500'
      : 'bg-emerald-500'

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800 shrink-0">
        <div className="flex items-center gap-2">
          <CalendarDays className="text-blue-400 shrink-0" size={20} />
          <div>
            <h1 className="text-sm font-semibold text-zinc-100">{t('title')}</h1>
            <p className="text-xs text-zinc-500 hidden sm:block">{weekLabel}</p>
          </div>
        </div>

        {/* Week navigation */}
        <div className="flex items-center gap-1">
          <button
            onClick={() => setWeekOffset(w => w - 1)}
            aria-label={t('prevWeek')}
            className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors"
          >
            <ChevronLeft size={18} />
          </button>

          <button
            onClick={() => setWeekOffset(0)}
            className={`px-3 py-1 text-xs rounded-lg font-medium transition-colors ${
              isCurrentWeek
                ? 'bg-blue-600 text-white'
                : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-100'
            }`}
          >
            {t('today')}
          </button>

          <button
            onClick={() => setWeekOffset(w => w + 1)}
            aria-label={t('nextWeek')}
            className="p-1.5 rounded-lg text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 transition-colors"
          >
            <ChevronRight size={18} />
          </button>
        </div>
      </div>

      {/* Week label on mobile */}
      <div className="sm:hidden px-4 py-1.5 border-b border-zinc-800 text-xs text-zinc-500">
        {weekLabel}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {loading ? (
          <div className="flex items-center justify-center h-full text-zinc-500">
            <Loader2 className="animate-spin mr-2" size={18} />
            <span className="text-sm">{t('status.loading', { ns: 'common' })}</span>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full gap-3 text-zinc-400 px-4">
            <AlertCircle className="text-red-400" size={32} />
            <p className="text-sm text-center">{error}</p>
            <button
              onClick={fetchPlan}
              className="px-4 py-2 text-sm bg-zinc-800 hover:bg-zinc-700 rounded-lg transition-colors"
            >
              {t('action.back', { ns: 'common' })}
            </button>
          </div>
        ) : plan ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-0 h-full">
            {/* Left column: Unscheduled Tasks */}
            <div className="border-b lg:border-b-0 lg:border-r border-zinc-800 overflow-auto">
              <div className="sticky top-0 bg-zinc-950 border-b border-zinc-800 px-4 py-2.5 z-10">
                <div className="flex items-center gap-2">
                  <ListTodo size={15} className="text-zinc-400" />
                  <span className="text-xs font-semibold text-zinc-300 uppercase tracking-wider">
                    {t('unscheduledTasks')}
                  </span>
                  {plan.unscheduledTasks.length > 0 && (
                    <span className="ml-auto text-xs bg-zinc-800 text-zinc-400 px-1.5 py-0.5 rounded-full">
                      {plan.unscheduledTasks.length}
                    </span>
                  )}
                </div>
              </div>

              {plan.unscheduledTasks.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 px-4 text-center gap-2">
                  <CheckCircle2 className="text-emerald-500" size={32} />
                  <p className="text-sm font-medium text-zinc-300">{t('noUnscheduledTasks')}</p>
                  <p className="text-xs text-zinc-500">{t('noUnscheduledTasksHint')}</p>
                </div>
              ) : (
                <ul className="divide-y divide-zinc-800/50">
                  {plan.unscheduledTasks.map(task => (
                    <TaskRow key={task.id} task={task} t={t} />
                  ))}
                </ul>
              )}
            </div>

            {/* Right column: Week Overview */}
            <div className="overflow-auto">
              {/* Capacity meter */}
              <div className="sticky top-0 bg-zinc-950 border-b border-zinc-800 px-4 py-2.5 z-10">
                <div className="flex items-center gap-2 mb-2">
                  <Clock size={15} className="text-zinc-400" />
                  <span className="text-xs font-semibold text-zinc-300 uppercase tracking-wider">
                    {t('capacity')}
                  </span>
                  <span className="ml-auto text-xs text-zinc-400">
                    {t('scheduledOf', {
                      scheduled: plan.scheduledHours.toFixed(1),
                      available: plan.availableHours,
                    })}
                  </span>
                </div>
                <div className="h-1.5 bg-zinc-800 rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all duration-500 ${capacityColor}`}
                    style={{ width: `${scheduledPct}%` }}
                  />
                </div>
                <p className="text-right text-xs text-zinc-600 mt-0.5">{scheduledPct}%</p>
              </div>

              {/* Day breakdown */}
              <div className="px-4 py-3">
                <div className="flex items-center gap-2 mb-3">
                  <CalendarDays size={15} className="text-zinc-400" />
                  <span className="text-xs font-semibold text-zinc-300 uppercase tracking-wider">
                    {t('weekOverview')}
                  </span>
                  <button
                    onClick={() => navigate('/calendar')}
                    className="ml-auto flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300 transition-colors"
                  >
                    <ExternalLink size={12} />
                    {t('goToCalendar')}
                  </button>
                </div>

                <div className="space-y-2">
                  {dayEvents.map((day, i) => {
                    const isWeekend = i >= 5
                    const isToday =
                      weekOffset === 0 &&
                      day.date.toDateString() === new Date().toDateString()
                    const maxHours = plan.availableHours / 5 // 8h per day

                    return (
                      <div key={day.label} className="flex items-center gap-3">
                        <span
                          className={`w-7 text-xs font-medium shrink-0 ${
                            isToday
                              ? 'text-blue-400'
                              : isWeekend
                              ? 'text-zinc-600'
                              : 'text-zinc-400'
                          }`}
                        >
                          {day.label}
                        </span>

                        <div className="flex-1 h-4 bg-zinc-800 rounded-sm overflow-hidden">
                          {day.hours > 0 && (
                            <div
                              className={`h-full rounded-sm transition-all duration-300 ${
                                isWeekend ? 'bg-zinc-600' : 'bg-blue-600'
                              }`}
                              style={{ width: `${Math.min(100, (day.hours / maxHours) * 100)}%` }}
                            />
                          )}
                        </div>

                        <div className="shrink-0 text-right min-w-0 flex items-center gap-1.5 justify-end">
                          {day.hours > 0 ? (
                            <span className="text-xs text-zinc-400">
                              {day.hours.toFixed(1)}h
                            </span>
                          ) : (
                            <span className="text-xs text-zinc-700">—</span>
                          )}
                          {day.events > 0 && (
                            <span className="text-xs text-zinc-500">
                              {t('events_one', { count: day.events })
                                .replace('_one', '')
                                .replace('_other', '')}
                              &nbsp;({day.events})
                            </span>
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>

                {/* Event list */}
                {plan.weekEvents.length > 0 && (
                  <div className="mt-6">
                    <h3 className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                      {t('weekOverview')}
                    </h3>
                    <ul className="space-y-1.5">
                      {plan.weekEvents.map(ev => {
                        const start = new Date(ev.starts_at)
                        const locale = i18n.language
                        const timeLabel = ev.all_day
                          ? start.toLocaleDateString(locale, { weekday: 'short', month: 'short', day: 'numeric' })
                          : start.toLocaleString(locale, {
                              weekday: 'short',
                              month: 'short',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                            })
                        return (
                          <li
                            key={ev.id}
                            className="flex items-start gap-2 py-1 text-sm"
                          >
                            <span
                              className="mt-1.5 w-2 h-2 rounded-full shrink-0"
                              style={{ backgroundColor: ev.color ?? '#3b82f6' }}
                            />
                            <div className="min-w-0">
                              <p className="text-zinc-200 truncate">{ev.title}</p>
                              <p className="text-xs text-zinc-500">{timeLabel}</p>
                            </div>
                          </li>
                        )
                      })}
                    </ul>
                  </div>
                )}

                {plan.weekEvents.length === 0 && (
                  <p className="text-xs text-zinc-600 mt-6 text-center">{t('noEvents')}</p>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )
}

// ─── TaskRow subcomponent ──────────────────────────────────────────────────────

interface TaskRowProps {
  task: PlanningTask
  t: ReturnType<typeof useTranslation<'planning'>>['t']
}

function TaskRow({ task, t }: TaskRowProps) {
  const priorityColor = PRIORITY_COLORS[task.priority] ?? PRIORITY_COLORS[3]
  const priorityLabel = PRIORITY_LABELS[task.priority] ?? 'Unknown'
  const emoji = PRIORITY_EMOJI[task.priority] ?? '⬜'

  const estimatedLabel = (() => {
    if (!task.estimated_minutes) return null
    if (task.estimated_minutes < 60) return t('estimatedTime', { minutes: task.estimated_minutes })
    return t('estimatedHours', { hours: (task.estimated_minutes / 60).toFixed(1) })
  })()

  const dueDateLabel = (() => {
    if (!task.due_date) return null
    const d = new Date(task.due_date)
    return t('dueOn', { date: d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) })
  })()

  return (
    <li className="flex items-start gap-3 px-4 py-3 hover:bg-zinc-900/50 transition-colors">
      {/* Priority dot */}
      <span className="mt-0.5 shrink-0 text-base leading-none" title={priorityLabel}>
        {emoji}
      </span>

      <div className="flex-1 min-w-0">
        <p className="text-sm text-zinc-200 truncate">{task.title}</p>
        <div className="flex items-center gap-2 mt-0.5 flex-wrap">
          <span
            className={`text-xs px-1.5 py-0.5 rounded text-white font-medium ${priorityColor}`}
          >
            {priorityLabel}
          </span>
          {estimatedLabel && (
            <span className="text-xs text-zinc-500 flex items-center gap-0.5">
              <Clock size={10} />
              {estimatedLabel}
            </span>
          )}
          {dueDateLabel && (
            <span className="text-xs text-zinc-500">{dueDateLabel}</span>
          )}
        </div>
      </div>
    </li>
  )
}
