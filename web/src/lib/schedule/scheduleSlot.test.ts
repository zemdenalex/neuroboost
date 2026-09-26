import { describe, it, expect } from 'vitest'
import { scheduleStart, scheduleDurations, whenShort, linkedStarts, SCHEDULE_SLOTS, instantFromLocalValue, localValueOf } from './scheduleSlot'

const MSK = 'Europe/Moscow' // UTC+3, no DST
const BER = 'Europe/Berlin' // DST ends 2026-10-25 03:00 → 02:00

describe('scheduleStart (mirrors bot/internal/handlers/schedule.go)', () => {
  const now = new Date('2026-09-26T11:23:45.500Z') // 14:23:45 in Moscow

  it('offers the four slots of the bot, in its order', () => {
    expect(SCHEDULE_SLOTS).toEqual(['now', 'hour', 'eve', 'tmr'])
  })

  it('now = this minute', () => {
    expect(scheduleStart('now', now, MSK).toISOString()).toBe('2026-09-26T11:23:00.000Z')
  })

  it('in an hour = an hour from now, to the minute', () => {
    expect(scheduleStart('hour', now, MSK).toISOString()).toBe('2026-09-26T12:23:00.000Z')
  })

  it('this evening = 19:00 in the person’s zone', () => {
    expect(scheduleStart('eve', now, MSK).toISOString()).toBe('2026-09-26T16:00:00.000Z')
  })

  it('this evening pressed at exactly 19:00 means tomorrow’s evening', () => {
    const at19 = new Date('2026-09-26T16:00:00.000Z')
    expect(scheduleStart('eve', at19, MSK).toISOString()).toBe('2026-09-27T16:00:00.000Z')
  })

  it('this evening pressed at 21:00 means tomorrow’s evening', () => {
    const at21 = new Date('2026-09-26T18:00:00.000Z')
    expect(scheduleStart('eve', at21, MSK).toISOString()).toBe('2026-09-27T16:00:00.000Z')
  })

  it('tomorrow morning = 09:00 tomorrow in the person’s zone', () => {
    expect(scheduleStart('tmr', now, MSK).toISOString()).toBe('2026-09-27T06:00:00.000Z')
  })

  it('tomorrow is the zone’s tomorrow, not the machine’s: 00:30 Moscow is still the 27th there', () => {
    const lateUtc = new Date('2026-09-26T21:30:00.000Z') // 00:30 on the 27th in Moscow
    expect(scheduleStart('tmr', lateUtc, MSK).toISOString()).toBe('2026-09-28T06:00:00.000Z')
  })

  it('tomorrow morning across the autumn DST change keeps 09:00 local', () => {
    const sat = new Date('2026-10-24T10:00:00.000Z') // Sat 12:00 CEST
    // Sun 25.10 09:00 is CET (UTC+1)
    expect(scheduleStart('tmr', sat, BER).toISOString()).toBe('2026-10-25T08:00:00.000Z')
  })

  it('this evening across the DST change rolls to 19:00 CET', () => {
    const satLate = new Date('2026-10-24T18:00:00.000Z') // Sat 20:00 CEST
    expect(scheduleStart('eve', satLate, BER).toISOString()).toBe('2026-10-25T18:00:00.000Z')
  })

  it('an unknown zone falls back to Moscow, the server default', () => {
    expect(scheduleStart('tmr', now, 'Not/AZone').toISOString()).toBe('2026-09-27T06:00:00.000Z')
  })
})

describe('scheduleDurations', () => {
  it('is the bot’s four when there is no estimate', () => {
    expect(scheduleDurations(undefined)).toEqual({ options: [15, 30, 60, 120], preferred: undefined })
  })

  it('prefers the estimate when it is one of the four', () => {
    expect(scheduleDurations(30)).toEqual({ options: [15, 30, 60, 120], preferred: 30 })
  })

  it('adds an estimate the four do not have, in order', () => {
    expect(scheduleDurations(45)).toEqual({ options: [15, 30, 45, 60, 120], preferred: 45 })
  })

  it('ignores an estimate that is not a usable length', () => {
    expect(scheduleDurations(0).preferred).toBeUndefined()
    expect(scheduleDurations(24 * 60 + 1).preferred).toBeUndefined()
  })
})

