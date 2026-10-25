import type { Calendar } from '../../api/calendars'
import { writableCalendars } from '../calendars/defaultTaskCalendar'

/**
 * Tasks a focus block can be linked to (docs/tasks-web-cleanup.md 4.8, audit T1a).
 *
 * 🔴 GET /api/tasks returns everything the user can READ, including calendars
 * shared with them as viewer; log-time writes only where they may CHANGE
 * things. Offering a read-only task meant the block failed at log-time and
 * tracking.ts rolled the whole block back: nothing saved.
 */
export function linkableTasks<T extends { calendar_id?: string; status: string }>(
  tasks: T[],
  calendars: Calendar[],
): T[] {
  const writable = new Set(writableCalendars(calendars).map((c) => c.id))
  return tasks.filter(
    (t) => t.status !== 'DONE' && t.status !== 'CANCELLED' && (!t.calendar_id || writable.has(t.calendar_id)),
  )
}
