import type { ParsedLine } from '../../api/parse'
import type { CreateTaskRequest } from '../../api/tasks'
import type { CreateEventRequest } from '../../api/events'

/**
 * The task the row saves for a parsed line: what the line said beats the
 * configured default; a field the line did not state keeps the default.
 * Tags add to the inherited filter tags rather than replacing them.
 */
export function taskFromParsed(built: CreateTaskRequest, parsed: ParsedLine): CreateTaskRequest {
  const request: CreateTaskRequest = { ...built, title: parsed.title }
  if (parsed.priority !== null) request.priority = parsed.priority
  if (parsed.estimated_minutes !== null) request.estimated_minutes = parsed.estimated_minutes
  if (parsed.due_date !== null) request.due_date = parsed.due_date
  if (parsed.rrule !== null) request.rrule = parsed.rrule
  if (parsed.tags.length > 0) request.tags = [...new Set([...(built.tags ?? []), ...parsed.tags])]
  return request
}

/**
 * The event for a confirmed line, as the bot's createOne builds it. Only what
 * was stated is sent: an absent reminder field asks the server for the
 * user's preset, while an explicit [] means silent.
 */
export function eventFromParsed(parsed: ParsedLine, taskId?: string): CreateEventRequest {
  const request: CreateEventRequest = {
    title: parsed.title,
    starts_at: parsed.starts_at ?? '',
    ends_at: parsed.ends_at ?? '',
    all_day: parsed.all_day,
    tags: parsed.tags,
  }
  if (parsed.rrule !== null) request.rrule = parsed.rrule
  if (parsed.color !== null) request.color = parsed.color
  if (parsed.calendar_id !== null) request.calendar_id = parsed.calendar_id
  if (parsed.reminder_offsets !== null) request.reminder_offsets = parsed.reminder_offsets
  if (taskId) request.task_id = taskId
  return request
}

/**
 * «вс, 27 сент., 15:00–16:00» in the zone the line was read in. An all-day
 * event prints its day (or first and last day) with no clock.
 */
export function describeParsedWhen(parsed: ParsedLine, locale: string): string {
  if (!parsed.starts_at || !parsed.ends_at) return ''
  const timeZone = parsed.timezone
  const start = new Date(parsed.starts_at)
  const end = new Date(parsed.ends_at)
  const day = (d: Date) =>
    new Intl.DateTimeFormat(locale, { timeZone, weekday: 'short', day: 'numeric', month: 'short' }).format(d)
  const clock = (d: Date) =>
    new Intl.DateTimeFormat(locale, { timeZone, hour: '2-digit', minute: '2-digit', hour12: false }).format(d)

  if (parsed.all_day) {
    // The end is the midnight AFTER the last day.
    const last = new Date(end.getTime() - 1)
    const first = day(start)
    return day(last) === first ? first : `${first} – ${day(last)}`
  }
  const sameDay = day(start) === day(end)
  return sameDay
    ? `${day(start)}, ${clock(start)}–${clock(end)}`
    : `${day(start)}, ${clock(start)} – ${day(end)}, ${clock(end)}`
}
