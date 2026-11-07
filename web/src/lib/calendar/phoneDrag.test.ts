import { describe, it, expect } from 'vitest'
import { readPhoneDragStart, readPhoneDrag, DEFAULT_PHONE_DRAG, holdLifts, gripShown } from './phoneDrag'

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

// Step 10: the five other ⚙️ choices. What they protect: a value the section
// wrote reads back as that value, and anything else (a missing key, a hand-edited
// blob, a value from an older build) reads as what the phone did before ⚙️ — one
// key's junk must not reset its neighbours.
describe('readPhoneDrag', () => {
  it('is today\'s behaviour when nothing is stored', () => {
    expect(readPhoneDrag(undefined)).toEqual(DEFAULT_PHONE_DRAG)
    expect(DEFAULT_PHONE_DRAG).toEqual({ start: 'hold', delayMs: 400, stepMin: 15, vibrate: true, edgeFlip: true, resize: 'selected' })
  })

  it('reads each stored choice, and junk in one key leaves the others alone', () => {
    expect(readPhoneDrag({
      phone_drag_start: 'both', phone_drag_delay_ms: 600, phone_drag_step_min: 30,
      phone_drag_vibrate: false, phone_drag_edge_flip: false, phone_drag_resize: 'always',
    })).toEqual({ start: 'both', delayMs: 600, stepMin: 30, vibrate: false, edgeFlip: false, resize: 'always' })
    expect(readPhoneDrag({
      phone_drag_delay_ms: 500, phone_drag_step_min: '30', phone_drag_vibrate: 'no',
      phone_drag_resize: 'sometimes', phone_drag_edge_flip: false,
    })).toEqual({ ...DEFAULT_PHONE_DRAG, edgeFlip: false })
  })
})
