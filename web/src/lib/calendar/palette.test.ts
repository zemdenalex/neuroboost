import { describe, it, expect } from 'vitest'
import { resolveColor, isValidColor, defaultCalendarColor, PALETTE, PALETTE_NAMES } from './palette'

describe('resolveColor', () => {
  it('resolves a palette name to its hex value', () => {
    expect(resolveColor('blue')).toBe(PALETTE.blue)
    expect(resolveColor('violet')).toBe(PALETTE.violet)
  })

  it('passes a hex value through, long and short form', () => {
    expect(resolveColor('#2563eb')).toBe('#2563eb')
    expect(resolveColor('#abc')).toBe('#abc')
    expect(resolveColor('#ABCDEF')).toBe('#ABCDEF')
  })

  // 🔴 The reported bug. A Tailwind class is not a CSS colour, and the old
  // preview turned it into `var(--blue-400)` — a variable that does not exist,
  // so the swatch stayed blank and the event kept the default styling.
  it('rejects a Tailwind class name', () => {
    expect(resolveColor('blue-400')).toBeUndefined()
    expect(resolveColor('bg-blue-500')).toBeUndefined()
    expect(resolveColor('text-red-600')).toBeUndefined()
  })

  it('rejects malformed hex', () => {
    expect(resolveColor('#12')).toBeUndefined()
    expect(resolveColor('#1234')).toBeUndefined()
    expect(resolveColor('#gggggg')).toBeUndefined()
    expect(resolveColor('2563eb')).toBeUndefined() // missing #
  })

  it('treats empty and blank as absent', () => {
    expect(resolveColor('')).toBeUndefined()
    expect(resolveColor('   ')).toBeUndefined()
    expect(resolveColor(null)).toBeUndefined()
    expect(resolveColor(undefined)).toBeUndefined()
  })

  it('ignores surrounding whitespace on a valid value', () => {
    expect(resolveColor('  blue  ')).toBe(PALETTE.blue)
    expect(resolveColor(' #abc ')).toBe('#abc')
  })

  // Deliberate: a browser paints "rebeccapurple", but the picker could never
  // show it as selected, so a value the UI cannot represent is refused.
  it('refuses CSS keywords outside the palette', () => {
    expect(resolveColor('rebeccapurple')).toBeUndefined()
    expect(resolveColor('transparent')).toBeUndefined()
  })

  it('every palette entry is itself a valid colour', () => {
    // Guards the palette against a typo that would make one swatch unpaintable.
    for (const name of PALETTE_NAMES) {
      expect(resolveColor(name), `palette entry ${name}`).toBeDefined()
      expect(resolveColor(PALETTE[name]), `hex of ${name}`).toBe(PALETTE[name])
    }
  })
})

describe('isValidColor', () => {
  it('agrees with resolveColor', () => {
    expect(isValidColor('blue')).toBe(true)
    expect(isValidColor('#abc')).toBe(true)
    expect(isValidColor('blue-400')).toBe(false)
    expect(isValidColor('')).toBe(false)
  })
})

describe('defaultCalendarColor', () => {
  it('gives consecutive calendars different colours', () => {
    expect(defaultCalendarColor(0)).not.toBe(defaultCalendarColor(1))
    expect(defaultCalendarColor(1)).not.toBe(defaultCalendarColor(2))
  })

  it('cycles rather than running out', () => {
    const n = PALETTE_NAMES.length
    expect(defaultCalendarColor(n)).toBe(defaultCalendarColor(0))
    expect(defaultCalendarColor(n + 3)).toBe(defaultCalendarColor(3))
  })

  it('is deterministic, so the same list twice looks the same', () => {
    expect(defaultCalendarColor(5)).toBe(defaultCalendarColor(5))
  })

  // The failure this guards: an out-of-range index returning undefined would
  // paint nothing, which is exactly the colourless calendar being fixed.
  it('always returns a paintable colour, whatever the index', () => {
    for (const i of [0, 3, 99, -1, -7, 1.5, NaN, Infinity]) {
      const color = defaultCalendarColor(i)
      expect(isValidColor(color), `index ${i} gave ${color}`).toBe(true)
    }
  })
})
