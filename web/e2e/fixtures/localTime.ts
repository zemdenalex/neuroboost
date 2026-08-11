/**
 * Local-day arithmetic for the drag specs.
 *
 * The grid lays out LOCAL days, not UTC days. A first version of the multi-day
 * resize test built its window on UTC boundaries, produced an event that never
 * crossed Moscow midnight, and failed for a reason unrelated to the fix it was
 * guarding. Everything here works in the account's own zone for that reason.
 */

export const DAY_MS = 24 * 3600 * 1000

/** Offset of an IANA zone at a given instant, in ms — the trick timezone.utils uses. */
export function offsetMs(timeZone: string, at: Date): number {
  const utc = new Date(at.toLocaleString('en-US', { timeZone: 'UTC' }))
  const local = new Date(at.toLocaleString('en-US', { timeZone }))
  return local.getTime() - utc.getTime()
}

/** The UTC instant of local midnight `dayOffset` days from today, in `timeZone`. */
export function localMidnightUtc(timeZone: string, dayOffset = 0): number {
  const now = Date.now()
  const off = offsetMs(timeZone, new Date(now))
  const localMidnight = Math.floor((now + off) / DAY_MS) * DAY_MS + dayOffset * DAY_MS
  return localMidnight - off
}

/** The UTC instant of the next local midnight. */
export function nextLocalMidnightUtc(timeZone: string): number {
  return localMidnightUtc(timeZone, 1)
}

/** Local weekday today, 0 = Sunday … 6 = Saturday. */
export function localWeekday(timeZone: string): number {
  const name = new Intl.DateTimeFormat('en-US', { timeZone, weekday: 'short' }).format(new Date())
  return ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(name)
}
