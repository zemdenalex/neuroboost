import { describe, it, expect } from 'vitest'
import { defaultTaskCalendarId, writableCalendars } from './defaultTaskCalendar'
import type { Calendar } from '../../api/calendars'

function cal(over: Partial<Calendar> & { id: string }): Calendar {
  return {
    name: over.id,
    color: null,
    kind: 'shared',
    role: 'editor',
    status: 'active',
    created_at: '2026-08-01T00:00:00Z',
    ...over,
  }
}

const personal = cal({ id: 'p1', kind: 'personal', role: 'owner' })
const shared = cal({ id: 's1' })
const readOnly = cal({ id: 's2', role: 'viewer' })
const invited = cal({ id: 's3', status: 'invited' })

describe('writableCalendars', () => {
  it('drops the ones a task cannot be put in', () => {
    // A viewer gets a 403 from the API, and an unaccepted invitation is not
    // a calendar you have yet.
    expect(writableCalendars([personal, shared, readOnly, invited]).map(c => c.id))
      .toEqual(['p1', 's1'])
  })
})

describe('defaultTaskCalendarId', () => {
  it('prefers what the user last chose', () => {
    expect(defaultTaskCalendarId([personal, shared], 's1')).toBe('s1')
  })

  it('ignores a remembered calendar that is gone or now read-only', () => {
    // Left the calendar, or demoted to viewer. Falling through to personal is
    // the safe half of the choice; keeping the id would produce a 403 at save.
    expect(defaultTaskCalendarId([personal, readOnly], 's2')).toBe('p1')
    expect(defaultTaskCalendarId([personal], 'deleted-id')).toBe('p1')
  })

  it('falls back to the personal calendar, never to whatever sorted first', () => {
    // 🔴 The regression that matters. A shared calendar listed ahead of the
    // personal one must not become the default — a task landing quietly in the
    // wrong calendar is found a week later.
    expect(defaultTaskCalendarId([shared, personal], null)).toBe('p1')
  })

  it('takes the first writable one only when there is no personal calendar', () => {
    expect(defaultTaskCalendarId([shared], null)).toBe('s1')
  })

  it('returns empty when nothing is writable, letting the API decide', () => {
    expect(defaultTaskCalendarId([readOnly, invited], null)).toBe('')
    expect(defaultTaskCalendarId([], null)).toBe('')
  })
})
