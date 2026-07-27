import { describe, it, expect } from 'vitest'
import { parseBinding, shouldSkipInTextField } from './useGlobalQuickAdd'

describe('parseBinding', () => {
  it('parses the default Ctrl+K', () => {
    expect(parseBinding('Ctrl+K')).toEqual({ ctrl: true, alt: false, shift: false, key: 'k' })
  })

  it('parses multiple modifiers in any order', () => {
    expect(parseBinding('Alt+Shift+N')).toEqual({ ctrl: false, alt: true, shift: true, key: 'n' })
    expect(parseBinding('Shift+Alt+N')).toEqual({ ctrl: false, alt: true, shift: true, key: 'n' })
  })

  it('accepts "Control" as a spelling of Ctrl', () => {
    expect(parseBinding('Control+K')?.ctrl).toBe(true)
  })

  it('parses a bare key with no modifiers', () => {
    expect(parseBinding('F2')).toEqual({ ctrl: false, alt: false, shift: false, key: 'f2' })
  })

  it('ignores surrounding whitespace', () => {
    expect(parseBinding(' Ctrl + K ')).toEqual({ ctrl: true, alt: false, shift: false, key: 'k' })
  })

  it('returns null for an unusable binding rather than binding something surprising', () => {
    expect(parseBinding('')).toBeNull()
    expect(parseBinding('+')).toBeNull()
    expect(parseBinding('Ctrl+')).toBeNull()
  })
})

describe('shouldSkipInTextField', () => {
  it('skips a bare shortcut while the user is typing', () => {
    // A plain "N" must never steal a keystroke from an input.
    expect(shouldSkipInTextField(parseBinding('N')!)).toBe(true)
    expect(shouldSkipInTextField(parseBinding('Shift+N')!)).toBe(true)
  })

  it('does NOT skip a Ctrl/Alt shortcut', () => {
    // This is the fix for the reported "Ctrl+K does nothing on /tasks":
    // that page auto-focuses the quick-add row, so focus is ALWAYS in an
    // input and the shortcut looked broken. Ctrl+K is not typing, so it has
    // no keystroke to steal.
    expect(shouldSkipInTextField(parseBinding('Ctrl+K')!)).toBe(false)
    expect(shouldSkipInTextField(parseBinding('Alt+N')!)).toBe(false)
    expect(shouldSkipInTextField(parseBinding('Ctrl+Shift+K')!)).toBe(false)
  })
})
