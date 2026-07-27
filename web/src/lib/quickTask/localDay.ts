/**
 * Midnight at the start of the local day `offsetDays` away from `now`,
 * returned as an absolute instant.
 *
 * Computed from the zone's own wall-clock parts rather than from UTC: a task
 * created at 00:40 Moscow time must not be filed under the previous day.
 *
 * The zone offset is read at `now`, not at the target midnight, so a target
 * that lands on the far side of a DST transition can be off by an hour. That
 * is acceptable here — the default zone is Europe/Moscow, which has had no DST
 * since 2014, and the value is a due *date*, not an appointment time.
 */
export function startOfLocalDay(now: Date, timeZone: string, offsetDays: number): Date {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).formatToParts(now)

  const get = (type: Intl.DateTimeFormatPartTypes): number =>
    Number(parts.find(p => p.type === type)?.value ?? '0')

  const localAsUtc = Date.UTC(get('year'), get('month') - 1, get('day'), get('hour'), get('minute'), get('second'))
  // How far the zone runs ahead of UTC at this instant. formatToParts has no
  // millisecond field, so compare against a whole-second `now`.
  const zoneOffsetMs = localAsUtc - Math.floor(now.getTime() / 1000) * 1000
  const midnightLocalAsUtc = Date.UTC(get('year'), get('month') - 1, get('day') + offsetDays)
  return new Date(midnightLocalAsUtc - zoneOffsetMs)
}
