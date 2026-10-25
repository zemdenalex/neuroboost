import type { Task } from '../../types'
import { tickAction, tickedToday, type TickAction } from '../../lib/tasks/tickAction'

// The calendar's task panel speaks camelCase (types/index.ts Task); tickAction
// reads the API's snake_case field. One mapping, so the panel decides a tick
// exactly as the Tasks page does: a running series answers today, never DONE.
const tickable = (task: Task) => ({ status: task.status, rrule: task.rrule, occurrence_state: task.occurrenceState })

export function sidebarTick(task: Task): TickAction {
  return tickAction(tickable(task))
}

export function sidebarTicked(task: Task): boolean {
  return tickedToday(tickable(task))
}
