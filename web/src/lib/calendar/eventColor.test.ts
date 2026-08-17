import { describe, it, expect } from 'vitest'
import { resolveEventColor, type CalendarColors } from './eventColor'
import { PALETTE } from './palette'

const COLORS: CalendarColors = {
  'cal-work': '#7c3aed',
  'cal-home': '#16a34a',
  'cal-nocolor': null,
  // Stored forms that are NOT valid CSS on their own — a calendar colour comes
  // from the same picker as an event's and may be any of these.
  'cal-tailwind': 'green-400',
  'cal-named': 'pink',
  // Unresolvable in EVERY environment: the Tailwind pattern wants `hue-NNN`
  // and the keyword check wants letters only. A bare word like `notacolour`
  // would NOT do — jsdom here has no CSS.supports, so resolveColor's documented
  // fallback accepts bare words and the assertion would pass or fail by
  // environment rather than by behaviour.
  'cal-junk': 'bg-blue-500',
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

/**
 * 🔴 Every value in the block above is a hex string — and a hex string is valid
 * CSS whatever this function does to it. So none of those cases could fail on
 * the defect Denis reported: the editor accepted `blue-400`, the preview showed
 * it (the preview calls resolveColor), and the block on the grid stayed grey,
 * because the raw string went into `backgroundColor` and the browser dropped it.
 *
 * The rule the old tests broke: a control built only from values that work
 * anyway is not a control. These cases use the forms that only survive if the
 * value is actually resolved.
 */
describe('resolveEventColor resolves what it returns', () => {
  it('turns a Tailwind class name into a colour a browser will paint', () => {
    expect(resolveEventColor({ color: 'blue-400' }, COLORS)).toBe('#60a5fa')
  })

  it('turns a palette name into its hex', () => {
    expect(resolveEventColor({ color: 'violet' }, COLORS)).toBe(PALETTE.violet)
  })

  it('resolves the calendar colour too, not only the event one', () => {
    // The calendar colour arrives from the API as stored, and the settings
    // picker writes palette names — so this branch needs resolving just as much.
    expect(resolveEventColor({ calendarId: 'cal-tailwind' }, COLORS)).toBe('#4ade80')
    expect(resolveEventColor({ calendarId: 'cal-named' }, COLORS)).toBe(PALETTE.pink)
  })

  it('keeps a CSS keyword, which was already working', () => {
    // Denis's rule: never remove a capability that worked. `rebeccapurple` and
    // `#abc` painted correctly before resolution was added and must still.
    expect(resolveEventColor({ color: 'rebeccapurple' }, COLORS)).toBe('rebeccapurple')
    expect(resolveEventColor({ color: '#abc' }, COLORS)).toBe('#abc')
  })

  it('treats an unresolvable event colour as a typo and falls through to the calendar', () => {
    // A deliberate mark is a colour; `blue-999` is a slip. Falling through keeps
    // the block visible instead of painting nothing, which is what the raw
    // string did — and what made the defect invisible in the first place.
    expect(resolveEventColor({ color: 'blue-999', calendarId: 'cal-work' }, COLORS)).toBe('#7c3aed')
    expect(resolveEventColor({ color: 'bg-blue-500' }, COLORS)).toBeUndefined()
  })

  it('drops an unresolvable calendar colour rather than emitting it', () => {
    expect(resolveEventColor({ calendarId: 'cal-junk' }, COLORS)).toBeUndefined()
  })
})
