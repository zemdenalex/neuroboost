import { describe, it, expect } from 'vitest'
import { readAddMode, readPhoneHeader } from './phoneCalendar'

describe('readPhoneHeader', () => {
  // The e2e account keeps phone_month_variant = strip; deriving the header
  // from it would move every phone spec into the strip header unasked.
  it('is the one-row header until chosen, whatever the old month choice', () => {
    expect(readPhoneHeader(undefined)).toBe('row')
    expect(readPhoneHeader({ phone_header_variant: 'bogus' } as { phone_header_variant?: unknown })).toBe('row')
    expect(readPhoneHeader({ phone_month_variant: 'strip' } as { phone_header_variant?: unknown })).toBe('row')
  })

  it('keeps the strip header once chosen', () => {
    expect(readPhoneHeader({ phone_header_variant: 'strip' })).toBe('strip')
  })
})

describe('readAddMode', () => {
  it('is simple until chosen, detailed once chosen', () => {
    expect(readAddMode(undefined)).toBe('simple')
    expect(readAddMode({ add_mode: 42 })).toBe('simple')
    expect(readAddMode({ add_mode: 'detailed' })).toBe('detailed')
  })
})
