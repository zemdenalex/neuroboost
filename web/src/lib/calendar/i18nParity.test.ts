import { describe, it, expect } from 'vitest'
import ru from '../../i18n/locales/ru/calendar.json'
import en from '../../i18n/locales/en/calendar.json'
import ruSettings from '../../i18n/locales/ru/settings.json'
import enSettings from '../../i18n/locales/en/settings.json'

function keys(o: Record<string, unknown>, prefix = ''): string[] {
  return Object.entries(o).flatMap(([k, v]) =>
    v && typeof v === 'object' ? keys(v as Record<string, unknown>, prefix + k + '.') : [prefix + k],
  )
}

function values(o: Record<string, unknown>): string[] {
  return Object.values(o).flatMap((v) => (v && typeof v === 'object' ? values(v as Record<string, unknown>) : [String(v)]))
}

describe('calendar texts', () => {
  it('has every key in both languages', () => {
    expect(keys(ru).sort()).toEqual(keys(en).sort())
  })

  // Denis: no long dashes in what people read.
  it('has no long dashes', () => {
    for (const v of [...values(ru), ...values(en)]) expect(v).not.toContain('—')
  })
})

describe('month view settings texts', () => {
  it('has every key in both languages', () => {
    expect(keys(ruSettings.monthView).sort()).toEqual(keys(enSettings.monthView).sort())
  })

  it('names all five variants and has no long dashes', () => {
    for (const v of ['list', 'classic', 'heat', 'split', 'commit']) {
      expect(keys(ruSettings.monthView)).toContain(`${v}.name`)
    }
    for (const v of [...values(ruSettings.monthView), ...values(enSettings.monthView)]) expect(v).not.toContain('—')
  })
})

describe('settings texts', () => {
  // Denis: no long dashes in what people read.
  it('has no long dashes anywhere in the file', () => {
    for (const v of [...values(ruSettings), ...values(enSettings)]) expect(v).not.toContain('—')
  })
})
