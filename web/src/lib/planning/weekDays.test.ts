import { describe, it, expect } from 'vitest'
import { weekDays } from './weekDays'

// The Planning page's seven day columns (queue: tests for pages without any).
// Local dates throughout: the keys and the Monday are the browser's, as on the page.
const ev = (id: string, start: Date, minutes: number, all_day = false) => ({
  id,
  title: id,
  starts_at: start.toISOString(),
  ends_at: new Date(start.getTime() + minutes * 60000).toISOString(),
  all_day,
})

describe('weekDays', () => {
  const monday = new Date(2026, 8, 21)

  it('is seven days from the Monday', () => {
    const days = weekDays([], monday)
    expect(days).toHaveLength(7)
    expect(days[0].date.getDate()).toBe(21)
    expect(days[6].date.getDate()).toBe(27)
  })

  it('puts each event on its day, in time order, whatever order they came in', () => {
    const days = weekDays(
      [ev('late', new Date(2026, 8, 23, 18), 60), ev('early', new Date(2026, 8, 23, 8), 30), ev('fri', new Date(2026, 8, 25, 10), 60)],
      monday,
    )
    expect(days[2].events.map((e) => e.id)).toEqual(['early', 'late'])
    expect(days[4].events.map((e) => e.id)).toEqual(['fri'])
    expect(days[0].events).toEqual([])
  })

  it('counts hours of timed events only', () => {
    const days = weekDays([ev('a', new Date(2026, 8, 22, 9), 90), ev('off', new Date(2026, 8, 22), 1440, true)], monday)
    expect(days[1].scheduledHours).toBe(1.5)
    expect(days[1].events).toHaveLength(2)
  })
})
