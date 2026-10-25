import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  MONTH_VARIANTS,
  readMonthVariant,
  readCalendarView,
  saveCalendarView,
  readPhoneMonthVariant,
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

// The phone reads its own key: a desktop choice ("list", "classic", …) is not
// a phone variant and must not leave the phone with a month it cannot draw.
describe('readPhoneMonthVariant', () => {
  it('is A ("split") when unset, unknown or a desktop-only variant', () => {
    expect(readPhoneMonthVariant(undefined)).toBe('split')
    expect(readPhoneMonthVariant({ phone_month_variant: 'list' })).toBe('split')
  })

  it('keeps a phone variant', () => {
    expect(readPhoneMonthVariant({ phone_month_variant: 'strip' })).toBe('strip')
  })
})
