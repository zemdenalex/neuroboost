import { useOnboarding } from '../../contexts/OnboardingContext'
import { HintsBubbles } from './HintsBubbles'

/**
 * Dispatcher for the on-demand hints reveal. Renders the component matching the
 * user's chosen style. Mounted once in Layout. The `walkthrough` and `markers`
 * styles ship in later iterations (5b / 5c); until then they render nothing.
 */
export function HintsLayer() {
  const { hintStyle, hintsActive } = useOnboarding()

  if (hintStyle === 'bubbles' && hintsActive) return <HintsBubbles />
  return null
}

export default HintsLayer
