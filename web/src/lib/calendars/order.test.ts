import { describe, it, expect } from 'vitest'
import { sortCalendars } from './order'
import type { Calendar } from '../../api/calendars'

const make = (over: Partial<Calendar>): Calendar => ({
  id: 'x',
  name: 'x',
  color: null,
  kind: 'shared',
  role: 'owner',
  status: 'active',
  created_at: '2026-08-11T00:00:00Z',
  ...over,
})

describe('sortCalendars', () => {
  it('puts the personal calendar first regardless of creation order', () => {
    const list = [
      make({ id: 'a', kind: 'shared', created_at: '2026-01-01T00:00:00Z' }),
      make({ id: 'p', kind: 'personal', created_at: '2026-08-01T00:00:00Z' }),
    ]
    expect(sortCalendars(list).map((c) => c.id)).toEqual(['p', 'a'])
  })

  it('orders the rest oldest first', () => {
    const list = [
      make({ id: 'new', created_at: '2026-08-10T00:00:00Z' }),
      make({ id: 'old', created_at: '2026-02-10T00:00:00Z' }),
    ]
    expect(sortCalendars(list).map((c) => c.id)).toEqual(['old', 'new'])
  })

  it('does not mutate its input', () => {
    const list = [make({ id: 'a' }), make({ id: 'p', kind: 'personal' })]
    sortCalendars(list)
    expect(list.map((c) => c.id)).toEqual(['a', 'p'])
  })
})
