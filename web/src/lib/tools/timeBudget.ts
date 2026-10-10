/**
 * «Бюджет времени» (Denis 25.09, docs/tasks-web-cleanup.md 4.10): how a work
 * day splits between kinds of work. The day's length comes from the work hours
 * in the account (⚙️ Рабочие часы), not from a number typed into this tool, and
 * the split is stored in the account (setting `time_budget`) instead of one
 * browser's localStorage. The decorative "week" drawn from the same numbers is
 * gone: it showed nothing the day did not.
 */

export interface TimeCategory {
  id: string
  labelKey: string // i18n key for the built-in ones
  customLabel: string // used when labelKey is ''
  hours: number
  color: string // Tailwind bg- colour class
  isCustom: boolean
}

function minutes(hhmm: string | undefined): number | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec(hhmm ?? '')
  if (!m) return null
  const h = Number(m[1])
  const min = Number(m[2])
  return h <= 23 && min <= 59 ? h * 60 + min : null
}

/** Hours in the work day from `work_start` / `work_end`; null when they make no span. */
export function workHoursPerDay(start: string | undefined, end: string | undefined): number | null {
  const s = minutes(start)
  const e = minutes(end)
  if (s === null || e === null || e <= s) return null
  return Math.round(((e - s) / 60) * 100) / 100
}

/** Number of work days chosen in the account; five when none are set. */
export function workDayCount(days: string[] | undefined): number {
  return days && days.length > 0 && days.length <= 7 ? days.length : 5
}

/** Categories stored in the account, or null when there are none or they are malformed. */
export function readBudgetCategories(raw: unknown): TimeCategory[] | null {
  if (!Array.isArray(raw) || raw.length === 0) return null
  const ok = raw.every(
    (c) =>
      c &&
      typeof c === 'object' &&
      typeof (c as TimeCategory).id === 'string' &&
      typeof (c as TimeCategory).labelKey === 'string' &&
      typeof (c as TimeCategory).customLabel === 'string' &&
      typeof (c as TimeCategory).hours === 'number' &&
      typeof (c as TimeCategory).color === 'string' &&
      typeof (c as TimeCategory).isCustom === 'boolean'
  )
  return ok ? (raw as TimeCategory[]) : null
}
