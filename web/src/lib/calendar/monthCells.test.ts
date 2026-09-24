import { describe, it, expect } from 'vitest'
import type { NbEvent } from '../../types'
import { eventsByDay, cellRows } from './monthCells'

const TZ = 'Europe/Moscow' // UTC+3, no DST
const days = ['2026-09-23', '2026-09-24', '2026-09-25', '2026-09-26']

function ev(id: string, startsAt: string, endsAt: string, extra: Partial<NbEvent> = {}): NbEvent {
  return { id, title: id, startsAt, endsAt, ...extra }
}

describe('eventsByDay', () => {
  it('files an event under the day in the user zone, not the UTC day', () => {
    // 22:30 UTC on the 23rd is 01:30 on the 24th in Moscow.
    const got = eventsByDay([ev('late', '2026-09-23T22:30:00Z', '2026-09-23T23:30:00Z')], days, TZ)
    expect(got['2026-09-23'].map((e) => e.event.id)).toEqual([])
    expect(got['2026-09-24'].map((e) => e.event.id)).toEqual(['late'])
  })

  it('draws a multi-day event on every one of its days, the time only on the first', () => {
    const got = eventsByDay([ev('trip', '2026-09-24T07:00:00Z', '2026-09-26T09:00:00Z')], days, TZ)
    expect(got['2026-09-24'][0]).toMatchObject({ first: true })
    expect(got['2026-09-25'][0]).toMatchObject({ first: false })
    expect(got['2026-09-26'][0]).toMatchObject({ first: false })
    expect(got['2026-09-23']).toEqual([])
  })

  it('does not spill onto the next day when an event ends exactly at midnight', () => {
    // 21:00Z = 00:00 Moscow on the 25th.
    const got = eventsByDay([ev('evening', '2026-09-24T17:00:00Z', '2026-09-24T21:00:00Z')], days, TZ)
    expect(got['2026-09-24']).toHaveLength(1)
    expect(got['2026-09-25']).toEqual([])
  })

  it('puts all-day events first, then timed ones by start', () => {
    const got = eventsByDay(
      [
        ev('lunch', '2026-09-24T10:00:00Z', '2026-09-24T11:00:00Z'),
        ev('standup', '2026-09-24T06:00:00Z', '2026-09-24T06:15:00Z'),
        // Starts at the same local midnight as the holiday and comes first in
        // the input: only the all-day rule puts the holiday ahead of it.
        ev('night', '2026-09-23T21:00:00Z', '2026-09-23T22:00:00Z'),
        ev('holiday', '2026-09-23T21:00:00Z', '2026-09-24T21:00:00Z', { allDay: true }),
      ],
      days,
      TZ,
    )
    expect(got['2026-09-24'].map((e) => e.event.id)).toEqual(['holiday', 'night', 'standup', 'lunch'])
  })

  it('gives every grid day a list, empty or not, and ignores events outside the grid', () => {
    const got = eventsByDay([ev('far', '2026-12-01T10:00:00Z', '2026-12-01T11:00:00Z')], days, TZ)
    expect(Object.keys(got).sort()).toEqual(days)
    expect(Object.values(got).every((l) => l.length === 0)).toBe(true)
  })

  it('skips an event with an unreadable start instead of throwing', () => {
    const got = eventsByDay([ev('bad', 'nope', 'nope')], days, TZ)
    expect(got['2026-09-24']).toEqual([])
  })
})

describe('cellRows', () => {
  it('shows up to max and counts the rest', () => {
    expect(cellRows([1, 2, 3, 4, 5], 3)).toEqual({ shown: [1, 2, 3], more: 2 })
  })

  it('does not spend a row on "+1 more": the last item is shown instead', () => {
    expect(cellRows([1, 2, 3, 4], 3)).toEqual({ shown: [1, 2, 3, 4], more: 0 })
  })

  it('handles fewer items than rows', () => {
    expect(cellRows([1], 3)).toEqual({ shown: [1], more: 0 })
  })
})

describe('eventsByDay across a DST change', () => {
  it('files 23:30 on the 25-hour day under that day (Berlin, 25 October 2026)', () => {
    // 23:30 CET = 22:30Z. Counting the day as 24 h from 00:00 CEST (22:00Z on the 24th)
    // would push it to the 26th.
    const berlinDays = ['2026-10-25', '2026-10-26']
    const got = eventsByDay([ev('late', '2026-10-25T22:30:00Z', '2026-10-25T22:45:00Z')], berlinDays, 'Europe/Berlin')
    expect(got['2026-10-25'].map((e) => e.event.id)).toEqual(['late'])
    expect(got['2026-10-26']).toEqual([])
  })
})
