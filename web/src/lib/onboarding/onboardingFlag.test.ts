import { describe, it, expect } from 'vitest'
import { getOnboardingFlags, setWelcomeSeen, setChecklistDismissed } from './onboardingFlag'

function fakeStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map(Object.entries(initial))
  return {
    getItem: (k) => (map.has(k) ? (map.get(k) as string) : null),
    setItem: (k, v) => { map.set(k, String(v)) },
    removeItem: (k) => { map.delete(k) },
    clear: () => map.clear(),
    key: (i) => Array.from(map.keys())[i] ?? null,
    get length() { return map.size },
  } as Storage
}

describe('onboardingFlag', () => {
  it('defaults to all-false when storage is empty', () => {
    expect(getOnboardingFlags(fakeStorage())).toEqual({ welcomeSeen: false, checklistDismissed: false })
  })

  it('reads true flags written by the setters', () => {
    const s = fakeStorage()
    setWelcomeSeen(s)
    setChecklistDismissed(s)
    expect(getOnboardingFlags(s)).toEqual({ welcomeSeen: true, checklistDismissed: true })
  })

  it('treats malformed values as false', () => {
    const s = fakeStorage({
      'neuroboost-onboarding-welcome-seen': 'yes',
      'neuroboost-onboarding-checklist-dismissed': '1',
    })
    expect(getOnboardingFlags(s)).toEqual({ welcomeSeen: false, checklistDismissed: false })
  })

  it('never throws when storage access throws', () => {
    const throwing = {
      getItem: () => { throw new Error('blocked') },
      setItem: () => { throw new Error('quota') },
      removeItem: () => {}, clear: () => {}, key: () => null, length: 0,
    } as unknown as Storage
    expect(() => getOnboardingFlags(throwing)).not.toThrow()
    expect(getOnboardingFlags(throwing)).toEqual({ welcomeSeen: false, checklistDismissed: false })
    expect(() => setWelcomeSeen(throwing)).not.toThrow()
    expect(() => setChecklistDismissed(throwing)).not.toThrow()
  })
})
