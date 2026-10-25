/**
 * The Eisenhower matrix rule: which quadrant a task belongs to, and what
 * priority a drop into a quadrant assigns.
 *
 * Extracted from pages/Tools/Eisenhower.tsx on 2026-08-13. It lived inside the
 * page as an unexported function, so nothing could reach it and the tool had no
 * coverage of any kind. This is the same shape as lib/calendars/order.ts and
 * lib/quickTask/* — the rule in a leaf module, the page rendering it.
 *
 * 🔴 The priority scale is inverted: 1 is the MOST urgent (Emergency) and 5 the
 * least (If Possible), with 0 meaning Buffer. Reading it as "bigger is more
 * important" puts emergencies in the eliminate quadrant, which is why the
 * mapping is tested rather than assumed.
 */
export type QuadrantId = 'q1' | 'q2' | 'q3' | 'q4'

/**
 * Priority → quadrant.
 *
 * 1 Emergency, 2 ASAP  → q1 Do First
 * 3 Normal             → q2 Schedule
 * 4 Low                → q3 Delegate
 * 5 If Possible, 0 Buffer, anything unrecognised → q4 Eliminate
 */
export function priorityToQuadrant(priority: number): QuadrantId {
  if (priority === 1 || priority === 2) return 'q1'
  if (priority === 3) return 'q2'
  if (priority === 4) return 'q3'
  return 'q4'
}

/**
 * Quadrant → the priority a task takes when dropped there.
 *
 * One representative value per quadrant. Note this is deliberately NOT the
 * inverse of priorityToQuadrant: q1 holds both 1 and 2, and dropping into it
 * picks 1, so a task at priority 2 dragged out and back becomes priority 1.
 */
export const QUADRANT_TO_PRIORITY: Record<QuadrantId, number> = {
  q1: 1,
  q2: 3,
  q3: 4,
  q4: 5,
}

/** Days from `today` to the task's due day (negative when overdue); both 'YYYY-MM-DD'. */
function daysUntil(due: string, today: string): number {
  return Math.round((Date.parse(`${due.slice(0, 10)}T12:00:00Z`) - Date.parse(`${today}T12:00:00Z`)) / 86_400_000)
}

/**
 * The quadrant by both axes (Denis 25.09, docs/tasks-web-cleanup.md 4.9):
 * urgent = due within two days or overdue; important = priority 1 or 2.
 * `today` is the person's own day, 'YYYY-MM-DD'.
 */
export function taskQuadrant(task: { priority: number; due_date?: string | null }, today: string): QuadrantId {
  const important = task.priority === 1 || task.priority === 2
  const urgent = !!task.due_date && daysUntil(task.due_date, today) <= 2
  if (important) return urgent ? 'q1' : 'q2'
  return urgent ? 'q3' : 'q4'
}

/**
 * The priority after a drop: only importance moves (urgency belongs to the
 * date). Into Do First / Schedule an unimportant task becomes 2; out of them
 * an important one becomes 3; a task already on the right side keeps its own.
 */
export function dropPriority(current: number, target: QuadrantId): number {
  const important = current === 1 || current === 2
  const wantImportant = target === 'q1' || target === 'q2'
  if (wantImportant === important) return current
  return wantImportant ? 2 : 3
}
