import { describe, it, expect, vi } from 'vitest'
import { loadDayColours, dayKey } from './loadDayColours'
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

describe('dayKey', () => {
  it('turns a column midnight (UTC) into its date', () => {
    expect(dayKey(Date.UTC(2026, 8, 24))).toBe('2026-09-24')
  })
})
