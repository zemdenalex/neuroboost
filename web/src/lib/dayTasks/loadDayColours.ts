import { useEffect, useState } from 'react'
import { useAuthContext } from '../../contexts/AuthContext'
import { listDays } from '../../api/dayTasks'
import { dayColours, todayInZone, type Day } from './dayColour'
import { readDayPrefs, type DayPrefs } from './dayView'

/**
 * The squares for a range of days: one request, the shared rule. Off asks
 * nothing; a failed read draws no squares instead of breaking the calendar.
 */
export async function loadDayColours(
  prefs: DayPrefs,
  from: string,
  to: string,
  today: string,
  list: (from: string, to: string) => Promise<Day[]> = listDays,
): Promise<Record<string, string>> {
  if (!prefs.enabled) return {}
  try {
    return dayColours(await list(from, to), today, prefs.paintBefore)
  } catch {
    return {}
  }
}

/** A week column's midnight (UTC timestamp) as its YYYY-MM-DD. */
export function dayKey(dayUtc0: number): string {
  return new Date(dayUtc0).toISOString().slice(0, 10)
}

const NONE: Record<string, string> = {}

/** The squares for the visible days of the calendar. */
export function useDayColours(from: string, to: string): Record<string, string> {
  const { user } = useAuthContext()
  const tz = user?.timezone || 'Europe/Moscow'
  const { enabled, paintBefore, target } = readDayPrefs(user?.settings)
  const [colours, setColours] = useState(NONE)

  useEffect(() => {
    let live = true
    void loadDayColours({ enabled, paintBefore, target }, from, to, todayInZone(new Date(), tz)).then((c) => {
      if (live) setColours(c)
    })
    return () => {
      live = false
    }
  }, [enabled, paintBefore, target, from, to, tz])

  return colours
}
