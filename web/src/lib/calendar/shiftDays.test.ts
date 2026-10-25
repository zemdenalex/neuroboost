import { describe, it, expect } from 'vitest'
import { daysBetween, shiftByDays, localTimeOn } from './shiftDays'

describe('daysBetween', () => {
  it('counts calendar days, forward and back', () => {
    expect(daysBetween('2026-09-24', '2026-09-27')).toBe(3)
    expect(daysBetween('2026-09-27', '2026-09-24')).toBe(-3)
    expect(daysBetween('2026-10-24', '2026-10-26')).toBe(2)
  })
})

describe('shiftByDays', () => {
  it('keeps the wall-clock time and the length in a zone without DST', () => {
    // 10:00–11:30 Moscow on the 24th, moved two days on.
    expect(shiftByDays('2026-09-24T07:00:00.000Z', '2026-09-24T08:30:00.000Z', 2, 'Europe/Moscow')).toEqual({
      startsAt: '2026-09-26T07:00:00.000Z',
      endsAt: '2026-09-26T08:30:00.000Z',
    })
  })

  it('keeps 10:00 at 10:00 across a DST change (Berlin, 25 October 2026)', () => {
    // Saturday 24 Oct 10:00 CEST (UTC+2) → Monday 26 Oct 10:00 CET (UTC+1).
    // Adding 48 hours would land at 09:00.
    expect(shiftByDays('2026-10-24T08:00:00.000Z', '2026-10-24T09:00:00.000Z', 2, 'Europe/Berlin')).toEqual({
      startsAt: '2026-10-26T09:00:00.000Z',
      endsAt: '2026-10-26T10:00:00.000Z',
    })
  })

  it('lands right just before the spring change, where one offset guess is wrong', () => {
    // Berlin jumps 02:00 → 03:00 on 29 March 2026 (01:00Z). 01:30 on the 28th
    // (CET, 00:30Z) moved one day is 01:30 CET on the 29th, still 00:30Z. The
    // offset read at "01:30 as if UTC" is already CEST and would give 00:30 local.
    expect(shiftByDays('2026-03-28T00:30:00.000Z', '2026-03-28T00:45:00.000Z', 1, 'Europe/Berlin')).toEqual({
      startsAt: '2026-03-29T00:30:00.000Z',
      endsAt: '2026-03-29T00:45:00.000Z',
    })
  })

  it('moves backwards too', () => {
    expect(shiftByDays('2026-09-24T07:00:00.000Z', '2026-09-24T08:00:00.000Z', -1, 'Europe/Moscow')).toEqual({
      startsAt: '2026-09-23T07:00:00.000Z',
      endsAt: '2026-09-23T08:00:00.000Z',
    })
  })

  it('moves an all-day event by whole local days', () => {
    // Moscow midnight 25th to midnight 26th, moved one day.
    expect(shiftByDays('2026-09-24T21:00:00.000Z', '2026-09-25T21:00:00.000Z', 1, 'Europe/Moscow')).toEqual({
      startsAt: '2026-09-25T21:00:00.000Z',
      endsAt: '2026-09-26T21:00:00.000Z',
    })
  })
})

describe('localTimeOn', () => {
  it('is that wall-clock time on that day in the zone', () => {
    expect(localTimeOn('2026-09-24', '09:00', 'Europe/Moscow')).toBe('2026-09-24T06:00:00.000Z')
    expect(localTimeOn('2026-10-26', '10:00', 'Europe/Berlin')).toBe('2026-10-26T09:00:00.000Z')
    expect(localTimeOn('2026-10-24', '10:00', 'Europe/Berlin')).toBe('2026-10-24T08:00:00.000Z')
  })

  it('reads a bad time as 09:00', () => {
    expect(localTimeOn('2026-09-24', 'soon', 'Europe/Moscow')).toBe('2026-09-24T06:00:00.000Z')
  })
})
