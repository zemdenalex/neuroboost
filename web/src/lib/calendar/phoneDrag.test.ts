import { describe, it, expect } from 'vitest'
import { readPhoneDragStart, holdLifts, gripShown } from './phoneDrag'

describe('readPhoneDragStart', () => {
  it('is the hold when nothing or something unknown is stored', () => {
    expect(readPhoneDragStart(undefined)).toBe('hold')
    expect(readPhoneDragStart({})).toBe('hold')
    expect(readPhoneDragStart({ phone_drag_start: 'grip' })).toBe('hold')
    expect(readPhoneDragStart({ phone_drag_start: 1 })).toBe('hold')
  })

  it('reads the three choices, and each turns on the right ways in', () => {
    const ways = (v: string) => {
      const s = readPhoneDragStart({ phone_drag_start: v })
      return [s, holdLifts(s), gripShown(s)]
    }
    expect(ways('hold')).toEqual(['hold', true, false])
    expect(ways('handle')).toEqual(['handle', false, true])
    expect(ways('both')).toEqual(['both', true, true])
  })
})
