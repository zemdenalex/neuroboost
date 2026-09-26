import { describe, expect, it } from 'vitest'
import { hasClockTime } from './QuickAddRow'

describe('hasClockTime', () => {
  it('finds a clock time', () => {
    for (const s of ['стоматолог 15:00', '9.30 зал', 'в 23:59', '07:05']) expect(hasClockTime(s)).toBe(true)
  })
  it('does not take other numbers for a time', () => {
    for (const s of ['купить 2 кг', '30м', '24:00 нет', '15:600', 'дом 3.5']) expect(hasClockTime(s)).toBe(false)
  })
})
