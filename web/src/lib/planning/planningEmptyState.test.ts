import { describe, it, expect } from 'vitest'
import { planningEmptyVariant } from './planningEmptyState'

describe('planningEmptyVariant', () => {
  it('returns null when there are unscheduled tasks to show', () => {
    expect(planningEmptyVariant(3, 0)).toBeNull()
    expect(planningEmptyVariant(1, 10)).toBeNull()
  })

  it('is "noTasks" when nothing is unscheduled and nothing is scheduled (new user)', () => {
    expect(planningEmptyVariant(0, 0)).toBe('noTasks')
  })

  it('is "allScheduled" when nothing is unscheduled but work is scheduled', () => {
    expect(planningEmptyVariant(0, 5)).toBe('allScheduled')
  })

  it('treats non-positive scheduled hours as no scheduled work', () => {
    expect(planningEmptyVariant(0, -1)).toBe('noTasks')
  })
})
