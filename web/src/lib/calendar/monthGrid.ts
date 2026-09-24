/**
 * The month view's 6×7 grid (spec V003-20260924-arc-web-month-view, R1).
 *
 * Always 42 days from a Monday, like the bot: the grid keeps its height from
 * month to month. Days are YYYY-MM-DD computed in UTC arithmetic, so the
 * process zone and DST changes cannot skip or repeat a day.
 */

const DAY_MS = 24 * 60 * 60 * 1000

/** The 42 calendar days shown for a month; `month` is 1–12. */
export function monthGrid(year: number, month: number): string[] {
  const first = Date.UTC(year, month - 1, 1)
  // getUTCDay: 0 = Sunday. Days back to that week's Monday.
  const back = (new Date(first).getUTCDay() + 6) % 7
  const start = first - back * DAY_MS
  const days: string[] = []
  for (let i = 0; i < 42; i++) {
    days.push(new Date(start + i * DAY_MS).toISOString().slice(0, 10))
  }
  return days
}

/** The month `by` months away; `month` is 1–12. */
export function shiftMonth(year: number, month: number, by: number): { year: number; month: number } {
  const index = year * 12 + (month - 1) + by
  return { year: Math.floor(index / 12), month: (((index % 12) + 12) % 12) + 1 }
}

/** The Monday (YYYY-MM-DD) of a day's ISO week. */
function mondayOf(day: string): number {
  const ms = Date.parse(day + 'T00:00:00Z')
  return ms - ((new Date(ms).getUTCDay() + 6) % 7) * DAY_MS
}

/** How many weeks from today's week to the day's week: the week view's offset. */
export function weekOffset(today: string, day: string): number {
  return Math.round((mondayOf(day) - mondayOf(today)) / (7 * DAY_MS))
}
