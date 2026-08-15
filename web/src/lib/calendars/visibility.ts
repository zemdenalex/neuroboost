/**
 * Which calendars are hidden from the grid.
 *
 * 🔴 The stored set is the HIDDEN ones, not the visible ones, and that choice
 * is load-bearing. A "visible" list would not contain a calendar created after
 * it was written — nor one someone else shares with you — so the new calendar
 * would be filtered out of the grid the moment it appeared, looking like the
 * create had failed. Storing the exceptions means the default is "show it",
 * which is the only default that stays right as the list grows.
 *
 * Nothing is pruned when a calendar is deleted. A stale id costs a few bytes
 * and filters nothing; pruning against a fetched list would erase the user's
 * choices during any load that came back short, which is a far worse failure
 * than an unused string in localStorage.
 */

const STORAGE_KEY = 'nb-hidden-calendars'

/**
 * localStorage throws in private-mode Safari and when the quota is full.
 * Reminder visibility is a preference, not data — the grid must render either
 * way, so both directions swallow the failure and fall back to "nothing
 * hidden".
 */
export function loadHiddenCalendars(): Set<string> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return new Set()
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return new Set()
    return new Set(parsed.filter((id): id is string => typeof id === 'string'))
  } catch {
    return new Set()
  }
}

export function saveHiddenCalendars(hidden: Set<string>): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify([...hidden]))
  } catch {
    /* the grid still filters correctly this session; only the memory is lost */
  }
}

/** A new set with `id` flipped, so React sees a changed identity. */
export function toggleHidden(hidden: Set<string>, id: string): Set<string> {
  const next = new Set(hidden)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  return next
}

/**
 * Filter for the grid.
 *
 * An event with no calendar is always shown. Events created before calendars
 * existed have no `calendarId`, and so does anything whose calendar the client
 * has not loaded — hiding those would make events vanish for a reason the user
 * has no control over and no way to see.
 */
export function visibleEvents<T extends { calendarId?: string }>(
  events: T[],
  hidden: Set<string>,
): T[] {
  if (hidden.size === 0) return events
  return events.filter(e => !e.calendarId || !hidden.has(e.calendarId))
}
