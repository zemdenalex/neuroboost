import { describe, it, expect, beforeEach } from 'vitest'
import type { SessionRecord } from './types'
import { localDayKey, summarizeToday, loadHistory, HISTORY_KEY } from './history'

function rec(partial: Partial<SessionRecord>): SessionRecord {
  return { date: '', mode: 'work', durationSeconds: 1500, completedAt: '', ...partial }
}

describe('localDayKey', () => {
  it('formats local Y-M-D, zero-padded', () => {
    expect(localDayKey(new Date(2026, 0, 5, 9, 0, 0))).toBe('2026-01-05')
  })
})

describe('summarizeToday', () => {
  const now = new Date(2026, 5, 16, 12, 0, 0) // local noon, Jun 16
  const y = now.getFullYear()
  const m = now.getMonth()
  const d = now.getDate()
  // A timestamp at a given local hour, offset by whole local days.
  const at = (hour: number, dayDelta = 0) => new Date(y, m, d + dayDelta, hour, 0, 0).toISOString()

  it('counts work blocks completed on the local day and sums minutes', () => {
    const history = [
      rec({ completedAt: at(9), durationSeconds: 1500 }),
      rec({ completedAt: at(14), durationSeconds: 1500 }),
    ]
    expect(summarizeToday(history, now)).toEqual({ count: 2, minutes: 50 })
  })

  it('groups by LOCAL day — an early-morning block stays "today" even when it is the previous UTC day', () => {
    // 01:00 local today: in any positive-offset zone this is the previous UTC date.
    // A UTC-based filter would wrongly drop it; local grouping keeps it.
    expect(summarizeToday([rec({ completedAt: at(1) })], now).count).toBe(1)
  })

  it('excludes other local days', () => {
    const history = [rec({ completedAt: at(12, -1) }), rec({ completedAt: at(12, +1) })]
    expect(summarizeToday(history, now)).toEqual({ count: 0, minutes: 0 })
  })

  it('ignores break sessions', () => {
    const history = [
      rec({ completedAt: at(10), mode: 'shortBreak' }),
      rec({ completedAt: at(11), mode: 'longBreak' }),
      rec({ completedAt: at(12), mode: 'work', durationSeconds: 1500 }),
    ]
    expect(summarizeToday(history, now)).toEqual({ count: 1, minutes: 25 })
  })

  it('skips records with missing or invalid completedAt', () => {
    const history = [
      rec({ completedAt: '' }),
      rec({ completedAt: 'not-a-date' }),
      rec({ completedAt: at(12), durationSeconds: 1500 }),
    ]
    expect(summarizeToday(history, now)).toEqual({ count: 1, minutes: 25 })
  })

  it('handles empty history', () => {
    expect(summarizeToday([], now)).toEqual({ count: 0, minutes: 0 })
  })
})

describe('loadHistory', () => {
  beforeEach(() => localStorage.clear())

  it('returns [] when nothing is stored', () => {
    expect(loadHistory()).toEqual([])
  })

  it('parses stored records', () => {
    localStorage.setItem(HISTORY_KEY, JSON.stringify([rec({ completedAt: '2026-06-16T09:00:00.000Z' })]))
    expect(loadHistory()).toHaveLength(1)
  })

  it('returns [] on malformed JSON', () => {
    localStorage.setItem(HISTORY_KEY, '{bad json')
    expect(loadHistory()).toEqual([])
  })

  it('returns [] when stored value is not an array', () => {
    localStorage.setItem(HISTORY_KEY, JSON.stringify({ nope: true }))
    expect(loadHistory()).toEqual([])
  })
})
