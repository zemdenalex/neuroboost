import { describe, it, expect } from 'vitest'
import { defaultScheduleSlot } from './defaultScheduleSlot'

const HOUR = 60 * 60 * 1000

describe('defaultScheduleSlot', () => {
  it('starts at the next full hour, strictly after now and within an hour', () => {
    const now = new Date('2026-06-17T11:23:45.000Z')
    const start = new Date(defaultScheduleSlot(now).startsAt)
    expect(start.getTime()).toBeGreaterThan(now.getTime())
    expect(start.getTime() - now.getTime()).toBeLessThanOrEqual(HOUR)
    // on the hour (local), no stray minutes/seconds/ms
    expect(start.getMinutes()).toBe(0)
    expect(start.getSeconds()).toBe(0)
    expect(start.getMilliseconds()).toBe(0)
  })

  it('advances a full hour even when now is exactly on the hour', () => {
    const now = new Date('2026-06-17T11:00:00.000Z')
    const start = new Date(defaultScheduleSlot(now).startsAt)
    expect(start.getTime() - now.getTime()).toBe(HOUR)
  })

  it('uses the estimate as duration, with a 15-minute floor and 60-minute default', () => {
    const now = new Date('2026-06-17T11:23:00.000Z')
    const dur = (est?: number) => {
      const s = defaultScheduleSlot(now, est)
      return (new Date(s.endsAt).getTime() - new Date(s.startsAt).getTime()) / 60000
    }
    expect(dur(30)).toBe(30)
    expect(dur(120)).toBe(120)
    expect(dur(5)).toBe(15) // floor
    expect(dur(0)).toBe(60) // zero/falsey -> default
    expect(dur(undefined)).toBe(60)
  })
})
