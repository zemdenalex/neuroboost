/** A task created during the current quick-add session. */
export interface TrailEntry {
  id: string
  parentId?: string
}

/**
 * Which parent the next quick-added task should get.
 *
 * 'in'  — nest under the most recently created task.
 * 'out' — climb one level from the most recently created task.
 *
 * The trail is the tasks created in this quick-add session, oldest first.
 * Nothing created yet means there is nothing to nest under, so both
 * directions land at the top level.
 */
export function nextParentId(trail: TrailEntry[], direction: 'in' | 'out'): string | undefined {
  const last = trail[trail.length - 1]
  if (!last) return undefined
  if (direction === 'in') return last.id
  const parent = trail.find(entry => entry.id === last.parentId)
  return parent?.parentId
}