describe('whenShort (mirrors bot linked.go whenShort)', () => {
  const now = new Date('2026-09-26T11:00:00.000Z') // Sat 14:00 Moscow

  it('today', () => {
    expect(whenShort(new Date('2026-09-26T16:00:00.000Z'), now, MSK, 'ru')).toBe('сегодня 19:00')
    expect(whenShort(new Date('2026-09-26T16:00:00.000Z'), now, MSK, 'en')).toBe('today 19:00')
  })

  it('tomorrow', () => {
    expect(whenShort(new Date('2026-09-27T06:00:00.000Z'), now, MSK, 'ru')).toBe('завтра 09:00')
    expect(whenShort(new Date('2026-09-27T06:00:00.000Z'), now, MSK, 'en')).toBe('tomorrow 09:00')
  })

  it('within the week: weekday', () => {
    // Wed 30.09 15:00 Moscow
    expect(whenShort(new Date('2026-09-30T12:00:00.000Z'), now, MSK, 'ru')).toBe('ср 15:00')
    expect(whenShort(new Date('2026-09-30T12:00:00.000Z'), now, MSK, 'en')).toBe('We 15:00')
  })

  it('further: dd.mm', () => {
    expect(whenShort(new Date('2026-10-22T12:00:00.000Z'), now, MSK, 'ru')).toBe('22.10 15:00')
  })

  it('counts days by the calendar in the zone, not by 24-hour blocks (a 23-hour DST day)', () => {
    // Sun 29.03.2026 is 23 hours long in Berlin; the bot's Hours()/24 reads Monday as «today».
    const sun = new Date('2026-03-28T23:30:00.000Z') // Sun 00:30 CET
    const monMorning = new Date('2026-03-30T07:00:00.000Z') // Mon 09:00 CEST
    expect(whenShort(monMorning, sun, BER, 'ru')).toBe('завтра 09:00')
  })
})

describe('linkedStarts (mirrors bot linked.go linkedEvents)', () => {
  const now = new Date('2026-09-26T11:00:00.000Z') // 14:00 Moscow

  it('takes the nearest linked event starting today or later', () => {
    const map = linkedStarts(
      [
        { task_id: 'a', starts_at: '2026-09-28T06:00:00Z' },
        { task_id: 'a', starts_at: '2026-09-27T06:00:00Z' },
        { task_id: 'b', starts_at: '2026-09-26T05:00:00Z' }, // 08:00 today, already past: still today
        { task_id: 'c', starts_at: '2026-09-25T20:00:00Z' }, // 23:00 yesterday
        { starts_at: '2026-09-27T06:00:00Z' },
        { task_id: '', starts_at: '2026-09-27T06:00:00Z' },
      ],
      now,
      MSK,
    )
    expect(map.get('a')).toBe('2026-09-27T06:00:00Z')
    expect(map.get('b')).toBe('2026-09-26T05:00:00Z')
    expect(map.has('c')).toBe(false)
    expect(map.size).toBe(2)
  })
})

describe('a typed time (the link sheet\'s «другое время»)', () => {
  it('is a wall time in the person\'s zone, not the browser\'s, and reads back the same', () => {
    expect(instantFromLocalValue('2026-10-02T15:00', MSK)?.toISOString()).toBe('2026-10-02T12:00:00.000Z')
    // Berlin after the DST change: +1, not the +2 of the day it was typed.
    expect(instantFromLocalValue('2026-10-26T09:00', BER)?.toISOString()).toBe('2026-10-26T08:00:00.000Z')
    expect(localValueOf(new Date('2026-10-02T12:00:00Z'), MSK)).toBe('2026-10-02T15:00')
    expect(instantFromLocalValue('2026-10-02', MSK)).toBeNull()
  })
})
