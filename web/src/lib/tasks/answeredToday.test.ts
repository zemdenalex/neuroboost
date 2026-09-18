import { describe, it, expect } from 'vitest'
import { answeredToday } from '../../types'
import type { Task } from '../../types'

const task = (over: Partial<Task>): Task =>
  ({ id: 't', title: 't', priority: 3, status: 'TODO', tags: [], createdAt: '', updatedAt: '', ...over } as Task)

describe('answeredToday', () => {
  // 🔴 Denis, 18.09: «if completed for the day it should stop being in the task
  // list, but appear somewhere gray, and on the next day it pops up again».
  it('is true for a repeating task dealt with today', () => {
    expect(answeredToday(task({ rrule: 'FREQ=DAILY', occurrenceState: 'done' }))).toBe(true)
    expect(answeredToday(task({ rrule: 'FREQ=DAILY', occurrenceState: 'skipped' }))).toBe(true)
  })

  it('is false for a repeating task nobody has touched today', () => {
    expect(answeredToday(task({ rrule: 'FREQ=DAILY' }))).toBe(false)
  })

  // 🔴 The control: a one-off task is never "answered today", whatever stray
  // state arrives. Without the rrule check an ordinary task would vanish from
  // the list on a stale field.
  it('is false for a one-off task even with a stray state', () => {
    expect(answeredToday(task({ occurrenceState: 'done' }))).toBe(false)
    expect(answeredToday(task({}))).toBe(false)
  })
})
