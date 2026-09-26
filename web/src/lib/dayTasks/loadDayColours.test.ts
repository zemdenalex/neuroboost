import { describe, it, expect, vi } from 'vitest'
import { loadDayColours, loadDays, dayKey } from './loadDayColours'
import type { Day } from './dayColour'

const taken: Day = { day: '2026-09-23', target: 5, confirmed: true, items: [], done: 3, level: 3, before_start: false }

describe('loadDayColours', () => {
  // Review Focus 4: switched off, the calendar asks the server nothing.
  it('does not ask when day tasks are off', async () => {
    const list = vi.fn()
    const got = await loadDayColours({ enabled: false, target: 5, paintBefore: false }, '2026-09-21', '2026-09-27', '2026-09-24', list)
    expect(got).toEqual({})
    expect(list).not.toHaveBeenCalled()
  })

  it('asks once for the whole range and applies the rule', async () => {
    const list = vi.fn().mockResolvedValue([taken])
    const got = await loadDayColours({ enabled: true, target: 5, paintBefore: false }, '2026-09-21', '2026-09-27', '2026-09-24', list)
    expect(list).toHaveBeenCalledTimes(1)
    expect(list).toHaveBeenCalledWith('2026-09-21', '2026-09-27')
    expect(got).toEqual({ '2026-09-23': '🟧' })
  })

  // A failed read draws no squares rather than breaking the calendar.
  it('draws nothing when the read fails', async () => {
    const list = vi.fn().mockRejectedValue(new Error('down'))
    const got = await loadDayColours({ enabled: true, target: 5, paintBefore: false }, '2026-09-21', '2026-09-27', '2026-09-24', list)
    expect(got).toEqual({})
  })
})

describe('loadDays', () => {
  // The month's variant E and the squares share this one read.
  it('asks nothing and returns no days when day tasks are off', async () => {
    const list = vi.fn()
    expect(await loadDays({ enabled: false, target: 5, paintBefore: false }, '2026-08-31', '2026-10-11', list)).toEqual([])
    expect(list).not.toHaveBeenCalled()
  })

  it('returns the server days as they are', async () => {
    const list = vi.fn().mockResolvedValue([taken])
    expect(await loadDays({ enabled: true, target: 5, paintBefore: false }, '2026-08-31', '2026-10-11', list)).toEqual([taken])
    expect(list).toHaveBeenCalledTimes(1)
  })

  it('returns no days when the read fails', async () => {
    const list = vi.fn().mockRejectedValue(new Error('down'))
    expect(await loadDays({ enabled: true, target: 5, paintBefore: false }, '2026-08-31', '2026-10-11', list)).toEqual([])
  })
})

describe('dayKey', () => {
  // The week grid's columns are LOCAL midnights as UTC instants
  // (getMidnightUtcMs): Moscow's 24 September starts at 23 Sep 21:00Z. Slicing
  // the UTC date put every square one day early in any zone east of UTC.
  it('turns a column midnight into its date in the user zone', () => {
    expect(dayKey(Date.UTC(2026, 8, 23, 21), 'Europe/Moscow')).toBe('2026-09-24')
    expect(dayKey(Date.UTC(2026, 8, 24, 4), 'America/Los_Angeles')).toBe('2026-09-24')
    expect(dayKey(Date.UTC(2026, 8, 24), 'UTC')).toBe('2026-09-24')
  })
})
