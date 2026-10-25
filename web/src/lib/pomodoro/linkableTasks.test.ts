import { describe, it, expect } from 'vitest'
import { linkableTasks } from './linkableTasks'
import type { Calendar } from '../../api/calendars'

// docs/tasks-web-cleanup.md 4.8 (audit T1a): Pomodoro offered every task the
// user can READ, but a finished block logs time through log-time, which writes
// only to calendars the user may change. A block linked to a task in a
// calendar shared read-only failed there and was rolled back entirely.
const cal = (id: string, role: Calendar['role'], status: Calendar['status'] = 'active'): Calendar => ({
  id,
  name: id,
  color: null,
  kind: 'shared',
  role,
  status,
  created_at: '',
})

describe('linkableTasks', () => {
  const calendars = [cal('mine', 'owner'), cal('team', 'editor'), cal('boss', 'viewer'), cal('new', 'editor', 'invited')]
  const task = (id: string, calendar_id?: string, status = 'TODO') => ({ id, calendar_id, status })

  it('keeps open tasks in calendars the user may change', () => {
    const got = linkableTasks([task('a', 'mine'), task('b', 'team'), task('c', 'boss'), task('d', 'new')], calendars)
    expect(got.map((t) => t.id)).toEqual(['a', 'b'])
  })
  it('drops finished and cancelled tasks', () => {
    const got = linkableTasks([task('a', 'mine', 'DONE'), task('b', 'mine', 'CANCELLED'), task('c', 'mine', 'SCHEDULED')], calendars)
    expect(got.map((t) => t.id)).toEqual(['c'])
  })
  it('keeps a task with no calendar: the API puts it in the personal one', () => {
    expect(linkableTasks([task('a')], calendars).map((t) => t.id)).toEqual(['a'])
  })
})
