import { describe, expect, it } from 'vitest'
import type { Task } from '../../types'
import { homeTasks, homeDayLine } from './todayView'

const task = (id: string, over: Partial<Task> = {}): Task => ({
  id, title: id, priority: 3, status: 'TODO', tags: [],
  createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z', ...over,
} as Task)

// Gap list row 8 (Denis 26.09: Home like the bot's «📅 Сегодня», today.go).
describe('homeTasks', () => {
  it('shows open tasks, most urgent first, five and the rest counted', () => {
    const tasks = [
      task('p5', { priority: 5 }), task('p1', { priority: 1 }), task('done', { status: 'DONE' }),
      task('sched', { priority: 2, status: 'SCHEDULED' }), task('p3a'), task('p3b'), task('p4', { priority: 4 }),
      task('buffer', { priority: 0 }),
    ]
    const { shown, more } = homeTasks(tasks)
    expect(shown.map((t) => t.id)).toEqual(['p1', 'sched', 'p3a', 'p3b', 'p4'])
    expect(more).toBe(2)
  })

  it('leaves out a repeating task already answered today', () => {
    const { shown } = homeTasks([task('daily', { rrule: 'FREQ=DAILY', occurrenceState: 'done' }), task('one')])
    expect(shown.map((t) => t.id)).toEqual(['one'])
  })
})

describe('homeDayLine', () => {
  it('names the square and the count, or that the day is not taken', () => {
    expect(homeDayLine(undefined)).toBeNull()
    expect(homeDayLine({ day: '2026-09-26', target: 5, confirmed: false, items: [], done: 0, level: 0, before_start: false })).toEqual({ kind: 'notTaken' })
    expect(homeDayLine({ day: '2026-09-26', target: 5, confirmed: true, items: [], done: 3, level: 3, before_start: false }))
      .toEqual({ kind: 'taken', square: '🟧', done: 3, target: 5 })
  })
})
