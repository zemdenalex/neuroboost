import { describe, it, expect } from 'vitest'
import { shouldRefetchOnReturn, STALE_AFTER_MS } from './refetchOnReturn'

const T = 1_000_000

describe('shouldRefetchOnReturn', () => {
  it('reloads when the tab comes back after a real absence', () => {
    expect(shouldRefetchOnReturn('visible', T, T + STALE_AFTER_MS)).toBe(true)
    expect(shouldRefetchOnReturn('visible', T, T + 5 * 60_000)).toBe(true)
  })

  it('does not reload on an alt-tab flick', () => {
    // The whole reason the threshold exists: switching windows fires
    // visibilitychange over and over, and each one would cost a week of events.
    expect(shouldRefetchOnReturn('visible', T, T + 200)).toBe(false)
    expect(shouldRefetchOnReturn('visible', T, T + STALE_AFTER_MS - 1)).toBe(false)
  })

  it('does nothing when the tab is being hidden', () => {
    // visibilitychange fires in both directions; only the return matters.
    expect(shouldRefetchOnReturn('hidden', T, T + 10 * 60_000)).toBe(false)
  })

  it('reloads rather than stalling when the clock jumped backwards', () => {
    // A resumed laptop or an NTP correction makes the age negative. Suppressing
    // the reload then would silence it exactly when the data is most stale.
    expect(shouldRefetchOnReturn('visible', T, T - 60_000)).toBe(true)
  })
})
