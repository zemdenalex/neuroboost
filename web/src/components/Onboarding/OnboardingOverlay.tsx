import { useOnboarding } from '../../contexts/OnboardingContext'
import { WelcomeCard } from './WelcomeCard'

/**
 * Renders whichever onboarding surface the context selects. Phase 2a handles
 * the welcome card; the guided checklist (view === 'checklist') is added in a
 * later phase and renders nothing here until then.
 */
export function OnboardingOverlay() {
  const { view, startChecklist, skipOnboarding } = useOnboarding()

  if (view === 'welcome') {
    return <WelcomeCard onStart={startChecklist} onSkip={skipOnboarding} />
  }

  return null
}

export default OnboardingOverlay
