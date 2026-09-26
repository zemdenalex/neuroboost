import { describe, it, expect } from 'vitest'
import { feedbackStats } from './feedbackStats'

// Audit A2 (docs/tasks-web-cleanup.md 4.1): Overview counted whatever the
// Backlog filter had last loaded, so «Total» meant «bugs» after a bug filter.
// The counting is a pure function of the list it is given; the page gives it
// the unfiltered one.
describe('feedbackStats', () => {
  const items = [
    { status: 'open', type: 'bug' },
    { status: 'open', type: 'idea' },
    { status: 'done', type: 'bug' },
  ]
  it('counts every status and type asked for, zero included', () => {
    const s = feedbackStats(items, ['open', 'done', 'wontfix'], ['bug', 'idea', 'other'])
    expect(s.total).toBe(3)
    expect(s.byStatus).toEqual({ open: 2, done: 1, wontfix: 0 })
    expect(s.byType).toEqual({ bug: 2, idea: 1, other: 0 })
  })
})
