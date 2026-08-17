import { describe, it, expect } from 'vitest'
import { toggleWorkDay, normaliseWorkDays, DEFAULT_WORK_DAYS, WEEK_DAYS } from './workDays'

describe('toggleWorkDay', () => {
  it('removes a day that is present', () => {
    expect(toggleWorkDay(['Mon', 'Tue', 'Wed'], 'Tue')).toEqual(['Mon', 'Wed'])
  })

  it('adds a day that is absent', () => {
    expect(toggleWorkDay(['Mon', 'Tue'], 'Wed')).toEqual(['Mon', 'Tue', 'Wed'])
  })

  // 🔴 The rule this file exists for. The previous version appended, so a day
  // switched off and on again landed at the end of the stored array.
  it('keeps ISO order when a day is switched off and back on', () => {
    const without = toggleWorkDay(DEFAULT_WORK_DAYS, 'Mon')
    expect(without).toEqual(['Tue', 'Wed', 'Thu', 'Fri'])

    const restored = toggleWorkDay(without, 'Mon')
    expect(restored, 'Monday must return to the front, not the end').toEqual([
      'Mon',
      'Tue',
      'Wed',
      'Thu',
      'Fri',
    ])
  })

  it('orders by position in the week, not alphabetically', () => {
    // "Fri" sorts before "Mon" as a string — a different wrong answer.
    expect(toggleWorkDay(['Mon'], 'Fri')).toEqual(['Mon', 'Fri'])
    expect(toggleWorkDay(['Sun'], 'Sat')).toEqual(['Sat', 'Sun'])
  })

  it('drops an unrecognised day rather than storing it', () => {
    // The button set only offers real days, but the stored blob is
    // user-writable and a junk entry must not survive a toggle.
    expect(toggleWorkDay(['Mon', 'Funday'], 'Tue')).toEqual(['Mon', 'Tue'])
  })

  // The negative control: a function that always returned the full week, or
  // always the input, would satisfy several of the tests above.
  it('can produce an empty week', () => {
    let days: readonly string[] = DEFAULT_WORK_DAYS
    for (const d of DEFAULT_WORK_DAYS) days = toggleWorkDay(days, d)
    expect(days).toEqual([])
  })
})

describe('normaliseWorkDays', () => {
  it('puts a stored array into ISO order', () => {
    expect(normaliseWorkDays(['Fri', 'Mon', 'Wed'])).toEqual(['Mon', 'Wed', 'Fri'])
  })

  it('removes duplicates', () => {
    expect(normaliseWorkDays(['Mon', 'Mon', 'Tue'])).toEqual(['Mon', 'Tue'])
  })

  it('drops junk without losing the valid days beside it', () => {
    expect(normaliseWorkDays(['Mon', 'monday', 42, null, 'Fri'])).toEqual(['Mon', 'Fri'])
  })

  it('falls back to the default working week for a non-array', () => {
    // A missing or corrupted value must render a sensible week rather than an
    // empty one, which would read as "you work no days".
    expect(normaliseWorkDays(undefined)).toEqual(DEFAULT_WORK_DAYS)
    expect(normaliseWorkDays(null)).toEqual(DEFAULT_WORK_DAYS)
    expect(normaliseWorkDays('Mon,Tue')).toEqual(DEFAULT_WORK_DAYS)
  })

  // An empty array is a real choice — someone with no fixed working days —
  // and must not be mistaken for a missing value.
  it('respects an explicitly empty array', () => {
    expect(normaliseWorkDays([])).toEqual([])
  })

  it('accepts the full week', () => {
    expect(normaliseWorkDays([...WEEK_DAYS])).toEqual([...WEEK_DAYS])
  })
})
