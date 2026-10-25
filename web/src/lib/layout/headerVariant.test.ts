import { describe, it, expect } from 'vitest'
import { effectiveHeaderVariant } from './headerVariant'

describe('effectiveHeaderVariant', () => {
  it('gives a phone the top bar whatever was saved', () => {
    expect(effectiveHeaderVariant('vertical', true)).toBe('horizontal')
    expect(effectiveHeaderVariant('horizontal', true)).toBe('horizontal')
  })
  it('keeps the saved choice on a desktop', () => {
    expect(effectiveHeaderVariant('vertical', false)).toBe('vertical')
    expect(effectiveHeaderVariant('horizontal', false)).toBe('horizontal')
    expect(effectiveHeaderVariant(null, false)).toBe('horizontal')
    expect(effectiveHeaderVariant('garbage', false)).toBe('horizontal')
  })
})
