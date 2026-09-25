import type { PlanningEvent } from '../../api/planning'
import { toLocalDateKey } from '../../utils/date'

export interface PlanningDay {
  date: Date
  scheduledHours: number
  events: PlanningEvent[]
}

/**
 * The Planning page's seven columns from the week's events: each on its local
 * day, in time order (the API returns a series' occurrences together, not by
 * time), hours counted for timed events only.
 */
export function weekDays(events: PlanningEvent[], monday: Date): PlanningDay[] {
  const buckets: Record<string, PlanningEvent[]> = {}
  const hours: Record<string, number> = {}
  const sorted = [...events].sort((a, b) => new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime())
  for (const ev of sorted) {
    const start = new Date(ev.starts_at)
    const key = toLocalDateKey(start)
    ;(buckets[key] ??= []).push(ev)
    if (!ev.all_day) {
      hours[key] = (hours[key] || 0) + (new Date(ev.ends_at).getTime() - start.getTime()) / 3600000
    }
  }
  const days: PlanningDay[] = []
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday)
    d.setDate(monday.getDate() + i)
    const key = toLocalDateKey(d)
    days.push({ date: d, scheduledHours: hours[key] || 0, events: buckets[key] || [] })
  }
  return days
}
