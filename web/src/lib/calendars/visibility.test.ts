import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  loadHiddenCalendars,
  saveHiddenCalendars,
  toggleHidden,
  visibleEvents,
} from './visibility'

const KEY = 'nb-hidden-calendars'

describe('loadHiddenCalendars', () => {
  beforeEach(() => localStorage.clear())

  it('is empty when nothing was stored', () => {
    expect(loadHiddenCalendars()).toEqual(new Set())
  })

  it('reads back what was saved', () => {
    saveHiddenCalendars(new Set(['a', 'b']))
    expect(loadHiddenCalendars()).toEqual(new Set(['a', 'b']))
  })

  // The blob is user-writable through devtools and survives deploys, so a
  // shape change must not take the calendar page down with it.
  it('falls back to nothing hidden on a malformed blob', () => {
    localStorage.setItem(KEY, 'not json')
    expect(loadHiddenCalendars()).toEqual(new Set())
    localStorage.setItem(KEY, '{"a":1}')
    expect(loadHiddenCalendars()).toEqual(new Set())
  })

  it('drops non-string entries rather than storing them as ids', () => {
    localStorage.setItem(KEY, JSON.stringify(['a', 3, null, 'b']))
    expect(loadHiddenCalendars()).toEqual(new Set(['a', 'b']))
  })

  it('survives localStorage throwing', () => {
    const spy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('SecurityError')
    })
    expect(loadHiddenCalendars()).toEqual(new Set())
    spy.mockRestore()
  })
})

describe('saveHiddenCalendars', () => {
  afterEach(() => vi.restoreAllMocks())

  it('does not throw when the quota is exhausted', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    expect(() => saveHiddenCalendars(new Set(['a']))).not.toThrow()
  })
})

describe('toggleHidden', () => {
  it('hides one that was shown and shows one that was hidden', () => {
    expect(toggleHidden(new Set(), 'a')).toEqual(new Set(['a']))
    expect(toggleHidden(new Set(['a']), 'a')).toEqual(new Set())
  })

  it('returns a new set so React sees the change', () => {
    const before = new Set(['a'])
    const after = toggleHidden(before, 'b')
    expect(after).not.toBe(before)
    expect(before).toEqual(new Set(['a']))
  })
})

describe('visibleEvents', () => {
  const events = [
    { id: '1', calendarId: 'work' },
    { id: '2', calendarId: 'home' },
    { id: '3' },
  ]

  it('returns everything when nothing is hidden', () => {
    expect(visibleEvents(events, new Set())).toBe(events)
  })

  it('drops events belonging to a hidden calendar', () => {
    expect(visibleEvents(events, new Set(['work'])).map(e => e.id)).toEqual(['2', '3'])
  })

  // 🔴 The failure this guards: events created before calendars existed carry
  // no calendarId, and so does anything whose calendar the client has not
  // loaded. Treating "no calendar" as a hideable calendar would make those
  // vanish with no checkbox that brings them back.
  it('always shows an event with no calendar', () => {
    expect(visibleEvents(events, new Set(['work', 'home'])).map(e => e.id)).toEqual(['3'])
  })

  it('ignores a hidden id no event uses', () => {
    expect(visibleEvents(events, new Set(['gone'])).map(e => e.id)).toEqual(['1', '2', '3'])
  })
})
