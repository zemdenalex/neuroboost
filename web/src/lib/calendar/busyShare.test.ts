import { describe, it, expect } from 'vitest'
import type { NbEvent } from '../../types'
import { busyShare, AWAKE_MINUTES } from './busyShare'

const TZ = 'Europe/Moscow' // UTC+3

function ev(startsAt: string, endsAt: string, allDay = false): NbEvent {
  return { id: startsAt, title: 't', startsAt, endsAt, allDay }
}

describe('busyShare', () => {
  it('is 0 for an empty day', () => {
    expect(busyShare([], '2026-09-24', TZ)).toBe(0)
  })

  it('is timed minutes over 16 hours', () => {
    // 10:00–14:00 Moscow = 240 minutes.
    expect(AWAKE_MINUTES).toBe(960)
    expect(busyShare([ev('2026-09-24T07:00:00Z', '2026-09-24T11:00:00Z')], '2026-09-24', TZ)).toBe(0.25)
  })

  it('does not count all-day events', () => {
    expect(busyShare([ev('2026-09-23T21:00:00Z', '2026-09-24T21:00:00Z', true)], '2026-09-24', TZ)).toBe(0)
  })

  it('counts only the part of an event inside the day', () => {
    // 22:00 on the 24th to 02:00 on the 25th, Moscow: 120 minutes on each day.
    const late = [ev('2026-09-24T19:00:00Z', '2026-09-24T23:00:00Z')]
    expect(busyShare(late, '2026-09-24', TZ)).toBe(120 / 960)
    expect(busyShare(late, '2026-09-25', TZ)).toBe(120 / 960)
  })

  it('does not count overlapping time twice', () => {
    const overlap = [
      ev('2026-09-24T07:00:00Z', '2026-09-24T09:00:00Z'),
      ev('2026-09-24T08:00:00Z', '2026-09-24T10:00:00Z'),
    ]
    expect(busyShare(overlap, '2026-09-24', TZ)).toBe(180 / 960)
  })

  it('stops at 1', () => {
    expect(busyShare([ev('2026-09-23T21:00:00Z', '2026-09-24T20:00:00Z')], '2026-09-24', TZ)).toBe(1)
  })
})
