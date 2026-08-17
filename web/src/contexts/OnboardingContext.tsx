import React, { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import {
  getOnboardingFlags,
  setWelcomeSeen as persistWelcomeSeen,
  setChecklistDismissed as persistChecklistDismissed,
  type OnboardingFlags,
} from '../lib/onboarding/onboardingFlag'
import { getHintStyle, type HintStyle } from '../lib/onboarding/hintStyle'
import { selectOnboardingView, type OnboardingView } from '../lib/onboarding/selectOnboardingView'
import { computeChecklistProgress, type ChecklistProgress } from '../lib/onboarding/checklistProgress'
import { useOnboardingCounts } from '../lib/onboarding/useOnboardingCounts'

export interface OnboardingContextValue {
  /** Which onboarding surface to render right now. */
  view: OnboardingView
  /** Current checklist progress (undefined until data counts have loaded). */
  progress?: ChecklistProgress
  /** "Show me how" — dismiss the welcome card and let the guided checklist run. */
  startChecklist: () => void
  /** "Skip" — dismiss the welcome card and the checklist entirely. */
  skipOnboarding: () => void
  /** Dismiss the guided checklist (e.g. its close button). */
  dismissChecklist: () => void
  /** The user's chosen hint reveal style (live; updated from Settings). */
  hintStyle: HintStyle
  /** Whether the on-demand hints reveal is currently active on this page. */
  hintsActive: boolean
  /** Reveal hints on the current page (from the "?" panel's "Show hints"). */
  showHints: () => void
  /** Hide the hints reveal. */
  hideHints: () => void
}

const OnboardingContext = createContext<OnboardingContextValue | undefined>(undefined)

/**
 * Holds first-run onboarding state for the authenticated app. Flags are seeded
 * from localStorage; when the checklist is in play, live data counts are
 * fetched once and fed through the pure progress core. The view is gated so the
 * checklist only appears after counts load (no all-incomplete flicker).
 */
export function OnboardingProvider({ children }: { children: React.ReactNode }) {
  const { pathname } = useLocation()
  const [flags, setFlags] = useState<OnboardingFlags>(() => getOnboardingFlags())
  const [hintStyle, setHintStyle] = useState<HintStyle>(() => getHintStyle())
  const [hintsActive, setHintsActive] = useState(false)

  // Keep the style live: Settings dispatches `neuroboost-hints-style-change`
  // (same tab), and `storage` covers other tabs. Mirrors the header-variant wiring.
  useEffect(() => {
    const onStyle = (e: Event) => {
      const detail = (e as CustomEvent<HintStyle>).detail
      setHintStyle(detail ?? getHintStyle())
    }
    const onStorage = (e: StorageEvent) => {
      if (e.key === 'neuroboost-hints-style') setHintStyle(getHintStyle())
    }
    window.addEventListener('neuroboost-hints-style-change', onStyle as EventListener)
    window.addEventListener('storage', onStorage)
    return () => {
      window.removeEventListener('neuroboost-hints-style-change', onStyle as EventListener)
      window.removeEventListener('storage', onStorage)
    }
  }, [])

  // Hints are per-page: leaving a route closes any active reveal.
  useEffect(() => {
    setHintsActive(false)
  }, [pathname])

  const showHints = useCallback(() => setHintsActive(true), [])
  const hideHints = useCallback(() => setHintsActive(false), [])

  const checklistActive = flags.welcomeSeen && !flags.checklistDismissed
  const { counts, loaded } = useOnboardingCounts(checklistActive)
  const progress = loaded && counts ? computeChecklistProgress(counts) : undefined

  // selectOnboardingView would return 'checklist' while progress is still
  // undefined; suppress that until counts have loaded to avoid a flicker.
  const baseView = selectOnboardingView(flags, progress)
  const view: OnboardingView = checklistActive && !loaded ? 'none' : baseView

  const startChecklist = useCallback(() => {
    persistWelcomeSeen()
    setFlags((f) => ({ ...f, welcomeSeen: true }))
  }, [])

  const skipOnboarding = useCallback(() => {
    persistWelcomeSeen()
    persistChecklistDismissed()
    setFlags((f) => ({ ...f, welcomeSeen: true, checklistDismissed: true }))
  }, [])

  const dismissChecklist = useCallback(() => {
    persistChecklistDismissed()
    setFlags((f) => ({ ...f, checklistDismissed: true }))
  }, [])

  // Once every step is done, dismiss for good. This both gives the checklist
  // natural closure and stops the per-navigation count refetch from running
  // forever for a user who finished but never clicked the close button.
  useEffect(() => {
    if (progress?.currentStep === 'done' && !flags.checklistDismissed) {
      dismissChecklist()
    }
  }, [progress?.currentStep, flags.checklistDismissed, dismissChecklist])

  const value: OnboardingContextValue = {
    view,
    progress,
    startChecklist,
    skipOnboarding,
    dismissChecklist,
    hintStyle,
    hintsActive,
    showHints,
    hideHints,
  }
  return <OnboardingContext.Provider value={value}>{children}</OnboardingContext.Provider>
}

export function useOnboarding(): OnboardingContextValue {
  const ctx = useContext(OnboardingContext)
  if (!ctx) {
    throw new Error('useOnboarding must be used within an OnboardingProvider')
  }
  return ctx
}
