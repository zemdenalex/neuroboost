/**
 * Which days count as working days.
 *
 * Extracted from the Settings work-hours section on 2026-08-14 so the rule can
 * be tested — the page itself has no regression suite of any kind.
 */

/** ISO order. Monday first, matching every other week in this product. */
export const WEEK_DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const

export type WeekDay = (typeof WEEK_DAYS)[number]

export const DEFAULT_WORK_DAYS: WeekDay[] = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri']

/**
 * Add or remove a day, keeping the result in ISO order.
 *
 * 🔴 The version this replaces appended with `[...days, day]`, so switching
 * Monday off and on again moved it to the END of the stored array:
 * `["Tue","Wed","Thu","Fri","Mon"]`. Nothing reads the order today — the value
 * is written to localStorage and to the API and consumed by neither — which is
 * exactly why it would have been someone else's confusing bug later, in code
 * that reasonably assumed a week runs Monday to Sunday.
 *
 * Sorting by index rather than alphabetically: "Fri" sorts before "Mon" in a
 * string comparison, which would be a different wrong answer.
 */
export function toggleWorkDay(days: readonly string[], day: string): WeekDay[] {
  const next = days.includes(day) ? days.filter((d) => d !== day) : [...days, day]
  return WEEK_DAYS.filter((d) => next.includes(d))
}

/**
 * Normalise a stored value into ISO order, dropping anything unrecognised.
 *
 * The settings blob is user-writable, so a stored array can hold junk, wrong
 * casing, or duplicates. Rendering that directly would light up buttons that do
 * not exist or light none at all.
 */
export function normaliseWorkDays(stored: unknown): WeekDay[] {
  if (!Array.isArray(stored)) return DEFAULT_WORK_DAYS
  const set = new Set(stored.filter((d): d is string => typeof d === 'string'))
  return WEEK_DAYS.filter((d) => set.has(d))
}
