import { describe, it, expect } from 'vitest'
import { parseBinding } from './useGlobalQuickAdd'

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
