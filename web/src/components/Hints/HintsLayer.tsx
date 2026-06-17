import { useOnboarding } from '../../contexts/OnboardingContext'
import { HintsBubbles } from './HintsBubbles'
import { HintsWalkthrough } from './HintsWalkthrough'
import { HintsMarkers } from './HintsMarkers'

/**
 * Dispatcher for the on-demand hints reveal. Renders the component matching the
 * user's chosen style. Mounted once in Layout.
 *
 * `markers` is always-on (persistent dots, no trigger), so it is checked before
 * the `hintsActive` gate; `bubbles` and `walkthrough` are on-demand reveals.
 */
export function HintsLayer() {
  const { hintStyle, hintsActive } = useOnboarding()

  if (hintStyle === 'markers') return <HintsMarkers />
  if (!hintsActive) return null
  if (hintStyle === 'bubbles') return <HintsBubbles />
  if (hintStyle === 'walkthrough') return <HintsWalkthrough />
  return null
}

export default HintsLayer
