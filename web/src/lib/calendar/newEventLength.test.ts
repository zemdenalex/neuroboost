import { describe, it, expect } from 'vitest'
import { readNewEventLength } from './newEventLength'

describe('readNewEventLength', () => {
  it('uses the stored length when it is one the page offers', () => {
    expect(readNewEventLength({ new_event_minutes: 90 })).toBe(90)
  })

  it('falls back to 30 for a missing, odd or hand-edited value', () => {
    expect(readNewEventLength(undefined)).toBe(30)
    expect(readNewEventLength({})).toBe(30)
    expect(readNewEventLength({ new_event_minutes: 37 })).toBe(30)
    expect(readNewEventLength({ new_event_minutes: '60' })).toBe(30)
  })
})
