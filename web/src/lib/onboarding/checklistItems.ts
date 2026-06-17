import type { ChecklistProgress, ChecklistStepId } from './checklistProgress'

export interface ChecklistItem {
  id: ChecklistStepId
  /** i18n key (under the `onboarding` namespace) for the step label. */
  labelKey: string
  /** Route the step's CTA navigates to. */
  ctaRoute: string
  done: boolean
  current: boolean
}

const STEP_META: ReadonlyArray<{ id: ChecklistStepId; labelKey: string; ctaRoute: string }> = [
  { id: 'createTask', labelKey: 'checklist.createTask', ctaRoute: '/tasks' },
  { id: 'scheduleTask', labelKey: 'checklist.scheduleTask', ctaRoute: '/calendar' },
  { id: 'completeAndReflect', labelKey: 'checklist.completeAndReflect', ctaRoute: '/reflections' },
]

/** Turn computed progress into a render-ready list for the guided checklist UI. */
export function buildChecklistItems(progress: ChecklistProgress): ChecklistItem[] {
  return STEP_META.map((meta) => ({
    id: meta.id,
    labelKey: meta.labelKey,
    ctaRoute: meta.ctaRoute,
    done: progress.steps[meta.id],
    current: progress.currentStep === meta.id,
  }))
}
