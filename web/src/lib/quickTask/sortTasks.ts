/** Statuses that mean the task is off the list, not waiting on you. */
const CLOSED = new Set(['DONE', 'CANCELLED'])

/**
 * The minimum a task needs for this comparator — in EITHER casing.
 *
 * This repo has two parallel task representations: raw snake_case from
 * `api/tasks` (used by the Tasks page) and camelCase from `types` (used by
 * TaskSidebar). Rather than pick a side and force conversions at the call
 * site, the comparator reads whichever field is present.
 */
export interface SortableTask {
  id: string
  status?: string
  dueDate?: string | null
  due_date?: string | null
  createdAt?: string | null
  created_at?: string | null
}

function timeOrNull(value: string | null | undefined): number | null {
  if (!value) return null
  const ms = Date.parse(value)
  return Number.isNaN(ms) ? null : ms
}

function dueOf(t: SortableTask): number | null {
  return timeOrNull(t.dueDate ?? t.due_date)
}

function madeOf(t: SortableTask): number | null {
  return timeOrNull(t.createdAt ?? t.created_at)
}

/**
 * Deterministic order for the tasks inside one priority group.
 *
 * Previously the order was whatever the API returned, which at 50 tasks a day
 * meant a list that reshuffled under the reader. The rules, in order:
 *
 *   1. open before closed — a completed task is history, not work
 *   2. dated before undated, soonest first — a deadline is the only signal
 *      strong enough to outrank recency
 *   3. newest first — so a task you just typed appears where you are looking
 *   4. id — never leave two rows free to swap places between renders
 */
export function sortWithinPriority<T extends SortableTask>(tasks: T[]): T[] {
  return [...tasks].sort((a, b) => {
    const aClosed = CLOSED.has(a.status ?? '') ? 1 : 0
    const bClosed = CLOSED.has(b.status ?? '') ? 1 : 0
    if (aClosed !== bClosed) return aClosed - bClosed

    const aDue = dueOf(a)
    const bDue = dueOf(b)
    if (aDue !== null && bDue !== null && aDue !== bDue) return aDue - bDue
    if (aDue !== null && bDue === null) return -1
    if (aDue === null && bDue !== null) return 1

    const aMade = madeOf(a)
    const bMade = madeOf(b)
    if (aMade !== null && bMade !== null && aMade !== bMade) return bMade - aMade
    if (aMade !== null && bMade === null) return -1
    if (aMade === null && bMade !== null) return 1

    return a.id.localeCompare(b.id)
  })
}
