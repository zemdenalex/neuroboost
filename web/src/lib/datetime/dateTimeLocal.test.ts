import { describe, it, expect } from 'vitest'
import { toDateTimeLocalValue, fromDateTimeLocalValue } from './dateTimeLocal'

describe('datetime-local conversion', () => {
  // The core property the bug violated: showing a due date in the picker and
  // saving it again (without changing it) must not shift the instant. Holds in
  // any timezone because both directions use the same (browser-local) frame.
  it('round-trips a UTC instant through the picker value without drift', () => {
    for (const iso of [
      '2026-06-17T09:00:00.000Z',
      '2026-01-01T23:30:00.000Z',
      '2026-07-04T00:15:00.000Z',
      '2025-12-31T18:45:00.000Z',
    ]) {
      expect(fromDateTimeLocalValue(toDateTimeLocalValue(iso))).toBe(iso)
    }
  })

  // Inverse direction: a picker value survives a UTC round-trip unchanged
  // (avoids DST-gap local times like spring-forward 02:00–03:00).
  it('round-trips a local picker value through UTC and back', () => {
    for (const local of ['2026-06-17T12:00', '2026-12-31T23:45', '2026-09-10T08:05']) {
      expect(toDateTimeLocalValue(fromDateTimeLocalValue(local))).toBe(local)
    }
  })

  // The picker value reflects LOCAL wall-clock, not the raw UTC slice (the bug
  // used `.toISOString().slice(0,16)`, i.e. UTC, while the input is local).
  it('formats local wall-clock, not the UTC slice', () => {
    const iso = '2026-06-17T09:00:00.000Z'
    const d = new Date(iso)
    const pad = (n: number) => String(n).padStart(2, '0')
    const expected = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
    expect(toDateTimeLocalValue(iso)).toBe(expected)
  })

  it('returns an empty string for empty input (no due date)', () => {
    expect(toDateTimeLocalValue('')).toBe('')
    expect(fromDateTimeLocalValue('')).toBe('')
  })
})
