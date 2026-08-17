export interface ChecklistCounts {
  taskCount: number
  eventCount: number
  completedTaskCount: number
  reflectionCount: number
}

export type ChecklistStepId = 'createTask' | 'scheduleTask' | 'completeAndReflect'

export interface ChecklistProgress {
  steps: Record<ChecklistStepId, boolean>
  currentStep: ChecklistStepId | 'done'
  completedCount: number
}

const STEP_ORDER: ChecklistStepId[] = ['createTask', 'scheduleTask', 'completeAndReflect']

export function computeChecklistProgress(counts: Partial<ChecklistCounts>): ChecklistProgress {
  const taskCount = counts.taskCount ?? 0
  const eventCount = counts.eventCount ?? 0
  const completedTaskCount = counts.completedTaskCount ?? 0
  const reflectionCount = counts.reflectionCount ?? 0

  const steps: Record<ChecklistStepId, boolean> = {
    createTask: taskCount >= 1,
    scheduleTask: eventCount >= 1,
    completeAndReflect: completedTaskCount >= 1 || reflectionCount >= 1,
  }
  const currentStep: ChecklistStepId | 'done' = STEP_ORDER.find((s) => !steps[s]) ?? 'done'
  const completedCount = STEP_ORDER.filter((s) => steps[s]).length

  return { steps, currentStep, completedCount }
}
