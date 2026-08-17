import type { OnboardingFlags } from './onboardingFlag'
import type { ChecklistProgress } from './checklistProgress'

export type OnboardingView = 'welcome' | 'checklist' | 'none'

/**
 * Decide which onboarding surface (if any) to show.
 * Order: the welcome card has priority until seen; then the guided checklist
 * runs until it is explicitly dismissed or every step is done. `progress` is
 * optional so the welcome branch works before any counts have loaded.
 */
export function selectOnboardingView(
  flags: OnboardingFlags,
  progress?: ChecklistProgress
): OnboardingView {
  if (!flags.welcomeSeen) return 'welcome'
  if (flags.checklistDismissed) return 'none'
  if (progress && progress.currentStep === 'done') return 'none'
  return 'checklist'
}
