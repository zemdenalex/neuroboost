import { localDateTimeToUtc } from './editor.utils'

/**
 * Dates of an all-day event in the editor (gap list row 9, 26.09). Stored as
 * the bot stores them (draftflow.go draftBounds): local midnight of the first
 * day to the local midnight AFTER the last day. The form shows first and last
 * day, which is how a person says «с 14.10 по 29.10».
 */

const shiftDay = (date: string, days: number): string => {
  const [y, m, d] = date.split('-').map(Number)
  const t = new Date(Date.UTC(y, m - 1, d + days))
  return t.toISOString().slice(0, 10)
}

/** The last day of an all-day event from its stored end (date + time). */
export function allDayLastDay(startDate: string, endDate: string, endTime: string): string {
  if (endTime === '00:00' && endDate > startDate) return shiftDay(endDate, -1)
  return endDate
}

/** Midnight of the first day to the midnight after the last, in the zone. */
export function allDayBounds(startDate: string, lastDay: string, timezone: string): { startsAt: string; endsAt: string } {
  const last = lastDay < startDate ? startDate : lastDay
  return {
    startsAt: localDateTimeToUtc(startDate, '00:00', timezone).toISOString(),
    endsAt: localDateTimeToUtc(shiftDay(last, 1), '00:00', timezone).toISOString(),
  }
}
