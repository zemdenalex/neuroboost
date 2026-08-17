import { describe, it, expect } from 'vitest'
import { presetLabel, isBuiltInPreset, BUILT_IN_PRESET_NAMES } from './presetLabel'
// The shipped files, not a hand-written copy of them.
import en from '../../i18n/locales/en/reminders.json'
import ru from '../../i18n/locales/ru/reminders.json'

/**
 * Stands in for i18next: returns the key when there is no translation.
 *
 * 🔴 `String(key)`, not `key`. The first version returned the key itself, and
 * that difference hid a real defect: when presetLabel looked a name up through
 * a prototype-inheriting table, `key` was Object.prototype.toString — a
 * FUNCTION — and this fake handed the identical reference back, so
 * `translated === key` was true and the test took the safe branch. Real
 * i18next stringifies (`String(keys)`) and can never return the same
 * reference, so production took the other one and rendered
 * "function toString() { [native code] }".
 *
 * A stand-in that is kinder than the real thing is not a test.
 */
function translator(dict: Record<string, string>) {
  return (key: string) => dict[key] ?? String(key)
}

const t = translator({
  'presetName.important': 'Important',
  'presetName.normal': 'Normal',
  'presetName.none': 'None',
})

describe('presetLabel', () => {
  it('translates the three built-in presets', () => {
    expect(presetLabel('важное', t)).toBe('Important')
    expect(presetLabel('обычное', t)).toBe('Normal')
    expect(presetLabel('без', t)).toBe('None')
  })

  // Someone's own preset is their text. Translating it would mean showing them
  // a name they never chose.
  it('leaves a name the user chose alone', () => {
    expect(presetLabel('дедлайн', t)).toBe('дедлайн')
    expect(presetLabel('Important', t)).toBe('Important')
    expect(presetLabel('', t)).toBe('')
  })

  // i18next hands back the key when a translation is missing, and rendering
  // "presetName.important" is worse than rendering the Russian.
  it('falls back to the stored name when the translation is missing', () => {
    expect(presetLabel('важное', translator({}))).toBe('важное')
  })

  it('does not mistake an inherited property for a built-in', () => {
    expect(presetLabel('toString', t)).toBe('toString')
    expect(isBuiltInPreset('constructor')).toBe(false)
  })
})

describe('the label table and the shipped presets agree', () => {
  it('every shipped preset has a label', () => {
    for (const name of BUILT_IN_PRESET_NAMES) {
      expect(isBuiltInPreset(name), `no label for shipped preset ${name}`).toBe(true)
    }
  })

  // 🔴 The test above ties two in-code maps to each other and nothing else.
  // The comment used to claim it caught "a preset reading обычное in an
  // otherwise English interface" — it did not: deleting the whole presetName
  // block from the English locale left all tests green while the interface
  // silently reverted to Russian. These two read the shipped files.
  it.each(['важное', 'обычное', 'без'])('%s is translated by the real English locale', name => {
    const t = (key: string) => {
      const parts = key.split('.')
      let node: unknown = en
      for (const part of parts) node = (node as Record<string, unknown>)?.[part]
      return typeof node === 'string' ? node : String(key)
    }
    const label = presetLabel(name, t)
    expect(label, `${name} has no English translation in the shipped locale`).not.toBe(name)
  })

  it('the Russian locale carries the same three keys', () => {
    expect(ru.presetName?.important, 'ru.presetName.important is missing').toBeTruthy()
    expect(ru.presetName?.normal, 'ru.presetName.normal is missing').toBeTruthy()
    expect(ru.presetName?.none, 'ru.presetName.none is missing').toBeTruthy()
  })
})
