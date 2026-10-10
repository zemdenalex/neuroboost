import { describe, it, expect } from 'vitest'
import { allDayBarHeight, takeCalendarHint, HINT_SHOWS } from './calendarChrome'

describe('allDayBarHeight (Denis 25.09, variant A)', () => {
  it('is a thin strip on a phone while the day has nothing all-day', () => {
    expect(allDayBarHeight({ isMobile: true, allDayCount: 0 })).toBe(20)
  })
  it('is full height as soon as there is an all-day event, and always on a desktop', () => {
    expect(allDayBarHeight({ isMobile: true, allDayCount: 1 })).toBe(80)
    expect(allDayBarHeight({ isMobile: false, allDayCount: 0 })).toBe(80)
  })
  it('opens while an all-day event is being created, so its ghost has room', () => {
    expect(allDayBarHeight({ isMobile: true, allDayCount: 0, creatingAllDay: true })).toBe(80)
  })
})

describe('takeCalendarHint', () => {
  function memory(initial: Record<string, string> = {}) {
    const m = new Map(Object.entries(initial))
    return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v) }
  }
  it('shows the tip on the first three opens, then stops', () => {
    const s = memory()
    const shown = Array.from({ length: 5 }, () => takeCalendarHint(s))
    expect(HINT_SHOWS).toBe(3)
    expect(shown).toEqual([true, true, true, false, false])
  })
  it('treats unreadable storage as a first open rather than hiding the tip forever', () => {
    expect(takeCalendarHint(memory({ 'nb-calendar-hint-shown': 'garbage' }))).toBe(true)
  })
  it('still shows the tip when storage throws', () => {
    const broken = {
      getItem: () => {
        throw new Error('blocked')
      },
      setItem: () => {
        throw new Error('blocked')
      },
    }
    expect(takeCalendarHint(broken)).toBe(true)
  })
})
