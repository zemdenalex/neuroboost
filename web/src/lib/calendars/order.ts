import type { Calendar } from '../../api/calendars'

/**
 * Personal calendar first, then the rest oldest first.
 *
 * The API already sorts this way; sorting again on the client keeps the list
 * stable after a local create, which appends rather than refetches.
 */
export function sortCalendars(list: Calendar[]): Calendar[] {
  return [...list].sort((a, b) => {
    if (a.kind !== b.kind) return a.kind === 'personal' ? -1 : 1
    return a.created_at.localeCompare(b.created_at)
  })
}
