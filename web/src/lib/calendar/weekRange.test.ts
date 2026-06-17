import { describe, it, expect } from 'vitest'
import { computeWeekRange } from './weekRange'

const DAY = 24 * 60 * 60 * 1000

describe('computeWeekRange', () => {
  it('always starts on Monday 00:00 and contains the reference day (every weekday, incl. Sunday)', () => {
    // Jun 15–21 2026 covers Mon…Sun. The Sunday case is the off-by-one regression:
    // the buggy formula put the week start on the NEXT Monday, excluding the reference.
    for (let i = 0; i < 7; i++) {
      const ref = new Date(2026, 5, 15 + i, 14, 30) // local time
      const { start, end } = computeWeekRange(ref, 0)
      expect(start.getDay()).toBe(1) // Monday
      expect(start.getHours()).toBe(0)
      expect(start.getMinutes()).toBe(0)
      expect(end.getDay()).toBe(1) // following Monday
      expect(start.getTime()).toBeLessThanOrEqual(ref.getTime())
      expect(ref.getTime()).toBeLessThan(end.getTime())
    }
  })

  it('shifts by whole weeks with the offset', () => {
    const ref = new Date(2026, 5, 17, 9, 0) // a non-DST-transition week
    const base = computeWeekRange(ref, 0)
    const next = computeWeekRange(ref, 1)
    const prev = computeWeekRange(ref, -1)
    expect(next.start.getTime() - base.start.getTime()).toBe(7 * DAY)
    expect(base.start.getTime() - prev.start.getTime()).toBe(7 * DAY)
  })
})
