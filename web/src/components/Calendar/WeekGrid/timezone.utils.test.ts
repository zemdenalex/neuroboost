import { describe, it, expect } from 'vitest'
import {
  getTimezoneOffsetMs,
  getMidnightUtcMs,
  getMondayUtcMs,
  utcToLocalMinutes,
  getDayIndex,
  formatTimeFromUtc,
} from './timezone.utils'

// No-DST zones → expected values are stable year-round, runtime-tz-independent
// (Intl with an explicit timeZone is deterministic).
const HOUR = 3_600_000
const MONDAY_UTC = Date.UTC(2026, 5, 15, 0, 0, 0, 0) // 2026-06-15 is a Monday
const REF = new Date('2026-06-15T12:00:00Z')

describe('getTimezoneOffsetMs', () => {
  it('returns the fixed offset per zone', () => {
    expect(getTimezoneOffsetMs('UTC', REF)).toBe(0)
    expect(getTimezoneOffsetMs('Europe/Moscow', REF)).toBe(3 * HOUR) // UTC+3, no DST
    expect(getTimezoneOffsetMs('Asia/Kolkata', REF)).toBe(5.5 * HOUR) // UTC+5:30
    expect(getTimezoneOffsetMs('Etc/GMT+5', REF)).toBe(-5 * HOUR) // POSIX sign: UTC-5
  })
})

describe('utcToLocalMinutes', () => {
  it('converts a UTC instant to local minutes-since-midnight', () => {
    expect(utcToLocalMinutes('2026-06-15T12:00:00Z', 'UTC')).toBe(720) // 12:00
    expect(utcToLocalMinutes('2026-06-15T12:00:00Z', 'Europe/Moscow')).toBe(900) // 15:00
    expect(utcToLocalMinutes('2026-06-15T12:00:00Z', 'Asia/Kolkata')).toBe(1050) // 17:30
  })

  it('wraps across the local day boundary for negative offsets', () => {
    // 02:00Z in UTC-5 is 21:00 the previous local day.
    expect(utcToLocalMinutes('2026-06-15T02:00:00Z', 'Etc/GMT+5')).toBe(21 * 60)
  })

  it('returns 0 at local midnight and 1410 at 23:30', () => {
    expect(utcToLocalMinutes('2026-06-15T00:00:00Z', 'UTC')).toBe(0)
    expect(utcToLocalMinutes('2026-06-15T23:30:00Z', 'UTC')).toBe(1410)
  })
})

describe('getDayIndex', () => {
  it('returns days since the reference Monday', () => {
    expect(getDayIndex('2026-06-15T12:00:00Z', MONDAY_UTC, 'UTC')).toBe(0)
    expect(getDayIndex('2026-06-17T08:00:00Z', MONDAY_UTC, 'UTC')).toBe(2)
    expect(getDayIndex('2026-06-21T23:00:00Z', MONDAY_UTC, 'UTC')).toBe(6)
  })

  it('is offset-invariant when mondayUtc0 already encodes the day-0 boundary', () => {
    // The offset cancels in localMs - mondayLocalMs, so the tz arg does not change the index.
    expect(getDayIndex('2026-06-17T08:00:00Z', MONDAY_UTC, 'Europe/Moscow')).toBe(2)
    expect(getDayIndex('2026-06-17T08:00:00Z', MONDAY_UTC, 'Asia/Kolkata')).toBe(2)
  })
})

describe('getMidnightUtcMs', () => {
  it('returns the UTC instant of local midnight', () => {
    expect(getMidnightUtcMs(REF.getTime(), 'UTC')).toBe(Date.UTC(2026, 5, 15, 0, 0))
    // 12:00Z is 15:00 in Moscow → local midnight is 2026-06-15 00:00 (+3) = 2026-06-14 21:00 UTC
    expect(getMidnightUtcMs(REF.getTime(), 'Europe/Moscow')).toBe(Date.UTC(2026, 5, 14, 21, 0))
    // +5:30 → local midnight = 2026-06-14 18:30 UTC
    expect(getMidnightUtcMs(REF.getTime(), 'Asia/Kolkata')).toBe(Date.UTC(2026, 5, 14, 18, 30))
    // -5 → 12:00Z is 07:00 local → local midnight = 2026-06-15 05:00 UTC
    expect(getMidnightUtcMs(REF.getTime(), 'Etc/GMT+5')).toBe(Date.UTC(2026, 5, 15, 5, 0))
  })
})

describe('getMondayUtcMs', () => {
  it('finds the Monday of the reference week in UTC', () => {
    const wed = new Date('2026-06-17T12:00:00Z')
    expect(getMondayUtcMs(wed, 0, 'UTC')).toBe(Date.UTC(2026, 5, 15, 0, 0))
    expect(getMondayUtcMs(wed, -1, 'UTC')).toBe(Date.UTC(2026, 5, 8, 0, 0))
    expect(getMondayUtcMs(wed, 1, 'UTC')).toBe(Date.UTC(2026, 5, 22, 0, 0))
  })

  it('treats Sunday as the last day of the week', () => {
    const sun = new Date('2026-06-21T12:00:00Z')
    expect(getMondayUtcMs(sun, 0, 'UTC')).toBe(Date.UTC(2026, 5, 15, 0, 0))
  })

  it('anchors to local Monday midnight for an offset zone', () => {
    const wed = new Date('2026-06-17T12:00:00Z')
    // Monday 00:00 Moscow time = 2026-06-14 21:00 UTC
    expect(getMondayUtcMs(wed, 0, 'Europe/Moscow')).toBe(Date.UTC(2026, 5, 14, 21, 0))
  })
})

describe('formatTimeFromUtc', () => {
  it('formats HH:MM in the target zone', () => {
    expect(formatTimeFromUtc('2026-06-15T12:00:00Z', 'UTC')).toBe('12:00')
    expect(formatTimeFromUtc('2026-06-15T12:00:00Z', 'Europe/Moscow')).toBe('15:00')
    expect(formatTimeFromUtc('2026-06-15T12:00:00Z', 'Asia/Kolkata')).toBe('17:30')
  })
})
