import { describe, it, expect } from 'vitest'
import { monthGrid, shiftMonth } from './monthGrid'

describe('monthGrid', () => {
  it('is always 42 days starting on a Monday', () => {
    // September 2026 starts on a Tuesday: the grid opens on Monday 31 August.
    const days = monthGrid(2026, 9)
    expect(days).toHaveLength(42)
    expect(days[0]).toBe('2026-08-31')
    expect(days[1]).toBe('2026-09-01')
    expect(days[41]).toBe('2026-10-11')
  })

  it('starts on the 1st when the month opens on a Monday', () => {
    // June 2026: 1 June is a Monday.
    expect(monthGrid(2026, 6)[0]).toBe('2026-06-01')
  })

  it('opens a Sunday-first month six days early', () => {
    // March 2026: 1 March is a Sunday.
    expect(monthGrid(2026, 3)[0]).toBe('2026-02-23')
    expect(monthGrid(2026, 3)[6]).toBe('2026-03-01')
  })

  it('walks a non-leap February without skipping or repeating a day', () => {
    const days = monthGrid(2027, 2)
    expect(days).toContain('2027-02-28')
    expect(days).not.toContain('2027-02-29')
    expect(new Set(days).size).toBe(42)
    const i = days.indexOf('2027-02-28')
    expect(days[i + 1]).toBe('2027-03-01')
  })

  it('is the same in any process zone: dates are computed, not read from a local clock', () => {
    // Europe/Berlin moves its clocks on 25 October 2026; the grid must still be consecutive.
    const days = monthGrid(2026, 10)
    for (let i = 1; i < days.length; i++) {
      const a = Date.parse(days[i - 1] + 'T00:00:00Z')
      const b = Date.parse(days[i] + 'T00:00:00Z')
      expect(b - a).toBe(24 * 60 * 60 * 1000)
    }
  })
})

describe('shiftMonth', () => {
  it('crosses the year both ways', () => {
    expect(shiftMonth(2026, 12, 1)).toEqual({ year: 2027, month: 1 })
    expect(shiftMonth(2026, 1, -1)).toEqual({ year: 2025, month: 12 })
    expect(shiftMonth(2026, 9, 0)).toEqual({ year: 2026, month: 9 })
    expect(shiftMonth(2026, 9, -21)).toEqual({ year: 2024, month: 12 })
  })
})
