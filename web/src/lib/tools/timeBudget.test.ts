import { describe, it, expect } from 'vitest'
import { readBudgetCategories, workDayCount, workHoursPerDay } from './timeBudget'

// Denis 25.09 (docs/tasks-web-cleanup.md 4.10): «Бюджет времени» from the work
// hours in the account, saved to the account, no decorative week.
describe('workHoursPerDay', () => {
  it('is the span between work start and end', () => {
    expect(workHoursPerDay('09:00', '18:00')).toBe(9)
    expect(workHoursPerDay('09:30', '17:00')).toBe(7.5)
  })
  it('is null when the hours are missing or make no span', () => {
    expect(workHoursPerDay(undefined, '18:00')).toBeNull()
    expect(workHoursPerDay('18:00', '09:00')).toBeNull()
    expect(workHoursPerDay('9am', '18:00')).toBeNull()
  })
})

describe('workDayCount', () => {
  it('counts the chosen days, five when none are set', () => {
    expect(workDayCount(['mon', 'tue', 'wed'])).toBe(3)
    expect(workDayCount(undefined)).toBe(5)
    expect(workDayCount([])).toBe(5)
  })
})

describe('readBudgetCategories', () => {
  it('keeps well-formed categories from the account', () => {
    const cats = [{ id: 'a', labelKey: '', customLabel: 'Учёба', hours: 2, color: 'bg-pink-500', isCustom: true }]
    expect(readBudgetCategories(cats)).toEqual(cats)
  })
  it('is null for anything else, so defaults or the old device copy apply', () => {
    expect(readBudgetCategories(undefined)).toBeNull()
    expect(readBudgetCategories('x')).toBeNull()
    expect(readBudgetCategories([{ id: 'a', hours: 'two' }])).toBeNull()
    // Complete in every field but one: the hours must be a number, not text.
    expect(
      readBudgetCategories([{ id: 'a', labelKey: '', customLabel: 'x', hours: '2', color: 'bg-pink-500', isCustom: true }]),
    ).toBeNull()
  })
})
