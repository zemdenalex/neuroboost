import { describe, it, expect } from 'vitest'
import { hintsForRoute } from './hintsContent'

describe('hintsForRoute', () => {
  it('returns ordered anchors for a workflow route, with derived i18n keys', () => {
    expect(hintsForRoute('/calendar')).toEqual([
      { anchor: 'calendar.newEvent', titleKey: 'hints.calendar.newEvent.title', bodyKey: 'hints.calendar.newEvent.body' },
      { anchor: 'calendar.grid', titleKey: 'hints.calendar.grid.title', bodyKey: 'hints.calendar.grid.body' },
      { anchor: 'calendar.taskSidebar', titleKey: 'hints.calendar.taskSidebar.title', bodyKey: 'hints.calendar.taskSidebar.body' },
    ])
  })
  it('ignores sub-paths and resolves by the first segment', () => {
    expect(hintsForRoute('/tasks/123').map(h => h.anchor)).toEqual(['tasks.new', 'tasks.schedule', 'tasks.complete'])
  })
  it('returns [] for the root and for routes that do not exist', () => {
    expect(hintsForRoute('/')).toEqual([])
    expect(hintsForRoute('')).toEqual([])
    expect(hintsForRoute('/nope')).toEqual([])
    // ⚠ /settings used to be asserted empty here. It was not a decision, it
    // was the gap Denis reported on 16.08 — markers on four pages out of nine
    // — written down as if it were intended. Coverage now belongs to
    // hintsCoverage.test.ts, which measures against the Help list.
  })
  it('derives every anchor key under the hints namespace with .title/.body', () => {
    for (const path of ['/home', '/calendar', '/tasks', '/planning']) {
      for (const h of hintsForRoute(path)) {
        expect(h.titleKey).toBe(`hints.${h.anchor}.title`)
        expect(h.bodyKey).toBe(`hints.${h.anchor}.body`)
      }
    }
  })
})
