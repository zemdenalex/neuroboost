import { useState, useEffect, useCallback, useMemo } from 'react'
import type React from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  Loader2,
} from 'lucide-react'
import { getWeekPlan, WeekPlan, PlanningTask, PlanningEvent } from '../../api/planning'
import { scheduleTask } from '../../api/tasks'
import { toLocalDateKey, startOfWeek, dateLocale } from '../../utils/date'
import { UnscheduledList } from './UnscheduledList'
import { CapacityMeter } from './CapacityMeter'
import { WeekOverviewGrid } from './WeekOverviewGrid'

export default function Planning() {
  const { t, i18n } = useTranslation('planning')

  // weekOffset: 0 = current week, -1 = last week, etc.
  const [weekOffset, setWeekOffset] = useState(0)
  const [plan, setPlan] = useState<WeekPlan | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Compute the Monday for this week offset
  const monday = useMemo(() => {
    const m = startOfWeek(new Date())
    m.setDate(m.getDate() + weekOffset * 7)
    return m
  }, [weekOffset])

  const sunday = useMemo(() => {
    const s = new Date(monday)
    s.setDate(monday.getDate() + 6)
    return s
  }, [monday])

  const fetchPlan = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await getWeekPlan(toLocalDateKey(monday))
      setPlan(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('error.load'))
    } finally {
      setLoading(false)
    }
  }, [monday, t])

  useEffect(() => {
    fetchPlan()
  }, [fetchPlan])

  // Format week range label
  const weekLabel = useMemo(() => {
    const locale = dateLocale(i18n.language)
    const startLabel = monday.toLocaleDateString(locale, { month: 'short', day: 'numeric' })
    const endLabel = sunday.toLocaleDateString(locale, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
    return `${startLabel} – ${endLabel}`
  }, [monday, sunday, i18n.language])

  // Build 7-day array with events bucketed by local date
  const weekDays = useMemo(() => {
    if (!plan) return []
    const buckets: Record<string, PlanningEvent[]> = {}
    const hours: Record<string, number> = {}
    for (const ev of plan.weekEvents) {
      const start = new Date(ev.starts_at)
      const key = toLocalDateKey(start)
      if (!buckets[key]) buckets[key] = []
      buckets[key].push(ev)
      if (!ev.all_day) {
        const duration = (new Date(ev.ends_at).getTime() - start.getTime()) / 3600000
        hours[key] = (hours[key] || 0) + duration
      }
    }
    const days: { date: Date; scheduledHours: number; events: PlanningEvent[] }[] = []
    for (let i = 0; i < 7; i++) {
      const d = new Date(monday)
      d.setDate(monday.getDate() + i)
      const key = toLocalDateKey(d)
      days.push({
        date: d,
        scheduledHours: hours[key] || 0,
        events: buckets[key] || [],
      })
    }
    return days
  }, [plan, monday])

  const handleTaskDragStart = useCallback(
    (e: React.DragEvent, task: PlanningTask) => {
      e.dataTransfer.setData('application/json', JSON.stringify(task))
      e.dataTransfer.effectAllowed = 'copy'
    },
    [],
  )

  const handleTaskDrop = useCallback(
    async (task: PlanningTask, date: Date) => {
      // Schedule at 09:00 local time on the dropped date
      const start = new Date(date)
      start.setHours(9, 0, 0, 0)
      // Clamp to a 15-min minimum so a task with no/zero estimate still schedules a
      // real block instead of a zero-length event (matches the Calendar scheduler).
      const durationMin = Math.max(15, task.estimated_minutes ?? 60)
      const end = new Date(start.getTime() + durationMin * 60000)
      try {
        await scheduleTask(task.id, {
          starts_at: start.toISOString(),
          ends_at: end.toISOString(),
          all_day: false,
        })
        await fetchPlan()
      } catch (err) {
        console.error('Failed to schedule task', err)
        setError(err instanceof Error ? err.message : t('error.schedule'))
      }
    },
    [fetchPlan, t],
  )

  const isCurrentWeek = weekOffset === 0

  return (
    <div className="flex flex-col h-full min-h-0 gap-3 p-4 md:p-6">
      {/* Header */}
      <header className="flex items-center justify-between gap-3 shrink-0">
        <div className="min-w-0">
          <h1 className="text-xl font-mono text-white truncate">{t('title')}</h1>
          <p className="text-sm text-zinc-400 font-mono truncate">{weekLabel}</p>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => setWeekOffset((w) => w - 1)}
            aria-label={t('prevWeek')}
            className="p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <button
            onClick={() => setWeekOffset(0)}
            className={`px-3 py-1.5 font-mono text-sm rounded-lg transition-colors ${
              isCurrentWeek
                ? 'bg-blue-600 text-white'
                : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700'
            }`}
          >
            {t('today')}
          </button>
          <button
            onClick={() => setWeekOffset((w) => w + 1)}
            aria-label={t('nextWeek')}
            className="p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800 rounded-lg transition-colors"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </header>

      {/* Loading / error states */}
      {loading && (
        <div className="flex items-center gap-2 text-zinc-400 text-sm">
          <Loader2 className="w-4 h-4 animate-spin" />
          <span>{t('status.loading', { ns: 'common' })}</span>
        </div>
      )}
      {error && !loading && (
        <div className="flex flex-col items-start gap-2 text-red-400 text-sm">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4" />
            <span>{error}</span>
          </div>
          <button
            onClick={fetchPlan}
            className="px-3 py-1.5 text-xs bg-zinc-800 hover:bg-zinc-700 text-zinc-200 rounded-lg transition-colors"
          >
            {t('action.back', { ns: 'common' })}
          </button>
        </div>
      )}

      {/* Two-pane body */}
      {plan && !loading && !error && (
        <div className="flex-1 flex flex-col lg:flex-row gap-3 min-h-0">
          <div className="lg:w-1/3 lg:min-w-[280px] max-h-[40vh] lg:max-h-none">
            <UnscheduledList
              tasks={plan.unscheduledTasks}
              scheduledHours={plan.scheduledHours}
              onDragStart={handleTaskDragStart}
            />
          </div>
          <div className="flex-1 flex flex-col gap-3 min-h-0">
            <CapacityMeter
              scheduledHours={plan.scheduledHours}
              availableHours={plan.availableHours}
            />
            <WeekOverviewGrid days={weekDays} onTaskDrop={handleTaskDrop} />
          </div>
        </div>
      )}
    </div>
  )
}
