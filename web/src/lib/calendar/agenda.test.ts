import { describe, it, expect } from 'vitest'
import { buildAgenda, localDayKey } from './agenda'
import type { NbEvent, Task } from '../../types'

const ev = (id: string, startsAt: string, title = id, allDay = false): NbEvent =>
  ({ id, title, startsAt, endsAt: startsAt, allDay } as NbEvent)

const task = (id: string, dueDate?: string, status = 'TODO'): Task =>
  ({ id, title: id, dueDate, status } as Task)

describe('localDayKey', () => {
  // 🔴 The day is the day where the USER is. 21:00 in Moscow is still today,
  // and an ISO slice would file it under tomorrow — the same mistake made
  // reading a due_date through a UTC session on 17.09.
  it('files an evening instant under the local day, not the UTC one', () => {
    const at = new Date('2026-09-18T21:30:00Z') // 00:30 on the 19th in Moscow
    expect(localDayKey(at, 'Europe/Moscow')).toBe('2026-09-19')
    expect(localDayKey(at, 'UTC')).toBe('2026-09-18')
  })
})

describe('buildAgenda', () => {
  const from = new Date('2026-09-18T00:00:00Z')

  it('groups by day and keeps days in order', () => {
    const agenda = buildAgenda(
      [ev('b', '2026-09-19T09:00:00Z'), ev('a', '2026-09-18T09:00:00Z')],
      [],
      'UTC',
      from,
      7,
    )
    expect(agenda.map((d) => d.key)).toEqual(['2026-09-18', '2026-09-19'])
  })

  it('omits empty days entirely', () => {
    const agenda = buildAgenda([ev('a', '2026-09-20T09:00:00Z')], [], 'UTC', from, 7)
    expect(agenda).toHaveLength(1)
    expect(agenda[0].key).toBe('2026-09-20')
  })

  it('sorts within a day, all-day first then by time', () => {
    const agenda = buildAgenda(
      [
        ev('late', '2026-09-18T18:00:00Z'),
        ev('early', '2026-09-18T09:00:00Z'),
        ev('whole', '2026-09-18T00:00:00Z', 'whole', true),
      ],
      [],
      'UTC',
      from,
      7,
    )
    expect(agenda[0].items.map((i) => i.id)).toEqual(['whole', 'early', 'late'])
  })

  it('includes dated tasks and drops undated ones', () => {
    const agenda = buildAgenda([], [task('dated', '2026-09-18T00:00:00Z'), task('someday')], 'UTC', from, 7)
    expect(agenda).toHaveLength(1)
    expect(agenda[0].items.map((i) => i.id)).toEqual(['dated'])
    expect(agenda[0].items[0].kind).toBe('task')
  })

  // 🔴 The window is a window. Without this the agenda quietly becomes
  // "everything ever", which on an account with a year of history is a list
  // nobody can reach the bottom of.
  it('ignores anything outside the window', () => {
    const agenda = buildAgenda(
      [ev('past', '2026-09-01T09:00:00Z'), ev('far', '2026-12-01T09:00:00Z')],
      [],
      'UTC',
      from,
      7,
    )
    expect(agenda).toHaveLength(0)
  })

  it('marks a finished task as done rather than hiding it', () => {
    const agenda = buildAgenda([], [task('x', '2026-09-18T00:00:00Z', 'DONE')], 'UTC', from, 7)
    expect(agenda[0].items[0].done).toBe(true)
  })

  it('survives a malformed date instead of throwing', () => {
    const agenda = buildAgenda([ev('bad', 'not-a-date')], [task('t', 'also-bad')], 'UTC', from, 7)
    expect(agenda).toHaveLength(0)
  })
})
