import { describe, it, expect } from 'vitest'
import { isHorizontalSwipe, SWIPE_MIN_PX } from './swipe'

describe('isHorizontalSwipe', () => {
  it('fires on a deliberate sideways drag', () => {
    expect(isHorizontalSwipe(120, 10)).toBe(true)
    expect(isHorizontalSwipe(-120, -10)).toBe(true)
  })

  it('does NOT fire on a scroll that drifted sideways', () => {
    // The reported defect, in numbers: a thumb scrolling a week down travels
    // far vertically and wanders sideways past the 50px threshold on the way.
    // Under the old rule (|dx| > 50 and nothing else) this changed the day.
    expect(isHorizontalSwipe(60, 300)).toBe(false)
    expect(isHorizontalSwipe(-80, 250)).toBe(false)
  })

  it('does not fire on a tap or a twitch', () => {
    expect(isHorizontalSwipe(0, 0)).toBe(false)
    expect(isHorizontalSwipe(SWIPE_MIN_PX, 0)).toBe(false) // strictly greater
  })

  it('refuses a 45-degree drag rather than guessing', () => {
    expect(isHorizontalSwipe(100, 100)).toBe(false)
  })

  it('needs both distance and dominance, not either', () => {
    // Dominantly horizontal but too short.
    expect(isHorizontalSwipe(20, 1)).toBe(false)
    // Long enough but not dominant.
    expect(isHorizontalSwipe(100, 80)).toBe(false)
  })
})
