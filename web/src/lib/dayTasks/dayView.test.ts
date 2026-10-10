import { describe, it, expect } from 'vitest'
import { readDayPrefs, shiftDay, dayWhen, canRemove, errorKey, weekDays, hourInZone } from './dayView'
import { ApiError } from '../../api/client'

describe('readDayPrefs', () => {
  // No key = on, 5, not painting: the bot's defaults (spec §11).
  it('defaults', () => {
    expect(readDayPrefs(undefined)).toEqual({ enabled: true, target: 5, paintBefore: false })
    expect(readDayPrefs({})).toEqual({ enabled: true, target: 5, paintBefore: false })
  })

  it('reads the three top-level keys the bot writes', () => {
    expect(readDayPrefs({ day_tasks_enabled: false, day_tasks_target: 3, day_tasks_paint_before: true })).toEqual({
      enabled: false,
      target: 3,
      paintBefore: true,
    })
  })

  it('ignores a target outside 3 to 7', () => {
    expect(readDayPrefs({ day_tasks_target: 9 }).target).toBe(5)
  })
})

describe('shiftDay', () => {
  it('crosses months and years', () => {
    expect(shiftDay('2026-09-30', 1)).toBe('2026-10-01')
    expect(shiftDay('2027-01-01', -1)).toBe('2026-12-31')
  })
})

describe('dayWhen / canRemove', () => {
  it('sorts a day into past, today, future', () => {
    expect(dayWhen('2026-09-23', '2026-09-24')).toBe('past')
    expect(dayWhen('2026-09-24', '2026-09-24')).toBe('today')
    expect(dayWhen('2026-09-25', '2026-09-24')).toBe('future')
  })

  // Spec §5: today until 12:00, the future always, the past never.
  it('allows removing as the server does', () => {
    expect(canRemove('2026-09-24', '2026-09-24', 11)).toBe(true)
    expect(canRemove('2026-09-24', '2026-09-24', 12)).toBe(false)
    expect(canRemove('2026-09-25', '2026-09-24', 23)).toBe(true)
    expect(canRemove('2026-09-23', '2026-09-24', 1)).toBe(false)
  })
})

describe('errorKey', () => {
  // Review Focus 5: a refusal is a sentence, not «Something went wrong».
  it('names the server refusals', () => {
    expect(errorKey(new ApiError('x', 'TOO_LATE', undefined))).toBe('errors.tooLate')
    expect(errorKey(new ApiError('x', 'NOT_OPEN', undefined))).toBe('errors.notOpen')
    expect(errorKey(new ApiError('x', 'TASK_NOT_FOUND', undefined))).toBe('errors.notFound')
    expect(errorKey(new Error('network'))).toBe('errors.generic')
  })
})

describe('weekDays', () => {
  it('lists seven days from a Monday', () => {
    const w = weekDays('2026-09-21')
    expect(w).toHaveLength(7)
    expect(w[0]).toBe('2026-09-21')
    expect(w[6]).toBe('2026-09-27')
  })
})

describe('hourInZone', () => {
  // Noon is Moscow's noon, not the browser's (spec §5, «whose today?»).
  it("is the person's hour", () => {
    const now = new Date('2026-09-24T09:30:00Z')
    expect(hourInZone(now, 'Europe/Moscow')).toBe(12)
    expect(hourInZone(now, 'UTC')).toBe(9)
  })
})
