import { describe, expect, it } from 'vitest'
import type { Task } from '../../types'
import { sidebarTick, sidebarTicked } from './sidebarTick'

const task = (over: Partial<Task>): Task => ({
  id: 't1', title: 'проветрить', priority: 3, status: 'TODO', tags: [],
  createdAt: '2026-09-01T00:00:00Z', updatedAt: '2026-09-01T00:00:00Z', ...over,
} as Task)

// Gap-list F1 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): the
// calendar's task panel (MobileTaskPanel on a phone) sent status DONE for every
// tick, and DONE on a series switches it off for good.
describe('sidebarTick', () => {
  it('answers today for a running series instead of ending it', () => {
    expect(sidebarTick(task({ rrule: 'FREQ=DAILY' }))).toEqual({ kind: 'occurrence', state: 'done' })
  })

  it('takes today\'s answer back on a second tick', () => {
    expect(sidebarTick(task({ rrule: 'FREQ=DAILY', occurrenceState: 'done' }))).toEqual({ kind: 'occurrence', state: 'open' })
  })

  it('still toggles the status of a one-off task', () => {
    expect(sidebarTick(task({}))).toEqual({ kind: 'status', next: 'DONE' })
    expect(sidebarTick(task({ status: 'DONE' }))).toEqual({ kind: 'status', next: 'TODO' })
  })
})

describe('sidebarTicked', () => {
  it('shows a series ticked only when today is done', () => {
    expect(sidebarTicked(task({ rrule: 'FREQ=DAILY' }))).toBe(false)
    expect(sidebarTicked(task({ rrule: 'FREQ=DAILY', occurrenceState: 'done' }))).toBe(true)
    expect(sidebarTicked(task({ status: 'DONE' }))).toBe(true)
  })
})
