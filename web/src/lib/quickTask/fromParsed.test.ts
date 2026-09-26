import { describe, it, expect } from 'vitest'
import { taskFromParsed, eventFromParsed, describeParsedWhen } from './fromParsed'
import type { ParsedLine } from '../../api/parse'
import type { CreateTaskRequest } from '../../api/tasks'

const base: ParsedLine = {
  kind: 'task', title: '', timezone: 'Europe/Moscow', tags: [], rrule: null,
  priority: null, due_date: null, estimated_minutes: null,
  starts_at: null, ends_at: null, all_day: false, color: null, calendar_id: null,
  calendar_name: null, reminder_offsets: null, is_task: false, uncertain: [],
}

const built: CreateTaskRequest = {
  title: 'купить молоко завтра !1 30м #дом', status: 'TODO', priority: 3,
  estimated_minutes: 15, tags: ['фильтр'], contexts: [], parent_id: 'p1',
}

describe('taskFromParsed', () => {
  it('takes what the line said over the defaults, and keeps the rest', () => {
    const req = taskFromParsed(built, {
      ...base, title: 'купить молоко', priority: 1, estimated_minutes: 30,
      due_date: '2026-09-26T21:00:00Z', tags: ['дом'], rrule: 'FREQ=WEEKLY',
    })
    expect(req).toEqual({
      title: 'купить молоко', status: 'TODO', priority: 1, estimated_minutes: 30,
      due_date: '2026-09-26T21:00:00Z', tags: ['фильтр', 'дом'], contexts: [],
      parent_id: 'p1', rrule: 'FREQ=WEEKLY',
    })
  })

  it('an unstated field keeps the default, and priority 0 is stated', () => {
    const req = taskFromParsed(built, { ...base, title: 'буфер', priority: 0 })
    expect(req.priority).toBe(0)
    expect(req.estimated_minutes).toBe(15)
    expect(req.due_date).toBeUndefined()
    expect(req.rrule).toBeUndefined()
  })
})

describe('eventFromParsed', () => {
  const event: ParsedLine = {
    ...base, kind: 'event', title: 'стоматолог',
    starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z',
  }

  it('sends only what was stated: no reminder field means the user preset', () => {
    expect(eventFromParsed(event)).toEqual({
      title: 'стоматолог', starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z',
      all_day: false, tags: [],
    })
  })

  it('an explicit empty reminder list is kept (silent), and the task link is set', () => {
    const req = eventFromParsed({ ...event, reminder_offsets: [], calendar_id: 'c1', color: 'blue', rrule: 'FREQ=DAILY' }, 't1')
    expect(req.reminder_offsets).toEqual([])
    expect(req.calendar_id).toBe('c1')
    expect(req.color).toBe('blue')
    expect(req.rrule).toBe('FREQ=DAILY')
    expect(req.task_id).toBe('t1')
  })
})

describe('describeParsedWhen', () => {
  it('prints wall time in the zone the line was read in, not the browser zone', () => {
    const text = describeParsedWhen({
      ...base, kind: 'event', starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z',
    }, 'ru')
    expect(text).toContain('15:00')
    expect(text).toContain('16:00')
  })

  it('an all-day event has no clock', () => {
    const text = describeParsedWhen({
      ...base, kind: 'event', all_day: true, starts_at: '2026-09-26T21:00:00Z', ends_at: '2026-09-27T21:00:00Z',
    }, 'ru')
    expect(text).not.toMatch(/\d\d:\d\d/)
    expect(text).toContain('27')
  })
})
