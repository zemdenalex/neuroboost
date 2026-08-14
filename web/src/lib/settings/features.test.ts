import { describe, it, expect } from 'vitest'
import { mergeFeatures, DEFAULT_FEATURES, type FeatureKey } from './features'

describe('mergeFeatures', () => {
  it('returns the defaults when nothing is stored', () => {
    expect(mergeFeatures(undefined)).toEqual(DEFAULT_FEATURES)
    expect(mergeFeatures(null)).toEqual(DEFAULT_FEATURES)
  })

  it('lets a stored value override its default', () => {
    const merged = mergeFeatures({ tools: false, dreams: true })
    expect(merged.tools).toBe(false)
    expect(merged.dreams).toBe(true)
  })

  // 🔴 The rule the section depends on. A blob written before a toggle existed
  // has no key for it; replacing rather than merging would leave that key
  // undefined, rendering a switch in neither state and saving undefined back.
  it('fills in toggles the stored object has never heard of', () => {
    const merged = mergeFeatures({ tools: true })

    for (const key of Object.keys(DEFAULT_FEATURES) as FeatureKey[]) {
      expect(typeof merged[key], `${key} must be a boolean, not undefined`).toBe('boolean')
    }
    expect(merged.timeline).toBe(DEFAULT_FEATURES.timeline)
  })

  it('ignores keys it cannot render', () => {
    const merged = mergeFeatures({ tools: true, teleportation: true })
    expect(merged).not.toHaveProperty('teleportation')
  })

  // A hand-edited blob can hold a string. Accepting it would render as enabled
  // and then save a string back, which is how a boolean column rots.
  it('accepts only real booleans', () => {
    const merged = mergeFeatures({ tools: 'false', dreams: 1, goals: null })
    expect(merged.tools).toBe(DEFAULT_FEATURES.tools)
    expect(merged.dreams).toBe(DEFAULT_FEATURES.dreams)
    expect(merged.goals).toBe(DEFAULT_FEATURES.goals)
  })

  it('is not confused by an array', () => {
    expect(mergeFeatures(['tools'])).toEqual(DEFAULT_FEATURES)
  })

  // The negative control: a function that always returned the defaults would
  // pass every test above except this one.
  it('actually reads the stored value', () => {
    expect(mergeFeatures({ tools: false })).not.toEqual(DEFAULT_FEATURES)
  })

  it('does not mutate the defaults between calls', () => {
    mergeFeatures({ tools: false })
    expect(DEFAULT_FEATURES.tools, 'the shared default object was written to').toBe(true)
  })
})
