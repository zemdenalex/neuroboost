import type { NbEvent } from '../../types'
import { localTimeOn } from './shiftDays'

/**
 * How full a day is, for the heatmap month (spec V003-20260924-arc-web-month-view, R4).
 *
 * Minutes covered by timed events inside the user's local day, overlaps
 * counted once, over 16 waking hours, capped at 1. All-day events are not
 * time spent, so they do not count.
 */

export const AWAKE_MINUTES = 960

const MINUTE_MS = 60 * 1000

function nextDay(day: string): string {
  return new Date(Date.parse(day + 'T00:00:00Z') + 24 * 60 * MINUTE_MS).toISOString().slice(0, 10)
}

export function busyShare(events: NbEvent[], day: string, timeZone: string): number {
  const dayStart = Date.parse(localTimeOn(day, '00:00', timeZone))
  const dayEnd = Date.parse(localTimeOn(nextDay(day), '00:00', timeZone))

  const spans: Array<[number, number]> = []
  for (const e of events) {
    if (e.allDay) continue
    const s = Math.max(Date.parse(e.startsAt), dayStart)
    const t = Math.min(Date.parse(e.endsAt), dayEnd)
    if (Number.isNaN(s) || Number.isNaN(t) || t <= s) continue
    spans.push([s, t])
  }
  spans.sort((a, b) => a[0] - b[0])

  let covered = 0
  let curStart = -Infinity
  let curEnd = -Infinity
  for (const [s, t] of spans) {
    if (s > curEnd) {
      if (curEnd > curStart) covered += curEnd - curStart
      curStart = s
      curEnd = t
    } else if (t > curEnd) {
      curEnd = t
    }
  }
  if (curEnd > curStart) covered += curEnd - curStart

  return Math.min(1, covered / MINUTE_MS / AWAKE_MINUTES)
}
