import { describe, it, expect } from 'vitest'
import { squareColour } from './squareColour'

describe('squareColour', () => {
  it('has a colour for every square the day-task rule can give', () => {
    for (const sq of ['⬛', '🟫', '🟥', '🟧', '🟨', '🟩']) expect(squareColour(sq)).toMatch(/^#[0-9a-f]{6}$/)
  })

  it('is green for a full day and nothing for no square or an unknown one', () => {
    expect(squareColour('🟩')).toBe('#22c55e')
    expect(squareColour(undefined)).toBeUndefined()
    expect(squareColour('x')).toBeUndefined()
  })
})
