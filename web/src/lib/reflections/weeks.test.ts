import { describe, it, expect } from 'vitest'
import { isoWeekKey, weekStart, average, isWithinRange } from './weeks'

// The Reflections page groups by ISO week (queue: tests for pages without any).
// Year boundaries are where hand-written week arithmetic goes wrong; the
// expected keys are the ISO calendar's, not this code's.
const noon = (d: string) => `${d}T12:00:00`

describe('isoWeekKey', () => {
  it.each([
    ['2026-01-01', '2026-W01'], // a Thursday: week 1 of its own year
    ['2027-01-01', '2026-W53'], // a Friday: the last week of the year before
    ['2024-12-30', '2025-W01'], // a Monday in December: week 1 of the next year
    ['2026-09-21', '2026-W39'],
    ['2026-09-27', '2026-W39'], // Sunday closes the week that Monday opened
    ['2026-09-28', '2026-W40'],
  ])('%s is %s', (day, key) => {
    expect(isoWeekKey(noon(day))).toBe(key)
  })
})

describe('weekStart', () => {
  it('is the Monday of the week, across a year boundary too', () => {
    const iso = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    expect(iso(weekStart('2026-W39'))).toBe('2026-09-21')
    expect(iso(weekStart('2025-W01'))).toBe('2024-12-30')
    expect(iso(weekStart('2026-W53'))).toBe('2026-12-28')
  })
})

describe('average', () => {
  it('ignores missing values and rounds to one decimal', () => {
    expect(average([7, null, 8, 8])).toBe(7.7)
    expect(average([null, null])).toBeNull()
  })
})

describe('isWithinRange', () => {
  const now = new Date(2026, 8, 26, 12)
  it('week is the last seven days, month the last month, all is all', () => {
    expect(isWithinRange(new Date(2026, 8, 20, 12).toISOString(), 'week', now)).toBe(true)
    expect(isWithinRange(new Date(2026, 8, 18, 12).toISOString(), 'week', now)).toBe(false)
    expect(isWithinRange(new Date(2026, 7, 27, 12).toISOString(), 'month', now)).toBe(true)
    expect(isWithinRange(new Date(2026, 7, 25, 12).toISOString(), 'month', now)).toBe(false)
    expect(isWithinRange('2020-01-01T00:00:00Z', 'all', now)).toBe(true)
  })
})
