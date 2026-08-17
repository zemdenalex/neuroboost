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

  // 🔴 The reported bug, and the correction that followed. "blue-400" used to
  // resolve to `var(--blue-400)` — a variable that does not exist — so the
  // swatch stayed blank. The first fix refused it outright, which took away a
  // colour space that people were already using. It now resolves through
  // Tailwind's own palette.
  it('resolves a Tailwind colour to its hex value', () => {
    expect(resolveColor('blue-400')).toBe('#60a5fa')
    expect(resolveColor('gray-400')).toBe('#9ca3af')
    expect(resolveColor('emerald-600')).toMatch(/^#[0-9a-f]{6}$/i)
  })

  it('still refuses a Tailwind CLASS, which is not a colour', () => {
    // `bg-` and `text-` are utilities, not values; accepting them would store
    // something no colour property can use.
    expect(resolveColor('bg-blue-500')).toBeUndefined()
    expect(resolveColor('text-red-600')).toBeUndefined()
  })

  it('refuses a shade that does not exist', () => {
    expect(resolveColor('blue-999')).toBeUndefined()
    expect(resolveColor('notahue-400')).toBeUndefined()
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

  // A bare hue means the CSS colour, not a Tailwind object: Tailwind has no
  // single value for "blue", CSS does, and that is what someone typing it
  // expects.
  it('accepts CSS colour keywords', () => {
    expect(resolveColor('blue')).toBe(PALETTE.blue) // ours wins — it is in PALETTE
    expect(resolveColor('rebeccapurple')).toBe('rebeccapurple')
    expect(resolveColor('teal')).toBe('teal')
  })

  it('refuses a word that is not a colour at all', () => {
    // jsdom may not implement CSS.supports; when it does not, the fallback
    // accepts bare words and this expectation is skipped rather than asserted
    // falsely.
    const canValidate = typeof CSS !== 'undefined' && typeof CSS.supports === 'function'
    if (!canValidate) return
    expect(resolveColor('notacolour')).toBeUndefined()
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
    expect(isValidColor('blue-400')).toBe(true)
    expect(isValidColor('bg-blue-500')).toBe(false)
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
