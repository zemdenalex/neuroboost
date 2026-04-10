import { useAuthContext } from '../contexts/AuthContext'

export interface FeatureFlags {
  dreams: boolean
  goals: boolean
  projects: boolean
  opportunities: boolean
  needs: boolean
  graph: boolean
  timeline: boolean
  tools: boolean
}

const DEFAULTS: FeatureFlags = {
  dreams: false,
  goals: false,
  projects: false,
  opportunities: false,
  needs: false,
  graph: false,
  timeline: false,
  tools: true,
}

/**
 * Returns the current user's feature flag settings merged with safe
 * defaults. Used by nav components to show/hide optional sections.
 */
export function useFeatureFlags(): FeatureFlags {
  const { user } = useAuthContext()
  const features = user?.settings?.features
  if (!features) return DEFAULTS
  return { ...DEFAULTS, ...features }
}
