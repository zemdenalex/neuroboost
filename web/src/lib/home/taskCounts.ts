import type { Task } from '../../types'
import { answeredToday } from '../../types'

/**
 * The three numbers on the Home dashboard: to do, done, overdue.
 *
 * 🔴 A running series (rrule set, status not DONE) is read by its day, not its
 * status: its due_date is the day the series started, so as a one-off it was
 * «overdue» for ever. It is to do while today has no answer, never overdue,
 * and a switched-off series (status DONE) is not a finished task. The bot's
 * statistics count the same way (statsview.go taskTotals).
 */
export function taskCounts(tasks: Task[], now: Date = new Date()): { todo: number; done: number; overdue: number } {
  let todo = 0
  let done = 0
  let overdue = 0
  for (const t of tasks) {
    if (t.rrule) {
      if (t.status !== 'DONE' && t.status !== 'CANCELLED' && !answeredToday(t)) todo++
      continue
    }
    if (t.status === 'DONE') done++
    else if (t.status !== 'CANCELLED') {
      todo++
      if (t.dueDate && new Date(t.dueDate) < now) overdue++
    }
  }
  return { todo, done, overdue }
}
