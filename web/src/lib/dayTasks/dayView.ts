/**
 * Day tasks page logic, kept out of the component so it can be tested
 * (spec docs/superpowers/specs/2026-09-24-web-day-tasks-design.md).
 */
import { ApiError } from '../../api/client'
import type { UserSettings } from '../../api/auth'

export interface DayPrefs {
  enabled: boolean
  target: number
  paintBefore: boolean
}

/** The bot's rule: no key = on, 5, not painting; a target outside 3–7 is ignored. */
export function readDayPrefs(s: Partial<UserSettings> | undefined | null): DayPrefs {
  const t = s?.day_tasks_target
  return {
    enabled: s?.day_tasks_enabled !== false,
    target: typeof t === 'number' && t >= 3 && t <= 7 ? t : 5,
    paintBefore: s?.day_tasks_paint_before === true,
  }
}

/** YYYY-MM-DD plus n days, on the calendar: no clock, no zone. */
export function shiftDay(day: string, n: number): string {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, d + n)).toISOString().slice(0, 10)
}

/** Seven days from `monday`. */
export function weekDays(monday: string): string[] {
  return Array.from({ length: 7 }, (_, i) => shiftDay(monday, i))
}

export type DayWhen = 'past' | 'today' | 'future'

export function dayWhen(day: string, today: string): DayWhen {
  if (day < today) return 'past'
  return day === today ? 'today' : 'future'
}

/** Spec §5: today until 12:00, the future always, the past never. The server enforces it too. */
export function canRemove(day: string, today: string, hour: number): boolean {
  const when = dayWhen(day, today)
  return when === 'future' || (when === 'today' && hour < 12)
}

/** The hour of `now` in the person's zone. */
export function hourInZone(now: Date, timeZone: string): number {
  const h = (tz: string) =>
    Number(new Intl.DateTimeFormat('en-GB', { timeZone: tz, hour: '2-digit', hourCycle: 'h23' }).format(now))
  try {
    return h(timeZone)
  } catch {
    return h('Europe/Moscow')
  }
}

/** The i18n key (daytasks namespace) for a failed day-tasks request. */
export function errorKey(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'TOO_LATE':
        return 'errors.tooLate'
      case 'NOT_OPEN':
        return 'errors.notOpen'
      case 'TASK_NOT_FOUND':
        return 'errors.notFound'
    }
  }
  return 'errors.generic'
}
