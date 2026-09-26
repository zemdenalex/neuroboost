/**
 * Week grouping and filters of the Reflections page, moved out of it to be
 * tested (docs/agents/queue.md: tests for pages without any).
 */

export type FilterRange = 'week' | 'month' | 'all'

/** Returns the ISO week number (1-53) and year for a date */
export function isoWeekKey(dateStr: string): string {
  const d = new Date(dateStr)
  // Thursday of the current week determines the ISO year
  const thursday = new Date(d)
  thursday.setDate(d.getDate() - ((d.getDay() + 6) % 7) + 3)
  const firstThursday = new Date(thursday.getFullYear(), 0, 4)
  const weekNum =
    1 +
    Math.round(
      ((thursday.getTime() - firstThursday.getTime()) / 86400000 -
        3 +
        ((firstThursday.getDay() + 6) % 7)) /
        7,
    )
  return `${thursday.getFullYear()}-W${String(weekNum).padStart(2, '0')}`
}

/** Returns the Monday of the ISO week for display */
export function weekStart(weekKey: string): Date {
  const [yearStr, weekStr] = weekKey.split('-W')
  const year = parseInt(yearStr, 10)
  const week = parseInt(weekStr, 10)
  // Jan 4 is always in week 1
  const jan4 = new Date(year, 0, 4)
  const monday = new Date(jan4)
  monday.setDate(jan4.getDate() - ((jan4.getDay() + 6) % 7) + (week - 1) * 7)
  return monday
}

/** Average of non-null values, returns null if none */
export function average(values: (number | null)[]): number | null {
  const nums = values.filter((v): v is number => v !== null)
  if (nums.length === 0) return null
  return Math.round((nums.reduce((a, b) => a + b, 0) / nums.length) * 10) / 10
}

export function isWithinRange(createdAt: string, range: FilterRange, now: Date = new Date()): boolean {
  if (range === 'all') return true
  const d = new Date(createdAt)
  if (range === 'week') {
    const weekAgo = new Date(now)
    weekAgo.setDate(now.getDate() - 7)
    return d >= weekAgo
  }
  if (range === 'month') {
    const monthAgo = new Date(now)
    monthAgo.setMonth(now.getMonth() - 1)
    return d >= monthAgo
  }
  return true
}
