const WELCOME_SEEN_KEY = 'neuroboost-onboarding-welcome-seen'
const CHECKLIST_DISMISSED_KEY = 'neuroboost-onboarding-checklist-dismissed'

export interface OnboardingFlags {
  welcomeSeen: boolean
  checklistDismissed: boolean
}

function readBool(storage: Storage, key: string): boolean {
  try {
    return storage.getItem(key) === 'true'
  } catch {
    return false
  }
}

function writeTrue(storage: Storage, key: string): void {
  try {
    storage.setItem(key, 'true')
  } catch {
    // flags are non-critical; ignore quota/availability errors
  }
}

export function getOnboardingFlags(storage: Storage = localStorage): OnboardingFlags {
  return {
    welcomeSeen: readBool(storage, WELCOME_SEEN_KEY),
    checklistDismissed: readBool(storage, CHECKLIST_DISMISSED_KEY),
  }
}

export function setWelcomeSeen(storage: Storage = localStorage): void {
  writeTrue(storage, WELCOME_SEEN_KEY)
}

export function setChecklistDismissed(storage: Storage = localStorage): void {
  writeTrue(storage, CHECKLIST_DISMISSED_KEY)
}
