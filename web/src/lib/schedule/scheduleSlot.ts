/**
 * Scheduling a task with a choice: when, then how long. A straight port of
 * the bot (bot/internal/handlers/schedule.go scheduleStart, keyboards.go
 * TaskScheduleDuration, linked.go whenShort / linkedEvents), gap list row 4 in
 * docs/team/research/V003-20260926-res-bot-vs-web-gaps.md.
 *
 * Every time here is resolved in the person's zone (user.timezone), not the
 * browser's: the Mini App on a phone abroad must still mean 19:00 at home,
 * the same as the bot does (learning-the-right-time-in-the-wrong-zone).
 */

export const SCHEDULE_SLOTS = ['now', 'hour', 'eve', 'tmr'] as const
export type ScheduleSlotKey = (typeof SCHEDULE_SLOTS)[number]

/** The bot's four lengths, in minutes (keyboards.go TaskScheduleDuration). */
export const SCHEDULE_MINUTES = [15, 30, 60, 120] as const

const FALLBACK_ZONE = 'Europe/Moscow'
const MINUTE = 60_000

interface WallParts {
  year: number
  month: number // 1-12
  day: number
  hour: number
  minute: number
  weekday: number // 0 = Monday
}

function formatter(timeZone: string): Intl.DateTimeFormat {
  const options: Intl.DateTimeFormatOptions = {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    weekday: 'short',
    hourCycle: 'h23',
  }
  try {
    return new Intl.DateTimeFormat('en-US', options)
  } catch {
    return new Intl.DateTimeFormat('en-US', { ...options, timeZone: FALLBACK_ZONE })
  }
}

const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function wall(at: Date, timeZone: string): WallParts & { second: number } {
  const parts = formatter(timeZone).formatToParts(at)
  const get = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value ?? ''
  return {
    year: Number(get('year')),
    month: Number(get('month')),
    day: Number(get('day')),
    hour: Number(get('hour')),
    minute: Number(get('minute')),
    second: Number(get('second')),
    weekday: WEEKDAYS.indexOf(get('weekday')),
  }
}

/** How far the zone runs ahead of UTC at `at`, in ms. */
function offsetAt(at: Date, timeZone: string): number {
  const w = wall(at, timeZone)
  const asUtc = Date.UTC(w.year, w.month - 1, w.day, w.hour, w.minute, w.second)
  return asUtc - Math.floor(at.getTime() / 1000) * 1000
}

/**
 * The instant a wall-clock time names in the zone. The offset is read at the
 * target, not at `now`, so 09:00 on the far side of a DST change stays 09:00.
 * Day overflow (day + 1 at the end of a month) is normalised by Date.UTC.
 */
function instantOf(year: number, month: number, day: number, hour: number, minute: number, timeZone: string): Date {
  const guess = Date.UTC(year, month - 1, day, hour, minute)
  let at = guess - offsetAt(new Date(guess), timeZone)
  const corrected = guess - offsetAt(new Date(at), timeZone)
  if (corrected !== at) at = corrected
  return new Date(at)
}

/**
 * A «YYYY-MM-DDTHH:mm» typed in a datetime-local field, read as a wall time in
 * the person's zone (not the browser's). Null for anything else.
 */
export function instantFromLocalValue(value: string, timeZone: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value)
  if (!m) return null
  const [, y, mo, d, h, mi] = m.map(Number)
  return instantOf(y, mo, d, h, mi, timeZone)
}

/** The datetime-local value of an instant, as a wall time in the zone. */
export function localValueOf(at: Date, timeZone: string): string {
  const w = wall(at, timeZone)
  const pad = (x: number) => String(x).padStart(2, '0')
  return `${w.year}-${pad(w.month)}-${pad(w.day)}T${pad(w.hour)}:${pad(w.minute)}`
}

/**
 * When a slot starts, pressed at `now` (schedule.go scheduleStart).
 * «This evening» after 19:00 means the next one, as in the bot's event flow.
 */
