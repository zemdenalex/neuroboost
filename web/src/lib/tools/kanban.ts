import type { TaskStatus } from '../../api/tasks'

/**
 * Kanban column rules: which column a task status renders in, and which status
 * a drop into a column writes back.
 *
 * Extracted from pages/Tools/Kanban.tsx on 2026-08-13, where both maps were
 * unexported and therefore unreachable by any test. The board had no coverage
 * of any kind.
 *
 * The two directions are NOT inverses, and both asymmetries are load-bearing:
 * INBOX is a view-only column that writes TODO, and CANCELLED renders under
 * Done because a cancelled task is finished with, but dropping into Done writes
 * DONE rather than preserving CANCELLED.
 */
export type KanbanColumnId = 'INBOX' | 'TODO' | 'IN_PROGRESS' | 'SCHEDULED' | 'DONE'

/**
 * Column → the status written to the API on drop.
 *
 * INBOX maps to TODO: the backend has no INBOX status, and a task dropped there
 * must still be a real, listable task.
 */
export const COLUMN_TO_STATUS: Record<KanbanColumnId, TaskStatus> = {
  INBOX: 'TODO',
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
