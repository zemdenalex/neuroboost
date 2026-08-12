import type React from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import type { PlanningTask, PlanningEvent } from '../../api/planning'
import { toLocalDateKey, dateLocale } from '../../utils/date'

export interface DayData {
  date: Date
  scheduledHours: number
  events: PlanningEvent[]
}

interface Props {
  days: DayData[]
  onTaskDrop: (task: PlanningTask, date: Date) => void
}

export function WeekOverviewGrid({ days, onTaskDrop }: Props) {
  const { t, i18n } = useTranslation('planning')
  const navigate = useNavigate()

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
  }

  const handleDrop = (e: React.DragEvent, date: Date) => {
    e.preventDefault()
    const data = e.dataTransfer.getData('application/json')
    if (!data) return
    try {
      const task = JSON.parse(data) as PlanningTask
      onTaskDrop(task, date)
    } catch {
      // ignore malformed drop payload
    }
  }

  const locale = dateLocale(i18n.language)

  return (
    // One column below xl, seven from xl (1280px) up.
    //
    // Seven columns on a 375px screen gave each day ~45px, rendering an event
    // chip into a 21px box: 209px of text showing as "H…", so the column could
    // not answer the only question it exists for — what is on this day.
    //
    // 🔴 The breakpoint is xl and not md because the tablet range measured WORSE
    // than mobile: 75px per chip at 768 and only 63px at 1024. This page puts the
    // unscheduled-tasks panel beside the grid, so widening the viewport does not
    // widen the columns proportionally — 1024 is narrower per column than 768.
    // A row per day keeps the week in order and gives the chip the full width.
    <div className="flex-1 grid grid-cols-1 xl:grid-cols-7 xl:auto-rows-fr gap-2 min-h-0 overflow-y-auto xl:overflow-visible">
      {days.map((day, i) => {
        const dayName = day.date.toLocaleDateString(locale, { weekday: 'short' })
        const dayNum = day.date.getDate()
        const isToday = new Date().toDateString() === day.date.toDateString()
        return (
          <div
            key={day.date.toISOString()}
            data-hint={i === 0 ? 'planning.day' : undefined}
            role="button"
            tabIndex={0}
            aria-label={`${dayName} ${dayNum}`}
            onDragOver={handleDragOver}
            onDrop={(e) => handleDrop(e, day.date)}
            onClick={() =>
              navigate(`/calendar?date=${toLocalDateKey(day.date)}`)
            }
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                navigate(`/calendar?date=${toLocalDateKey(day.date)}`)
              }
            }}
            className={`flex flex-col p-2 bg-zinc-900 border rounded-lg cursor-pointer hover:border-blue-600 transition-colors ${
              isToday ? 'border-blue-500' : 'border-zinc-800'
            }`}
          >
            <div className="flex items-baseline gap-3 mb-1 xl:mb-2">
              {/* Takes the full width in the seven-column layout, so the day
                  number stays hard right exactly as before; in the row layout it
                  yields the right edge to the hours, which move up onto this line
                  to save vertical space. */}
              <div className="flex-1 flex items-baseline justify-between gap-2 min-w-0">
                <span className="text-xs uppercase tracking-wider text-zinc-500">
                  {dayName}
                </span>
                <span
                  className={`text-lg font-mono ${
                    isToday ? 'text-blue-400' : 'text-zinc-300'
                  }`}
                >
                  {dayNum}
                </span>
              </div>
              <span className="xl:hidden text-xs text-zinc-400 font-mono tabular-nums shrink-0">
                {day.scheduledHours.toFixed(1)}
                {t('hoursShort')}
              </span>
            </div>
            <div className="hidden xl:block text-xs text-zinc-400 mb-2 font-mono tabular-nums">
              {day.scheduledHours.toFixed(1)}
              {t('hoursShort')}
            </div>
            <ul className="flex-1 space-y-1 overflow-hidden">
              {day.events.slice(0, 3).map((ev) => (
                <li
                  key={ev.id}
                  className="text-xs text-zinc-300 truncate px-1.5 py-0.5 rounded bg-zinc-800"
                  title={ev.title}
                >
                  {ev.title}
                </li>
              ))}
              {day.events.length > 3 && (
                <li className="text-xs text-zinc-500">
                  {t('moreEvents', { count: day.events.length - 3 })}
                </li>
              )}
            </ul>
          </div>
        )
      })}
    </div>
  )
}
