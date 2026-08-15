import tailwindColors from 'tailwindcss/colors'

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
 * Accepts, in order:
 *   - a swatch name from PALETTE          `"violet"`
 *   - a hex value, long or short          `"#2563eb"`, `"#abc"`
 *   - a Tailwind colour                   `"blue-400"`, `"gray-400"`
 *   - a CSS colour keyword                `"blue"`, `"red"`, `"rebeccapurple"`
 *
 * 🔴 The swatches are an ADDITION, not a replacement. An earlier version of
 * this function accepted only the first two and the field became swatch-only,
 * which took away arbitrary colours that already worked. Denis's rule, and it
 * is a good one: never remove a working capability to make room for a simpler
 * one. The picker is for people who want to choose quickly; the text field
 * stays for everyone else.
 *
 * Tailwind values come from the tailwindcss package itself rather than a
 * hand-copied table — a table would drift from the classes used everywhere
 * else in this app, and drift is invisible until someone types a shade that
 * used to exist.
 *
 * Anything unrecognised returns undefined so the caller falls back to default
 * styling rather than painting nothing while appearing set.
 */
export function resolveColor(input: string | null | undefined): string | undefined {
  if (typeof input !== 'string') return undefined
  const value = input.trim()
  if (value === '') return undefined

  if (value in PALETTE) return PALETTE[value as PaletteName]
  if (HEX.test(value)) return value

  const tailwind = resolveTailwind(value)
  if (tailwind) return tailwind

  return resolveCssKeyword(value)
}

/**
 * `blue-400`, `gray-400`, and the bare hues Tailwind ships as objects.
 *
 * A bare hue (`"blue"`) is NOT resolved here on purpose — Tailwind has no
 * single value for it, and CSS does, so it falls through to the keyword check
 * below and means the CSS colour. That is what someone typing "blue" expects.
 */
function resolveTailwind(value: string): string | undefined {
  const match = /^([a-z]+)-(\d{2,3})$/.exec(value)
  if (!match) return undefined

  const shades = (tailwindColors as unknown as Record<string, unknown>)[match[1]]
  if (!shades || typeof shades !== 'object') return undefined

  const hex = (shades as Record<string, unknown>)[match[2]]
  return typeof hex === 'string' ? hex : undefined
}

/**
 * A CSS colour keyword, checked by asking the browser rather than by keeping a
 * list of 148 names that would need maintaining.
 *
 * CSS.supports is absent in some test environments, so the fallback accepts a
 * bare word: an unknown one simply paints nothing, which is the same outcome as
 * before this function existed and strictly better than refusing a colour the
 * browser understands.
 */
function resolveCssKeyword(value: string): string | undefined {
  if (!/^[a-zA-Z]+$/.test(value)) return undefined

  const supports = typeof CSS !== 'undefined' && typeof CSS.supports === 'function'
  if (supports) return CSS.supports('color', value) ? value : undefined
  return value
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
