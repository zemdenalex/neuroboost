import type { TaskStatus } from '../../api/tasks'

/**
 * What the checkbox in a task row does (docs/tasks-web-cleanup.md 4.11).
 *
 * 🔴 For a running series `status` describes the SERIES: DONE switches it off
 * for ever. A tick answers today's day of it instead (POST
 * /tasks/{id}/occurrences), as the bot and Kanban (lib/tools/kanban.ts) do; a
 * second tick on a day already done takes the answer back ("open").
 */
export type TickAction =
  | { kind: 'status'; next: TaskStatus }
  | { kind: 'occurrence'; state: 'done' | 'open' }

interface Tickable {
  status: TaskStatus
  rrule?: string
  occurrence_state?: string
}

export function tickAction(task: Tickable): TickAction {
  if (task.rrule && task.status !== 'DONE') {
    return { kind: 'occurrence', state: task.occurrence_state === 'done' ? 'open' : 'done' }
  }
  return { kind: 'status', next: task.status === 'DONE' ? 'TODO' : 'DONE' }
}

/** Whether the row's checkbox shows ticked. */
export function tickedToday(task: Tickable): boolean {
  if (task.rrule && task.status !== 'DONE') return task.occurrence_state === 'done'
  return task.status === 'DONE'
}