export function scheduleStart(slot: ScheduleSlotKey, now: Date, timeZone: string): Date {
  const toMinute = (t: number) => new Date(Math.floor(t / MINUTE) * MINUTE)
  const w = wall(now, timeZone)
  switch (slot) {
    case 'now':
      return toMinute(now.getTime())
    case 'hour':
      return toMinute(now.getTime() + 60 * MINUTE)
    case 'eve': {
      const start = instantOf(w.year, w.month, w.day, 19, 0, timeZone)
      return start.getTime() > now.getTime() ? start : instantOf(w.year, w.month, w.day + 1, 19, 0, timeZone)
    }
    case 'tmr':
      return instantOf(w.year, w.month, w.day + 1, 9, 0, timeZone)
  }
}

/**
 * The lengths offered, and which one the task's own estimate points at. An
 * estimate the four do not have is offered too, so choosing it is one tap.
 */
export function scheduleDurations(estimatedMinutes: number | undefined): { options: number[]; preferred: number | undefined } {
  const usable = estimatedMinutes && estimatedMinutes > 0 && estimatedMinutes <= 24 * 60 ? estimatedMinutes : undefined
  const options: number[] = [...SCHEDULE_MINUTES]
  if (usable && !options.includes(usable)) {
    options.push(usable)
    options.sort((a, b) => a - b)
  }
  return { options, preferred: usable }
}

const WEEKDAY_SHORT: Record<'ru' | 'en', string[]> = {
  ru: ['пн', 'вт', 'ср', 'чт', 'пт', 'сб', 'вс'],
  en: ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'],
}

/**
 * A moment the way a list line has room for (linked.go whenShort): «сегодня
 * 15:00», «завтра 15:00», «пт 15:00» within the week, «22.10 15:00» further.
 *
 * Days are counted by the zone's calendar, not by 24-hour blocks: the bot's
 * Hours()/24 reads Monday as «today» across a 23-hour DST Sunday.
 */
export function whenShort(at: Date, now: Date, timeZone: string, lang: string): string {
  const l: 'ru' | 'en' = lang.startsWith('ru') ? 'ru' : 'en'
  const t = wall(at, timeZone)
  const n = wall(now, timeZone)
  const days = Math.round((Date.UTC(t.year, t.month - 1, t.day) - Date.UTC(n.year, n.month - 1, n.day)) / 86_400_000)
  const pad = (x: number) => String(x).padStart(2, '0')
  const hhmm = `${pad(t.hour)}:${pad(t.minute)}`
  if (days === 0) return (l === 'ru' ? 'сегодня ' : 'today ') + hhmm
  if (days === 1) return (l === 'ru' ? 'завтра ' : 'tomorrow ') + hhmm
  if (days > 1 && days < 7) return `${WEEKDAY_SHORT[l][t.weekday]} ${hhmm}`
  return `${pad(t.day)}.${pad(t.month)} ${hhmm}`
}

/**
 * For every task, the start of the event its time went into: the nearest one
 * starting today or later (linked.go linkedEvents). One events fetch answers
 * every row; a task scheduled only in the past shows nothing.
 */
export function linkedStarts(
  events: ReadonlyArray<{ task_id?: string | null; starts_at: string }>,
  now: Date,
  timeZone: string,
): Map<string, string> {
  const n = wall(now, timeZone)
  const dayStart = instantOf(n.year, n.month, n.day, 0, 0, timeZone).getTime()
  const out = new Map<string, string>()
  const best = new Map<string, number>()
  for (const e of events) {
    if (!e.task_id) continue
    const at = Date.parse(e.starts_at)
    if (Number.isNaN(at) || at < dayStart) continue
    const prev = best.get(e.task_id)
    if (prev === undefined || at < prev) {
      best.set(e.task_id, at)
      out.set(e.task_id, e.starts_at)
    }
  }
  return out
}
