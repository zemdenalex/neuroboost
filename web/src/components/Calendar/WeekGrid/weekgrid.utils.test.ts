import { describe, it, expect } from 'vitest'
import {
  processEventsForWeek,
  getDaySpan,
  assignLanes,
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

describe('assignLanes', () => {
  const rng = (startMin: number, endMin: number) => ({ startMin, endMin })

  it('gives a lone event the full width', () => {
    expect(assignLanes([rng(600, 660)])).toEqual([{ leftPct: 0, widthPct: 1 }])
  })

  it('returns [] for no events', () => {
    expect(assignLanes([])).toEqual([])
  })

  it('keeps sequential (non-overlapping) events full width', () => {
    // 10:00–11:00 then 11:00–12:00 — touching is NOT overlapping.
    const lanes = assignLanes([rng(600, 660), rng(660, 720)])
    expect(lanes).toEqual([{ leftPct: 0, widthPct: 1 }, { leftPct: 0, widthPct: 1 }])
  })

  it('splits two overlapping events into halves', () => {
    const lanes = assignLanes([rng(600, 720), rng(660, 780)])
    expect(lanes).toEqual([{ leftPct: 0, widthPct: 0.5 }, { leftPct: 0.5, widthPct: 0.5 }])
  })

  it('splits three mutually overlapping events into thirds', () => {
    const lanes = assignLanes([rng(600, 720), rng(610, 720), rng(620, 720)])
    expect(lanes).toEqual([
      { leftPct: 0, widthPct: 1 / 3 },
      { leftPct: 1 / 3, widthPct: 1 / 3 },
      { leftPct: 2 / 3, widthPct: 1 / 3 },
    ])
  })

  it('reuses a freed column within a transitive cluster (staircase)', () => {
    // A 10–12, B 11–13, C 12–14. A&B overlap, B&C overlap, A&C touch (no overlap).
    // Cluster {A,B,C}; A→col0, B→col1, C reuses col0. 2 columns total.
    const lanes = assignLanes([rng(600, 720), rng(660, 780), rng(720, 840)])
    expect(lanes).toEqual([
      { leftPct: 0, widthPct: 0.5 },
      { leftPct: 0.5, widthPct: 0.5 },
      { leftPct: 0, widthPct: 0.5 },
    ])
  })

  it('is independent of input order', () => {
    const a = assignLanes([rng(660, 780), rng(600, 720)])
    // First input (later start) lands in column 1; second in column 0.
    expect(a[0]).toEqual({ leftPct: 0.5, widthPct: 0.5 })
    expect(a[1]).toEqual({ leftPct: 0, widthPct: 0.5 })
  })
})

describe('processEventsForWeek — overlap lanes', () => {
  it('lays two overlapping single-day events side by side', () => {
    const { timedPerDay } = processEventsForWeek(
      [ev('a', '2026-06-15T10:00:00Z', '2026-06-15T11:00:00Z'),
       ev('b', '2026-06-15T10:30:00Z', '2026-06-15T11:30:00Z')],
      MONDAY, TZ, 7
    )
    const day0 = timedPerDay.get(MONDAY)!
    expect(day0).toHaveLength(2)
    const byId = Object.fromEntries(day0.map((e) => [e.id, e]))
    expect(byId.a.widthPct).toBe(0.5)
    expect(byId.b.widthPct).toBe(0.5)
    expect(new Set([byId.a.leftPct, byId.b.leftPct])).toEqual(new Set([0, 0.5]))
  })

  it('does NOT merge two adjacent sub-slot events (uses real minutes, not clamped height)', () => {
    // Both are 5 min — their rendered HEIGHT is clamped to the 15-min minimum, which
    // would falsely overlap if laning used pixels. Real minutes keep them full-width.
    const { timedPerDay } = processEventsForWeek(
      [ev('x', '2026-06-15T10:00:00Z', '2026-06-15T10:05:00Z'),
       ev('y', '2026-06-15T10:10:00Z', '2026-06-15T10:15:00Z')],
      MONDAY, TZ, 7
    )
    const day0 = timedPerDay.get(MONDAY)!
    expect(day0.every((e) => e.widthPct === 1)).toBe(true)
  })

  it('does not squeeze a single-day event because a multi-day segment shares the day', () => {
    const { timedPerDay } = processEventsForWeek(
      [ev('multi', '2026-06-15T09:00:00Z', '2026-06-17T09:00:00Z'), // Mon→Wed
       ev('single', '2026-06-16T10:00:00Z', '2026-06-16T11:00:00Z')], // Tue, inside multi
      MONDAY, TZ, 7
    )
    const tue = timedPerDay.get(MONDAY + DAY_MS)!
    const single = tue.find((e) => e.id === 'single')!
    expect(single.widthPct).toBe(1) // laned only among single-day events
  })
})
