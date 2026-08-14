import { describe, it, expect } from 'vitest'
import { resolveEventColor, type CalendarColors } from './eventColor'

const COLORS: CalendarColors = {
  'cal-work': '#7c3aed',
  'cal-home': '#16a34a',
  'cal-nocolor': null,
}

describe('resolveEventColor', () => {
  it('falls back to the calendar colour when the event has none', () => {
    expect(resolveEventColor({ calendarId: 'cal-work' }, COLORS)).toBe('#7c3aed')
  })

  // 🔴 The precedence, asserted as precedence rather than as two separate
  // cases: a colour set on one event is a deliberate mark on that event, and
  // the calendar must not erase it.
  it("lets the event's own colour win over the calendar's", () => {
    const event = { color: '#ef4444', calendarId: 'cal-work' }
    expect(resolveEventColor(event, COLORS)).toBe('#ef4444')
    expect(resolveEventColor(event, COLORS)).not.toBe(COLORS['cal-work'])
  })

  it('returns undefined when neither has a colour, so the grid keeps its default', () => {
    expect(resolveEventColor({ calendarId: 'cal-nocolor' }, COLORS)).toBeUndefined()
    expect(resolveEventColor({}, COLORS)).toBeUndefined()
  })

  it('treats an empty or blank colour as absent', () => {
    // The editor stores '' when the field is left blank, and '' passed to
    // backgroundColor paints nothing while still counting as set.
    expect(resolveEventColor({ color: '', calendarId: 'cal-work' }, COLORS)).toBe('#7c3aed')
    expect(resolveEventColor({ color: '   ', calendarId: 'cal-work' }, COLORS)).toBe('#7c3aed')
    expect(resolveEventColor({ color: '' }, COLORS)).toBeUndefined()
  })

  it('survives an event pointing at a calendar that is not in the map', () => {
    // Happens between the events arriving and the calendar list arriving, and
    // for an event in a calendar the user has since been removed from.
    expect(resolveEventColor({ calendarId: 'cal-gone' }, COLORS)).toBeUndefined()
    expect(resolveEventColor({ color: '#fff', calendarId: 'cal-gone' }, COLORS)).toBe('#fff')
  })

  it('survives an empty colour map', () => {
    expect(resolveEventColor({ calendarId: 'cal-work' }, {})).toBeUndefined()
  })

  // The negative control: a function that always returned the event colour, or
  // always the calendar's, would each pass some of the above.
  it('reads both sources', () => {
    expect(resolveEventColor({ color: '#111', calendarId: 'cal-home' }, COLORS)).toBe('#111')
    expect(resolveEventColor({ calendarId: 'cal-home' }, COLORS)).toBe('#16a34a')
  })
})
