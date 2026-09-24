import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  MONTH_VARIANTS,
  readMonthVariant,
  readCalendarView,
  saveCalendarView,
  effectiveView,
} from './monthVariant'

afterEach(() => {
  vi.restoreAllMocks()
  try {
    localStorage.clear()
  } catch {
    /* jsdom */
  }
})

describe('readMonthVariant', () => {
  it('is "list" when nothing is set', () => {
    expect(readMonthVariant(undefined)).toBe('list')
    expect(readMonthVariant({})).toBe('list')
  })

  it('reads each of the five variants', () => {
    for (const v of MONTH_VARIANTS) {
      expect(readMonthVariant({ month_view_variant: v })).toBe(v)
    }
    expect(MONTH_VARIANTS).toEqual(['list', 'classic', 'heat', 'split', 'commit'])
  })

  it('falls back to "list" for a value it does not know', () => {
    expect(readMonthVariant({ month_view_variant: 'grid' })).toBe('list')
    expect(readMonthVariant({ month_view_variant: 3 })).toBe('list')
  })
})

describe('calendar view on this device', () => {
  it('is the week until something is saved', () => {
    expect(readCalendarView()).toBe('week')
  })

  it('remembers the month', () => {
    saveCalendarView('month')
    expect(readCalendarView()).toBe('month')
    saveCalendarView('week')
    expect(readCalendarView()).toBe('week')
  })

  it('reads a stray value as the week', () => {
    localStorage.setItem('nb-calendar-view', 'year')
    expect(readCalendarView()).toBe('week')
  })

  it('survives a storage that throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    expect(() => saveCalendarView('month')).not.toThrow()
    expect(readCalendarView()).toBe('week')
  })
})

describe('effectiveView', () => {
  it('keeps the saved view on a desktop', () => {
    expect(effectiveView('month', false)).toBe('month')
    expect(effectiveView('week', false)).toBe('week')
  })

  it('never opens the month on a phone: the switch is hidden there', () => {
    expect(effectiveView('month', true)).toBe('week')
  })
})
