import type { TaskStatus } from '../../api/tasks'

/**
 * Kanban column rules: which column a task status renders in, and which status
 * a drop into a column writes back.
 *
 * Extracted from pages/Tools/Kanban.tsx on 2026-08-13, where both maps were
 * unexported and therefore unreachable by any test. The board had no coverage
 * of any kind.
 *
 * The two directions are NOT inverses: CANCELLED renders under Done because a
 * cancelled task is finished with, but dropping into Done writes DONE rather
 * than preserving CANCELLED.
 *
 * 25.09 (audit, Denis approved the cleanup): the INBOX column is gone. The
 * backend has no such status, so it was always empty, and a task created in it
 * jumped to TODO and seemed to vanish.
 */
export type KanbanColumnId = 'TODO' | 'IN_PROGRESS' | 'SCHEDULED' | 'DONE'

/** Column → the status written to the API on drop. */
export const COLUMN_TO_STATUS: Record<KanbanColumnId, TaskStatus> = {
  TODO: 'TODO',
  IN_PROGRESS: 'IN_PROGRESS',
  SCHEDULED: 'SCHEDULED',
  DONE: 'DONE',
}

/**
 * Status → the column a task renders in.
 *
 * CANCELLED renders under Done: it is not "done", but it is finished, and the
 * board has no column for it. Anything unrecognised falls back to TODO so a
 * task can never become invisible — a task that renders nowhere reads to the
 * user as data loss.
 */
export function statusToColumn(status: TaskStatus): KanbanColumnId {
  switch (status) {
    case 'TODO':
      return 'TODO'
    case 'IN_PROGRESS':
      return 'IN_PROGRESS'
    case 'SCHEDULED':
      return 'SCHEDULED'
    case 'DONE':
      return 'DONE'
    case 'CANCELLED':
      return 'DONE'
    default:
      return 'TODO'
  }
}

/** What a drop does. */
export type DropAction =
  | { kind: 'none' }
  | { kind: 'status'; status: TaskStatus }
  /** A repeating task dropped on Done closes today's occurrence, not the series. */
  | { kind: 'occurrence'; date: string }

/**
 * - Same column: nothing.
 * - Scheduled: nothing. The column shows tasks that have a time in the
 *   calendar; a drop there used to set the status with no event behind it.
 *   Scheduling happens in the calendar.
 * - Done with a repeating task: today's occurrence only (as the bot and the
 *   day-tasks page do); writing DONE closed the whole series.
 */
export function dropAction(task: { status: TaskStatus; rrule?: string | null }, target: KanbanColumnId, today: string): DropAction {
  if (statusToColumn(task.status) === target) return { kind: 'none' }
  if (target === 'SCHEDULED') return { kind: 'none' }
  if (target === 'DONE' && task.rrule) return { kind: 'occurrence', date: today }
  return { kind: 'status', status: COLUMN_TO_STATUS[target] }
}
