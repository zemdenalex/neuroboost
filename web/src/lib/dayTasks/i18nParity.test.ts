import { describe, it, expect } from 'vitest'
import ru from '../../i18n/locales/ru/daytasks.json'
import en from '../../i18n/locales/en/daytasks.json'

function keys(o: Record<string, unknown>, prefix = ''): string[] {
  return Object.entries(o).flatMap(([k, v]) =>
    v && typeof v === 'object' ? keys(v as Record<string, unknown>, prefix + k + '.') : [prefix + k],
  )
}

function values(o: Record<string, unknown>): string[] {
  return Object.values(o).flatMap((v) => (v && typeof v === 'object' ? values(v as Record<string, unknown>) : [String(v)]))
}

describe('daytasks texts', () => {
  it('has every key in both languages', () => {
    expect(keys(ru).sort()).toEqual(keys(en).sort())
  })

  // Denis: no long dashes in what people read.
  it('has no long dashes', () => {
    for (const v of [...values(ru), ...values(en)]) expect(v).not.toContain('—')
  })
})
