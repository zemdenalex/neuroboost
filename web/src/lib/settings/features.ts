/**
 * Optional-view toggles.
 *
 * Extracted from the Settings feature-toggles section on 2026-08-14 so the
 * merge rule can be tested.
 */

/**
 * The toggles the page offers, and their defaults.
 *
 * 🔴 `opportunities`, `needs` and `graph` lost their backends on 2026-08-14, and
 * no view is imported for any key but `tools`. Since 27.10 those are drawn as
 * «soon» and cannot be switched (SOON_FEATURES below, Denis's choice); the keys
 * stay because removing a stored key is a product decision, not a refactor.
 */
export const DEFAULT_FEATURES = {
  dreams: false,
  goals: false,
  projects: false,
  opportunities: false,
  needs: false,
  graph: false,
  timeline: false,
  tools: true,
} as const

export type FeatureKey = keyof typeof DEFAULT_FEATURES

/**
 * Toggles with no screen behind them: no view is imported for any of these
 * (checked 27.10 — even the DreamsView/GoalsView «TODO» stubs are unused).
 * Denis 27.10 (friction audit #10): keep them visible, marked «soon», not
 * switchable. A stored value is left as it is. Remove a key from here when its
 * screen ships.
 */
export const SOON_FEATURES: ReadonlySet<FeatureKey> = new Set<FeatureKey>([
  'dreams', 'goals', 'projects', 'timeline', 'opportunities', 'needs', 'graph',
])
export type Features = Record<FeatureKey, boolean>

/**
 * Merge what the server stored over the defaults.
 *
 * 🔴 Merged rather than replaced. A settings blob written before a toggle
 * existed has no key for it, and a replaced object would leave that key
 * `undefined` — rendering a switch that is neither on nor off, and sending
 * `undefined` back on the next save.
 *
 * Unknown keys in the stored value are dropped: they cannot be rendered (there
 * is no label for them) and keeping them would grow the stored object forever.
 */
export function mergeFeatures(stored: unknown): Features {
  const result = { ...DEFAULT_FEATURES } as Features
  if (stored === null || typeof stored !== 'object' || Array.isArray(stored)) return result

  for (const key of Object.keys(DEFAULT_FEATURES) as FeatureKey[]) {
    const value = (stored as Record<string, unknown>)[key]
    // Only a real boolean overrides a default. A string "true" from a
    // hand-edited blob would otherwise render as enabled and save as a string.
    if (typeof value === 'boolean') result[key] = value
  }
  return result
}
