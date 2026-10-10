import { describe, it, expect } from 'vitest'
import { taskCounts } from './taskCounts'
import type { Task } from '../../types'

// Home dashboard counters (queue: tests for pages without any).
// 🔴 A repeating task's due_date is the day its series started, and its
// status describes the series. Read like a one-off, a daily series from last
// week is «overdue» for ever and «to do» even after today is ticked. The bot's
// statistics (bot/internal/handlers/statsview.go taskTotals) already leave
// series out of open and overdue; the web counted them.
const task = (p: Partial<Task>): Task => ({ id: Math.random().toString(36), title: 't', status: 'TODO', priority: 3, ...p }) as Task
const now = new Date('2026-09-26T12:00:00Z')

describe('taskCounts', () => {
  it('counts one-off tasks as before', () => {
    const c = taskCounts(
      [
        task({ status: 'TODO', dueDate: '2026-09-20T09:00:00Z' }),
        task({ status: 'SCHEDULED' }),
        task({ status: 'DONE', dueDate: '2026-09-20T09:00:00Z' }),
        task({ status: 'CANCELLED' }),
      ],
      now,
    )
    expect(c).toEqual({ todo: 2, done: 1, overdue: 1 })
  })

  it('a running series is never overdue, and to do only while today is unanswered', () => {
    const daily = { rrule: 'FREQ=DAILY', status: 'TODO' as const, dueDate: '2026-09-19T08:00:00Z' }
    expect(taskCounts([task(daily)], now)).toEqual({ todo: 1, done: 0, overdue: 0 })
    expect(taskCounts([task({ ...daily, occurrenceState: 'done' })], now)).toEqual({ todo: 0, done: 0, overdue: 0 })
  })

  it('a switched-off series is not a finished task', () => {
    expect(taskCounts([task({ rrule: 'FREQ=DAILY', status: 'DONE' })], now).done).toBe(0)
  })
})
