import { useOnboarding } from '../../contexts/OnboardingContext'
import { HintsBubbles } from './HintsBubbles'
import { HintsWalkthrough } from './HintsWalkthrough'

/**
 * Dispatcher for the on-demand hints reveal. Renders the component matching the
 * user's chosen style. Mounted once in Layout. The `markers` style ships in a
 * later iteration (5c); until then it renders nothing.
 */
export function HintsLayer() {
  const { hintStyle, hintsActive } = useOnboarding()

  if (!hintsActive) return null
  if (hintStyle === 'bubbles') return <HintsBubbles />
  if (hintStyle === 'walkthrough') return <HintsWalkthrough />
  return null
}

export default HintsLayer
