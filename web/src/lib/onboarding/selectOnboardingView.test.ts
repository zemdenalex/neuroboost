import { describe, it, expect } from 'vitest'
import { selectOnboardingView } from './selectOnboardingView'
import type { ChecklistProgress } from './checklistProgress'

const inProgress: ChecklistProgress = {
  steps: { createTask: true, scheduleTask: false, completeAndReflect: false },
  currentStep: 'scheduleTask',
  completedCount: 1,
}
const allDone: ChecklistProgress = {
  steps: { createTask: true, scheduleTask: true, completeAndReflect: true },
  currentStep: 'done',
  completedCount: 3,
}

describe('selectOnboardingView', () => {
  it("shows the welcome card until the welcome has been seen", () => {
    expect(selectOnboardingView({ welcomeSeen: false, checklistDismissed: false })).toBe('welcome')
    // welcome takes priority even if other flags/progress would say otherwise
    expect(selectOnboardingView({ welcomeSeen: false, checklistDismissed: true }, allDone)).toBe('welcome')
  })

  it('shows the checklist after welcome is seen and it is not dismissed or finished', () => {
    expect(selectOnboardingView({ welcomeSeen: true, checklistDismissed: false })).toBe('checklist')
    expect(selectOnboardingView({ welcomeSeen: true, checklistDismissed: false }, inProgress)).toBe('checklist')
  })

  it('shows nothing once the checklist is dismissed', () => {
    expect(selectOnboardingView({ welcomeSeen: true, checklistDismissed: true })).toBe('none')
  })

  it('shows nothing once all checklist steps are done', () => {
    expect(selectOnboardingView({ welcomeSeen: true, checklistDismissed: false }, allDone)).toBe('none')
  })
})
