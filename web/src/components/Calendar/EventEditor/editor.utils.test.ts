import { describe, it, expect } from 'vitest'
import {
  parseTimeInput,
  getAdjustedEndDate,
  validateDateRange,
  utcToLocalDateTime,
  localDateTimeToUtc,
} from './editor.utils'

describe('parseTimeInput', () => {
  it('parses HHMM / HMM digit forms', () => {
    expect(parseTimeInput('1050')).toBe('10:50')
    expect(parseTimeInput('0930')).toBe('09:30')
    expect(parseTimeInput('930')).toBe('09:30')
  })

  it('parses colon forms and zero-pads', () => {
    expect(parseTimeInput('10:50')).toBe('10:50')
    expect(parseTimeInput('9:30')).toBe('09:30')
    expect(parseTimeInput('  10:50 ')).toBe('10:50') // whitespace stripped
  })

  it('rejects out-of-range and unparseable input', () => {
    expect(parseTimeInput('')).toBeNull()
    expect(parseTimeInput('abc')).toBeNull()
    expect(parseTimeInput('1060')).toBeNull() // minutes > 59
    expect(parseTimeInput('2400')).toBeNull() // hours > 23
    expect(parseTimeInput('99')).toBeNull() // ambiguous, no colon
  })
})

describe('getAdjustedEndDate', () => {
  // Pushes the end date to the next calendar day for a same-day cross-midnight
  // event. Must be timezone-independent — the old impl mixed a local-parsed Date
  // with toISOString(), so positive-offset users (e.g. Moscow) got the wrong day.
  it('advances to the next day for a same-day cross-midnight event', () => {
    expect(getAdjustedEndDate('2026-06-15', '2026-06-15', '23:00', '01:00')).toBe('2026-06-16')
  })

  it('crosses month and year boundaries correctly', () => {
    expect(getAdjustedEndDate('2026-06-30', '2026-06-30', '23:00', '00:30')).toBe('2026-07-01')
    expect(getAdjustedEndDate('2026-12-31', '2026-12-31', '23:30', '00:30')).toBe('2027-01-01')
  })

  it('leaves the end date unchanged when not cross-midnight', () => {
    expect(getAdjustedEndDate('2026-06-15', '2026-06-15', '09:00', '10:00')).toBe('2026-06-15')
  })

  it('leaves a multi-day event (different dates) unchanged', () => {
    expect(getAdjustedEndDate('2026-06-15', '2026-06-16', '23:00', '01:00')).toBe('2026-06-16')
  })
})

describe('validateDateRange', () => {
  it('returns valid when any field is empty (nothing to validate yet)', () => {
    expect(validateDateRange('', '', '', '', 'UTC').valid).toBe(true)
  })

  it('accepts a normal same-day range', () => {
    const r = validateDateRange('2026-06-15', '2026-06-15', '09:00', '10:00', 'UTC')
    expect(r.valid).toBe(true)
    expect(r.isCrossMidnight).toBe(false)
  })

  it('treats a same-day end-before-start as a valid cross-midnight span', () => {
    const r = validateDateRange('2026-06-15', '2026-06-15', '23:00', '01:00', 'Europe/Moscow')
    expect(r.isCrossMidnight).toBe(true)
    expect(r.valid).toBe(true)
  })

  it('rejects equal start and end', () => {
    const r = validateDateRange('2026-06-15', '2026-06-15', '09:00', '09:00', 'UTC')
    expect(r.valid).toBe(false)
  })

  it('rejects an end date before the start date', () => {
    const r = validateDateRange('2026-06-16', '2026-06-15', '09:00', '10:00', 'UTC')
    expect(r.valid).toBe(false)
  })
})

describe('utcToLocalDateTime / localDateTimeToUtc', () => {
  it('converts a UTC instant to local wall time', () => {
    expect(utcToLocalDateTime(new Date('2026-06-15T12:00:00Z'), 'Europe/Moscow')).toEqual({
      date: '2026-06-15',
      time: '15:00',
    })
    expect(utcToLocalDateTime(new Date('2026-06-15T12:00:00Z'), 'UTC')).toEqual({
      date: '2026-06-15',
      time: '12:00',
    })
  })

  it('round-trips local → UTC → local', () => {
    const utc = localDateTimeToUtc('2026-06-15', '15:00', 'Europe/Moscow')
    expect(utc.toISOString()).toBe('2026-06-15T12:00:00.000Z')
    expect(utcToLocalDateTime(utc, 'Europe/Moscow')).toEqual({ date: '2026-06-15', time: '15:00' })
  })
})
