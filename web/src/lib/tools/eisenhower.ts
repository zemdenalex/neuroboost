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
