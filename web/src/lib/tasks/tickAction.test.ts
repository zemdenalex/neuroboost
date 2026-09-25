import { describe, it, expect } from 'vitest'
import { tickAction, tickedToday } from './tickAction'

// docs/tasks-web-cleanup.md 4.11: the Tasks page ticked a repeating task by
// writing status = DONE, which switches the whole series off. A tick on a
// running series answers today's day of it, as the bot and Kanban do.
describe('tickAction', () => {
  it('closes and reopens a one-off task by status', () => {
    expect(tickAction({ status: 'TODO' })).toEqual({ kind: 'status', next: 'DONE' })
    expect(tickAction({ status: 'DONE' })).toEqual({ kind: 'status', next: 'TODO' })
  })
  it('answers today of a running series, never its status', () => {
    expect(tickAction({ status: 'TODO', rrule: 'FREQ=DAILY' })).toEqual({ kind: 'occurrence', state: 'done' })
    expect(tickAction({ status: 'SCHEDULED', rrule: 'FREQ=DAILY', occurrence_state: 'skipped' })).toEqual({
      kind: 'occurrence',
      state: 'done',
    })
  })
  it('a second tick on a day already done takes the answer back', () => {
    expect(tickAction({ status: 'TODO', rrule: 'FREQ=DAILY', occurrence_state: 'done' })).toEqual({
      kind: 'occurrence',
      state: 'open',
    })
  })
  it('a switched-off series is switched back on by status', () => {
    expect(tickAction({ status: 'DONE', rrule: 'FREQ=DAILY' })).toEqual({ kind: 'status', next: 'TODO' })
  })
})

describe('tickedToday', () => {
  it('is the status for a one-off and today for a series', () => {
    expect(tickedToday({ status: 'DONE' })).toBe(true)
    expect(tickedToday({ status: 'TODO' })).toBe(false)
    expect(tickedToday({ status: 'TODO', rrule: 'FREQ=DAILY', occurrence_state: 'done' })).toBe(true)
    expect(tickedToday({ status: 'TODO', rrule: 'FREQ=DAILY', occurrence_state: 'skipped' })).toBe(false)
    expect(tickedToday({ status: 'DONE', rrule: 'FREQ=DAILY' })).toBe(true)
  })
})
