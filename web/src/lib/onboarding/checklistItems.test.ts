import { describe, it, expect } from 'vitest'
import { buildChecklistItems } from './checklistItems'
import { computeChecklistProgress } from './checklistProgress'

describe('buildChecklistItems', () => {
  it('returns the three steps in order with stable routes', () => {
    const items = buildChecklistItems(computeChecklistProgress({}))
    expect(items.map((i) => i.id)).toEqual(['createTask', 'scheduleTask', 'completeAndReflect'])
    expect(items.map((i) => i.ctaRoute)).toEqual(['/tasks', '/calendar', '/reflections'])
    expect(items.every((i) => i.labelKey.startsWith('checklist.'))).toBe(true)
  })

  it('marks done steps and flags exactly the current step', () => {
    const progress = computeChecklistProgress({ taskCount: 1, eventCount: 0, completedTaskCount: 0, reflectionCount: 0 })
    const items = buildChecklistItems(progress)
    expect(items.find((i) => i.id === 'createTask')).toMatchObject({ done: true, current: false })
    expect(items.find((i) => i.id === 'scheduleTask')).toMatchObject({ done: false, current: true })
    expect(items.find((i) => i.id === 'completeAndReflect')).toMatchObject({ done: false, current: false })
  })

  it('flags nothing as current once every step is done', () => {
    const progress = computeChecklistProgress({ taskCount: 1, eventCount: 1, completedTaskCount: 1, reflectionCount: 0 })
    const items = buildChecklistItems(progress)
    expect(items.every((i) => i.done)).toBe(true)
    expect(items.some((i) => i.current)).toBe(false)
  })
})
