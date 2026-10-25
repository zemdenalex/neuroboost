/**
 * Moving an event by calendar days (spec V003-20260924-arc-web-month-view, R8).
 *
 * A day is not 24 hours: across a DST change, adding 48 h to a 10:00 event
 * puts it at 09:00 or 11:00. The event keeps its wall-clock time in the
 * user's zone instead, and each end is moved on its own, so an event that
 * crosses the change keeps its local length.
 */

const DAY_MS = 24 * 60 * 60 * 1000

/** Calendar days from one YYYY-MM-DD to another. */
export function daysBetween(from: string, to: string): number {
  return Math.round((Date.parse(to + 'T00:00:00Z') - Date.parse(from + 'T00:00:00Z')) / DAY_MS)
}

/** The zone's wall clock at an instant, as if it were UTC. */
function wallClock(ms: number, timeZone: string): number {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(new Date(ms))
  const get = (type: Intl.DateTimeFormatPartTypes) => Number(parts.find((p) => p.type === type)?.value ?? '0')
  // en-CA writes midnight as 24 with hour12: false in some engines.
  const hour = get('hour') % 24
  return Date.UTC(get('year'), get('month') - 1, get('day'), hour, get('minute'), get('second'))
}

/** The instant a wall-clock time (given as if UTC) happens in the zone. */
function fromWallClock(wall: number, timeZone: string): number {
  // The offset at the guess, then once more at the corrected instant: two
  // passes settle every time except the hour a DST change skips.
  let guess = wall - (wallClock(wall, timeZone) - wall)
  guess = wall - (wallClock(guess, timeZone) - guess)
  return guess
}

function shiftOne(iso: string, days: number, timeZone: string): string {
  const ms = Date.parse(iso)
  const wall = wallClock(ms, timeZone) + days * DAY_MS
  // Keep the milliseconds formatToParts drops.
  return new Date(fromWallClock(wall, timeZone) + (ms % 1000)).toISOString()
}

/** The event moved `days` calendar days in `timeZone`, wall-clock times kept. */
export function shiftByDays(
  startsAt: string,
  endsAt: string,
  days: number,
  timeZone: string,
): { startsAt: string; endsAt: string } {
  return { startsAt: shiftOne(startsAt, days, timeZone), endsAt: shiftOne(endsAt, days, timeZone) }
}

/** A wall-clock time ("HH:MM") on a day in the zone, as an ISO instant; a bad time reads as 09:00. */
export function localTimeOn(day: string, time: string, timeZone: string): string {
  const m = /^(\d{1,2}):(\d{2})$/.exec(time)
  const [h, min] = m && Number(m[1]) < 24 && Number(m[2]) < 60 ? [Number(m[1]), Number(m[2])] : [9, 0]
  const wall = Date.parse(day + 'T00:00:00Z') + (h * 60 + min) * 60 * 1000
  return new Date(fromWallClock(wall, timeZone)).toISOString()
}
