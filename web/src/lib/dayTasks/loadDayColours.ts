import { useEffect, useMemo, useState } from 'react'
import { useAuthContext } from '../../contexts/AuthContext'
import { listDays } from '../../api/dayTasks'
import { dayColours, todayInZone, type Day } from './dayColour'
import { readDayPrefs, type DayPrefs } from './dayView'
import { localDayKey } from '../calendar/agenda'

/**
 * The days of a range as the server has them: one request. Off asks nothing;
 * a failed read gives no days instead of breaking the calendar. The squares
 * and the month's day-tasks variant both read this one list.
 */
export async function loadDays(
  prefs: DayPrefs,
  from: string,
  to: string,
  list: (from: string, to: string) => Promise<Day[]> = listDays,
): Promise<Day[]> {
  if (!prefs.enabled) return []
  try {
    return await list(from, to)
  } catch {
    return []
  }
}

/** The squares for a range of days: one request, the shared rule. */
export async function loadDayColours(
  prefs: DayPrefs,
  from: string,
  to: string,
  today: string,
  list: (from: string, to: string) => Promise<Day[]> = listDays,
): Promise<Record<string, string>> {
  return dayColours(await loadDays(prefs, from, to, list), today, prefs.paintBefore)
}

/**
 * A week column's day as YYYY-MM-DD. The column holds a LOCAL midnight as a
 * UTC instant (getMidnightUtcMs), so its date is read in the user's zone, not
 * sliced from the ISO string: east of UTC that slice is the day before.
 */
export function dayKey(dayUtc0: number, timeZone: string): string {
  return localDayKey(new Date(dayUtc0 + 12 * 60 * 60 * 1000), timeZone)
}

const NO_DAYS: Day[] = []

export interface DayTasksView {
  enabled: boolean
  days: Day[]
  colours: Record<string, string>
}

/** The day tasks of the visible days: the raw days and their squares, from one read. */
export function useDayTasks(from: string, to: string): DayTasksView {
  const { user } = useAuthContext()
  const tz = user?.timezone || 'Europe/Moscow'
  const { enabled, paintBefore, target } = readDayPrefs(user?.settings)
  const [days, setDays] = useState(NO_DAYS)

  useEffect(() => {
    let live = true
    void loadDays({ enabled, paintBefore, target }, from, to).then((d) => {
      if (live) setDays(d)
    })
    return () => {
      live = false
    }
  }, [enabled, paintBefore, target, from, to])

  const today = todayInZone(new Date(), tz)
  const colours = useMemo(() => dayColours(days, today, paintBefore), [days, today, paintBefore])
  return { enabled, days, colours }
}

/** The squares for the visible days of the calendar. */
export function useDayColours(from: string, to: string): Record<string, string> {
  return useDayTasks(from, to).colours
}
