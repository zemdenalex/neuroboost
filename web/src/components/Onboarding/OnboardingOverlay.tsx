import { useOnboarding } from '../../contexts/OnboardingContext'
import { WelcomeCard } from './WelcomeCard'
import { GuidedChecklist } from './GuidedChecklist'

/**
 * Renders whichever onboarding surface the context selects: the first-run
 * welcome card, then the guided checklist, then nothing.
 */
export function OnboardingOverlay() {
  const { view, progress, startChecklist, skipOnboarding, dismissChecklist } = useOnboarding()

  if (view === 'welcome') {
    return <WelcomeCard onStart={startChecklist} onSkip={skipOnboarding} />
  }

  if (view === 'checklist' && progress) {
    return <GuidedChecklist progress={progress} onDismiss={dismissChecklist} />
  }

  return null
}

export default OnboardingOverlay
