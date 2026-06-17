import { describe, it, expect } from 'vitest'
import { computeChecklistProgress } from './checklistProgress'

describe('computeChecklistProgress', () => {
  it('all steps incomplete when every count is zero', () => {
    const p = computeChecklistProgress({ taskCount: 0, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 })
    expect(p.steps).toEqual({ createTask: false, scheduleTask: false, completeAndReflect: false })
    expect(p.currentStep).toBe('createTask')
    expect(p.completedCount).toBe(0)
  })

  it('createTask done, next step is scheduleTask', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 })
    expect(p.steps.createTask).toBe(true)
    expect(p.currentStep).toBe('scheduleTask')
    expect(p.completedCount).toBe(1)
  })

  it('completeAndReflect satisfied by a reflection alone', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 1, completedTaskCount: 0, reflectionCount: 2 })
    expect(p.steps.completeAndReflect).toBe(true)
    expect(p.currentStep).toBe('done')
    expect(p.completedCount).toBe(3)
  })

  it('completeAndReflect satisfied by a completed task alone', () => {
    const p = computeChecklistProgress({ taskCount: 1, eventCount: 1, completedTaskCount: 1, reflectionCount: 0 })
    expect(p.steps.completeAndReflect).toBe(true)
    expect(p.currentStep).toBe('done')
  })

  it('treats missing/undefined counts as zero', () => {
    const p = computeChecklistProgress({})
    expect(p.currentStep).toBe('createTask')
    expect(p.completedCount).toBe(0)
  })
})
