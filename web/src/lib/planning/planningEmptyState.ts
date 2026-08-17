export type PlanningEmptyVariant = 'noTasks' | 'allScheduled'

/**
 * Which empty state (if any) the unscheduled-tasks list should show.
 * `null` means there are tasks to render. Otherwise distinguish a brand-new
 * user (nothing scheduled yet) from someone who has scheduled everything, so
 * the copy is honest in both cases.
 */
export function planningEmptyVariant(
  unscheduledCount: number,
  scheduledHours: number
): PlanningEmptyVariant | null {
  if (unscheduledCount > 0) return null
  return scheduledHours > 0 ? 'allScheduled' : 'noTasks'
}
