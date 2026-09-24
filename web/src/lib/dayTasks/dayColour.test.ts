import { describe, it, expect } from 'vitest'
import { dayColours, dayLevelSquare, todayInZone, type Day } from './dayColour'

const day = (d: Partial<Day> & { day: string }): Day => ({
  target: 5,
  confirmed: false,
  items: [],
  done: 0,
  level: 0,
  before_start: false,
  ...d,
})

// The same table as bot/internal/handlers/daycolour_test.go: two clients, one
// rule (spec 2026-09-22 §11). Change one, change both.
describe('dayColours', () => {
  const days = [
    day({ day: '2026-09-21', before_start: true }),
    day({ day: '2026-09-22', level: 0 }),
    day({ day: '2026-09-23', confirmed: true, level: 3 }),
    day({ day: '2026-09-24' }),
    day({ day: '2026-09-25', confirmed: true, level: 5 }),
  ]

  it('paints the past after the start, not the future, not an untaken today', () => {
    expect(dayColours(days, '2026-09-24', false)).toEqual({ '2026-09-22': '⬛', '2026-09-23': '🟧' })
  })

  it('paints days before the start only when chosen', () => {
    expect(dayColours(days, '2026-09-24', true)['2026-09-21']).toBe('⬛')
  })

  it('paints today once it is taken', () => {
    const taken = days.map((d) => (d.day === '2026-09-24' ? { ...d, confirmed: true, level: 1 } : d))
    expect(dayColours(taken, '2026-09-24', false)['2026-09-24']).toBe('🟫')
  })
})

describe('dayLevelSquare', () => {
  it('draws the six levels', () => {
    expect([0, 1, 2, 3, 4, 5].map(dayLevelSquare).join('')).toBe('⬛🟫🟥🟧🟨🟩')
  })

  // Review Focus 3: a level the server should never send is ⬛, as in the bot.
  it('draws ⬛ for a level out of range', () => {
    expect(dayLevelSquare(-1)).toBe('⬛')
    expect(dayLevelSquare(9)).toBe('⬛')
  })
})

describe('todayInZone', () => {
  // Review Focus 2: 00:30 in Moscow is still the 23rd in UTC; the day is
  // Moscow's, whatever the browser clock says.
  it("is the person's day, not UTC's", () => {
    const now = new Date('2026-09-23T21:30:00Z')
    expect(todayInZone(now, 'Europe/Moscow')).toBe('2026-09-24')
    expect(todayInZone(now, 'UTC')).toBe('2026-09-23')
  })

  it('falls back to Moscow for a zone the browser does not know', () => {
    expect(todayInZone(new Date('2026-09-23T21:30:00Z'), 'Not/AZone')).toBe('2026-09-24')
  })
})
