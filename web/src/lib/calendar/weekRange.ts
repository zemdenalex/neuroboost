/**
 * The Monday-based week window [start, end) containing `reference`, shifted by
 * `offset` whole weeks. Mirrors `startOfWeek` in utils/date.ts: Sunday stays in
 * the CURRENT week (the off-by-one that emptied the calendar on Sundays came from
 * `getDate() - getDay() + 1`, which sends Sunday to the next Monday).
 */
export function computeWeekRange(reference: Date, offset: number): { start: Date; end: Date } {
  const start = new Date(reference)
  start.setDate(reference.getDate() - ((reference.getDay() + 6) % 7) + offset * 7)
  start.setHours(0, 0, 0, 0)

  const end = new Date(start)
  end.setDate(start.getDate() + 7)

  return { start, end }
}
