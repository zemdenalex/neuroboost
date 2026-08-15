/**
 * The colours a calendar or an event may be.
 *
 * Added 2026-08-15 for two problems that turned out to be one. Calendars were
 * created with no colour at all — `createCalendar(name)` never passed one — so
 * every calendar was colourless and the grid could not tell them apart. And the
 * event colour field was free text: Denis reported that "blue" or "blue-400"
 * did nothing, which is correct, because a Tailwind class name is not a CSS
 * colour and the preview resolved it to `var(--blue-400)`, a variable that does
 * not exist.
 *
 * One named palette answers both: calendars get one by default, and the field
 * accepts a palette name or a hex value and nothing else.
 */

/** Palette entries. Names are stable identifiers, not display text. */
export const PALETTE = {
  violet: '#7c3aed',
  blue: '#2563eb',
  cyan: '#0891b2',
  green: '#16a34a',
  amber: '#d97706',
  red: '#dc2626',
  pink: '#db2777',
  slate: '#475569',
} as const

export type PaletteName = keyof typeof PALETTE

export const PALETTE_NAMES = Object.keys(PALETTE) as PaletteName[]

const HEX = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/

/**
 * Turn whatever is stored into something a browser will actually paint.
 *
 * Accepts a palette name (`"blue"`) or a hex value (`"#2563eb"`, `"#abc"`).
 * Everything else — a Tailwind class, an empty string, a typo — returns
 * undefined so the caller falls back to the default styling instead of
 * silently painting nothing.
 *
 * 🔴 Deliberately does NOT accept CSS colour keywords beyond the palette.
 * "rebeccapurple" would work in a browser and then not exist in the swatch
 * list, so the picker could not show it as selected — a value the UI cannot
 * represent is a value that will confuse whoever opens the form next.
 */
export function resolveColor(input: string | null | undefined): string | undefined {
  if (typeof input !== 'string') return undefined
  const value = input.trim()
  if (value === '') return undefined
  if (value in PALETTE) return PALETTE[value as PaletteName]
  if (HEX.test(value)) return value
  return undefined
}

/** True when the stored value is something this app can render and re-select. */
export function isValidColor(input: string | null | undefined): boolean {
  return resolveColor(input) !== undefined
}

/**
 * A default colour for the nth calendar, so a new one is never colourless.
 *
 * Cycles rather than randomising: two calendars created in a row get different
 * colours, and creating the same list twice gives the same result — which
 * matters for anyone reading a screenshot or a test.
 */
export function defaultCalendarColor(index: number): string {
  const names = PALETTE_NAMES
  // A negative or fractional index would land outside the array and return
  // undefined, painting nothing — the exact failure this function exists to
  // prevent.
  const safe = Number.isFinite(index) ? Math.abs(Math.trunc(index)) : 0
  return PALETTE[names[safe % names.length]]
}
