import type { NbEvent } from '../../types'
import { localDayKey } from './agenda'

/**
 * What a month cell holds (spec V003-20260924-arc-web-month-view, R5).
 *
 * Days are the user's calendar days, as in the week: an event after 21:00 UTC
 * belongs to tomorrow in Moscow. A multi-day event is drawn on each of its
 * days; `first` marks the day it starts, the only one that shows its time.
 */

export interface MonthItem {
  event: NbEvent
  first: boolean
  /** The day it ends on (for a multi-day event, where its end time shows). */
  last: boolean
}

const MINUTE_MS = 60 * 1000

/** Each grid day (YYYY-MM-DD) mapped to its events: all-day first, then by start. */
export function eventsByDay(events: NbEvent[], days: string[], timezone: string): Record<string, MonthItem[]> {
  const out: Record<string, MonthItem[]> = {}
  for (const d of days) out[d] = []
  const first = days[0]
  const last = days[days.length - 1]

  for (const event of events) {
    const start = Date.parse(event.startsAt)
    if (Number.isNaN(start)) continue
    const endRaw = Date.parse(event.endsAt)
    // An end is exclusive: an event ending at midnight is not on the next day.
    const endIncl = Number.isNaN(endRaw) || endRaw <= start ? start : endRaw - MINUTE_MS
    const startDay = localDayKey(new Date(start), timezone)
    const endDay = localDayKey(new Date(endIncl), timezone)
    if (endDay < first || startDay > last) continue
    for (const d of days) {
      if (d >= startDay && d <= endDay) out[d].push({ event, first: d === startDay, last: d === endDay })
    }
  }

  for (const d of days) {
    out[d].sort((a, b) => {
      const allDay = Number(Boolean(b.event.allDay)) - Number(Boolean(a.event.allDay))
      if (allDay !== 0) return allDay
      return Date.parse(a.event.startsAt) - Date.parse(b.event.startsAt)
    })
  }
  return out
}

/**
 * The rows a cell shows and how many are left over. "+1 more" would take the
 * same row as the item it hides, so one extra item is shown instead.
 */
export function cellRows<T>(items: T[], max: number): { shown: T[]; more: number } {
  if (items.length <= max + 1) return { shown: items, more: 0 }
  return { shown: items.slice(0, max), more: items.length - max }
}
