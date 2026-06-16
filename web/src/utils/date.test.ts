import { describe, it, expect } from 'vitest'
import { addDays, startOfWeek, toLocalDateKey } from './date'

describe('addDays', () => {
  it('adds days, preserves time, does not mutate input', () => {
    const d = new Date(2026, 5, 16, 10, 30)
    const r = addDays(d, 3)
    expect(r.getDate()).toBe(19)
    expect(r.getHours()).toBe(10)
    expect(r.getMinutes()).toBe(30)
    expect(d.getDate()).toBe(16) // input unchanged
  })

  it('subtracts with a negative count across a month boundary', () => {
    expect(addDays(new Date(2026, 5, 2), -5).getMonth()).toBe(4) // June 2 → May 28
  })

  it('crosses a year boundary', () => {
    const r = addDays(new Date(2026, 11, 30), 5) // Dec 30 → Jan 4
    expect(r.getFullYear()).toBe(2027)
    expect(r.getMonth()).toBe(0)
    expect(r.getDate()).toBe(4)
  })
})

describe('startOfWeek', () => {
  it('returns the Monday at local midnight for every weekday', () => {
    for (let i = 0; i < 14; i++) {
      const d = new Date(2026, 5, 10 + i, 13, 45, 30, 500)
      const mon = startOfWeek(d)
      expect(mon.getDay()).toBe(1) // Monday
      expect([mon.getHours(), mon.getMinutes(), mon.getSeconds(), mon.getMilliseconds()]).toEqual([0, 0, 0, 0])
      expect(mon.getTime()).toBeLessThanOrEqual(d.getTime())
      const dayStart = new Date(d)
      dayStart.setHours(0, 0, 0, 0)
      const backDays = Math.round((dayStart.getTime() - mon.getTime()) / 86_400_000)
      expect(backDays).toBeGreaterThanOrEqual(0)
      expect(backDays).toBeLessThanOrEqual(6)
    }
  })

  it('treats Sunday as the last day of the week (returns the previous Monday)', () => {
    const sun = new Date(2026, 5, 21, 9, 0) // a Sunday
    expect(startOfWeek(sun).getDay()).toBe(1)
    expect(sun.getTime() - startOfWeek(sun).getTime()).toBeGreaterThan(0)
  })

  it('is idempotent — a Monday maps to itself', () => {
    const mon = startOfWeek(new Date(2026, 5, 17, 13, 0))
    expect(startOfWeek(mon).getTime()).toBe(mon.getTime())
  })

  it('does not mutate its input', () => {
    const d = new Date(2026, 5, 17, 13, 45)
    const before = d.getTime()
    startOfWeek(d)
    expect(d.getTime()).toBe(before)
  })
})

describe('toLocalDateKey', () => {
  it('formats local Y-M-D, zero-padded', () => {
    expect(toLocalDateKey(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05')
    expect(toLocalDateKey(new Date(2026, 11, 31, 0, 0))).toBe('2026-12-31')
  })
})
