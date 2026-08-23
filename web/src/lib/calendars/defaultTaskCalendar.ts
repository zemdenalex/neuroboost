import type { Calendar } from '../../api/calendars'

/**
 * Which calendar a task created from the calendar page goes into.
 *
 * 🔴 The defect this answers (Denis, 23.08). Quick-add sent
 * `{ title, priority }` and nothing else, so the API applied its own default —
 * the author's personal calendar — and a task made while looking at a shared
 * week was invisible to the other person forever. No error, no hint: the task
 * existed, in a place nobody was looking at.
 *
 * 🔴 It does NOT pick "the first one". A task that quietly lands in the wrong
 * calendar is found a week later, so the rule is: prefer what the user last
 * chose, otherwise their own personal calendar — the same place the API would
 * have put it — and show the choice on screen either way. Guessing a shared
 * calendar because it happened to sort first is exactly the silent
 * misplacement this is fixing.
 */

/** Calendars a task can actually be put in. A viewer would get a 403. */
export function writableCalendars(calendars: Calendar[]): Calendar[] {
  return calendars.filter(c => c.status === 'active' && c.role !== 'viewer')
}

export function defaultTaskCalendarId(calendars: Calendar[], remembered: string | null): string {
  const writable = writableCalendars(calendars)
  if (writable.length === 0) return ''

  if (remembered && writable.some(c => c.id === remembered)) return remembered

  const personal = writable.find(c => c.kind === 'personal')
  if (personal) return personal.id

  // No personal calendar in the list at all. Rare — it means the only writable
  // calendars are shared ones — and there is no better answer than the first,
  // but the picker is on screen showing which one it is.
  return writable[0].id
}
