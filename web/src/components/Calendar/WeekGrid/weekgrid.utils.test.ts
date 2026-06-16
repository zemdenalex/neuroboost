import { describe, it, expect } from 'vitest'
import {
  processEventsForWeek,
  getDaySpan,
  minsToTop,
  topToMins,
  snapMin,
  clampMins,
  formatMinutesToTime,
} from './weekgrid.utils'
import { DAY_MS, MIN_SLOT_MIN } from './weekgrid.constants'
import type { NbEvent } from './weekgrid.types'

const TZ = 'UTC' // offset 0 → local minutes == UTC minutes, fully deterministic
const MONDAY = Date.UTC(2026, 5, 15, 0, 0, 0, 0) // day-0 reference

function ev(id: string, startsAt: string, endsAt: string, allDay = false): NbEvent {
  return { id, title: id, startsAt, endsAt, allDay }
}

describe('processEventsForWeek — timed events', () => {
  it('positions a single-day timed event', () => {
    const { timedPerDay } = processEventsForWeek(
      [ev('s', '2026-06-15T09:00:00Z', '2026-06-15T09:30:00Z')],
      MONDAY,
      TZ,
      7
    )
    const day0 = timedPerDay.get(MONDAY)!
    expect(day0).toHaveLength(1)
    expect(day0[0].top).toBe(minsToTop(540))
    expect(day0[0].height).toBe(minsToTop(30))
  })

  it('renders the final day of a multi-day event ending at midnight as a FULL day', () => {
    // Mon 10:00 → Wed 00:00. The span is Mon–Tue (midnight end belongs to the
    // previous day), and Tuesday is fully covered, so it must render 0→1440.
    const { timedPerDay } = processEventsForWeek(
      [ev('m', '2026-06-15T10:00:00Z', '2026-06-17T00:00:00Z')],
      MONDAY,
      TZ,
      7
    )
    const day0 = timedPerDay.get(MONDAY)!
    const day1 = timedPerDay.get(MONDAY + DAY_MS)!
    const day2 = timedPerDay.get(MONDAY + 2 * DAY_MS)!

    expect(day0).toHaveLength(1)
    expect(day0[0].top).toBe(minsToTop(600))
    expect(day0[0].height).toBe(minsToTop(1440 - 600))

    expect(day1).toHaveLength(1)
    expect(day1[0].top).toBe(0)
    expect(day1[0].height).toBe(minsToTop(1440)) // the bug rendered this as 0

    expect(day2).toHaveLength(0) // event already ended; Wednesday is empty
  })

  it('keeps a partial final day for a multi-day event ending mid-day', () => {
    const { timedPerDay } = processEventsForWeek(
      [ev('m', '2026-06-15T10:00:00Z', '2026-06-16T14:00:00Z')], // Mon 10:00 → Tue 14:00
      MONDAY,
      TZ,
      7
    )
    const day1 = timedPerDay.get(MONDAY + DAY_MS)!
    expect(day1).toHaveLength(1)
    expect(day1[0].top).toBe(0)
    expect(day1[0].height).toBe(minsToTop(840)) // 0 → 14:00
  })
})

describe('getDaySpan', () => {
  it('treats a midnight end as the previous day', () => {
    const span = getDaySpan('2026-06-15T22:00:00Z', '2026-06-16T00:00:00Z', MONDAY, TZ)
    expect(span.startDay).toBe(0)
    expect(span.endDay).toBe(0)
    expect(span.spanDays).toBe(1)
  })

  it('computes the span of a multi-day event', () => {
    const span = getDaySpan('2026-06-15T10:00:00Z', '2026-06-17T14:00:00Z', MONDAY, TZ)
    expect(span.startDay).toBe(0)
    expect(span.endDay).toBe(2)
    expect(span.spanDays).toBe(3)
  })
})

describe('position helpers', () => {
  it('snapMin rounds to the nearest slot', () => {
    expect(snapMin(7)).toBe(0)
    expect(snapMin(8)).toBe(15)
    expect(snapMin(37)).toBe(30)
  })

  it('clampMins clamps to [0, 1440 - slot]', () => {
    expect(clampMins(-5)).toBe(0)
    expect(clampMins(2000)).toBe(1440 - MIN_SLOT_MIN)
    expect(clampMins(600)).toBe(600)
  })

  it('minsToTop and topToMins round-trip', () => {
    expect(topToMins(minsToTop(615))).toBeCloseTo(615)
  })

  it('formatMinutesToTime zero-pads h:mm', () => {
    expect(formatMinutesToTime(90)).toBe('01:30')
    expect(formatMinutesToTime(540)).toBe('09:00')
  })
})
