import type { SessionRecord } from './types'

export const HISTORY_KEY = 'nb-pomodoro-history'

/** Local calendar-day key (YYYY-MM-DD) in the runtime timezone. */
export function localDayKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export interface TodaySummary {
  count: number
  minutes: number
}

/**
 * Summarizes completed WORK blocks for the LOCAL calendar day of `now`. Each
 * record is grouped by its `completedAt` timestamp converted to the local day,
 * so the "Today" stat rolls over at the user's midnight — not UTC midnight.
 */
export function summarizeToday(history: SessionRecord[], now: Date): TodaySummary {
  const todayKey = localDayKey(now)
  let count = 0
  let seconds = 0
  for (const r of history) {
    if (r.mode !== 'work') continue
    const ts = new Date(r.completedAt)
    if (Number.isNaN(ts.getTime())) continue
    if (localDayKey(ts) !== todayKey) continue
    count++
    seconds += r.durationSeconds
  }
  return { count, minutes: Math.floor(seconds / 60) }
}

/** Reads the persisted pomodoro session history; returns [] on any failure. */
export function loadHistory(): SessionRecord[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    const list = raw ? JSON.parse(raw) : []
    return Array.isArray(list) ? list : []
  } catch {
    return []
  }
}
