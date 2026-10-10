import type { TaskStatus } from '../../api/tasks'

/**
 * The Tasks page status filter. «To do» includes a scheduled task: it has a
 * time, not an answer, and the bot keeps it in the list with 📅 (pass 3, A15).
 */
export function matchesStatusFilter(status: TaskStatus, filter: TaskStatus | 'ALL'): boolean {
  if (filter === 'ALL' || status === filter) return true
  return filter === 'TODO' && status === 'SCHEDULED'
}
