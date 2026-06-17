import React, { createContext, useContext, useState, useCallback } from 'react'
import {
  getOnboardingFlags,
  setWelcomeSeen as persistWelcomeSeen,
  setChecklistDismissed as persistChecklistDismissed,
  type OnboardingFlags,
} from '../lib/onboarding/onboardingFlag'
import { selectOnboardingView, type OnboardingView } from '../lib/onboarding/selectOnboardingView'

export interface OnboardingContextValue {
  /** Which onboarding surface to render right now. */
  view: OnboardingView
  /** "Show me how" — dismiss the welcome card and let the guided checklist run. */
  startChecklist: () => void
  /** "Skip" — dismiss the welcome card and the checklist entirely. */
  skipOnboarding: () => void
}

const OnboardingContext = createContext<OnboardingContextValue | undefined>(undefined)

/**
 * Holds first-run onboarding state for the authenticated app. State is seeded
 * from localStorage (via the pure flag core) and updated optimistically so the
 * UI reacts immediately. The guided-checklist view (which needs live data
 * counts) is wired in a later phase; for now `selectOnboardingView` runs with
 * no progress, which is enough to drive the welcome card.
 */
export function OnboardingProvider({ children }: { children: React.ReactNode }) {
  const [flags, setFlags] = useState<OnboardingFlags>(() => getOnboardingFlags())

  const startChecklist = useCallback(() => {
    persistWelcomeSeen()
    setFlags((f) => ({ ...f, welcomeSeen: true }))
  }, [])

  const skipOnboarding = useCallback(() => {
    persistWelcomeSeen()
    persistChecklistDismissed()
    setFlags((f) => ({ ...f, welcomeSeen: true, checklistDismissed: true }))
  }, [])

  const view = selectOnboardingView(flags)

  const value: OnboardingContextValue = { view, startChecklist, skipOnboarding }
  return <OnboardingContext.Provider value={value}>{children}</OnboardingContext.Provider>
}

export function useOnboarding(): OnboardingContextValue {
  const ctx = useContext(OnboardingContext)
  if (!ctx) {
    throw new Error('useOnboarding must be used within an OnboardingProvider')
  }
  return ctx
}
