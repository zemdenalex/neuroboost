import { describe, it, expect } from 'vitest'
import { initialScrollHour } from './initialScroll'

describe('initialScrollHour', () => {
  it('opens today one hour before now', () => {
    expect(initialScrollHour({ nowHour: 17, todayVisible: true, workStart: '09:00' })).toBe(16)
  })
  it('never goes above midnight', () => {
    expect(initialScrollHour({ nowHour: 0, todayVisible: true })).toBe(0)
  })
  it('opens another period at the start of the working day', () => {
    expect(initialScrollHour({ nowHour: 17, todayVisible: false, workStart: '09:30' })).toBe(9)
  })
  it('falls back to 08:00 without a usable work start', () => {
    expect(initialScrollHour({ nowHour: 17, todayVisible: false })).toBe(8)
    expect(initialScrollHour({ nowHour: 17, todayVisible: false, workStart: 'soon' })).toBe(8)
    expect(initialScrollHour({ nowHour: 17, todayVisible: false, workStart: '99:00' })).toBe(8)
  })
})
