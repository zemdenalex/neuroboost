export interface StepState {
  index: number
  isFirst: boolean
  isLast: boolean
}

/**
 * Clamp a step index into a valid position for a walkthrough of `total` steps,
 * reporting whether it sits at the first/last position. An empty list (total ≤ 0)
 * collapses to index 0 that is both first and last.
 */
export function clampStep(index: number, total: number): StepState {
  if (total <= 0) return { index: 0, isFirst: true, isLast: true }
  const clamped = Math.max(0, Math.min(index, total - 1))
  return { index: clamped, isFirst: clamped === 0, isLast: clamped === total - 1 }
}

/**
 * Move one step in `dir` (+1 forward, -1 back) and clamp — never wraps past the ends.
 */
export function nextStep(current: number, total: number, dir: 1 | -1): StepState {
  return clampStep(current + dir, total)
}
