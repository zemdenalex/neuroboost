import { describe, it, expect } from 'vitest'
import { presetLabel, isBuiltInPreset, BUILT_IN_PRESET_NAMES } from './presetLabel'

/** Stands in for i18next: returns the key when there is no translation. */
function translator(dict: Record<string, string>) {
  return (key: string) => dict[key] ?? key
}

const en = translator({
  'presetName.important': 'Important',
  'presetName.normal': 'Normal',
  'presetName.none': 'None',
})

describe('presetLabel', () => {
  it('translates the three built-in presets', () => {
    expect(presetLabel('важное', en)).toBe('Important')
    expect(presetLabel('обычное', en)).toBe('Normal')
    expect(presetLabel('без', en)).toBe('None')
  })

  // Someone's own preset is their text. Translating it would mean showing them
  // a name they never chose.
  it('leaves a name the user chose alone', () => {
    expect(presetLabel('дедлайн', en)).toBe('дедлайн')
    expect(presetLabel('Important', en)).toBe('Important')
    expect(presetLabel('', en)).toBe('')
  })

  // i18next hands back the key when a translation is missing, and rendering
  // "presetName.important" is worse than rendering the Russian.
  it('falls back to the stored name when the translation is missing', () => {
    expect(presetLabel('важное', translator({}))).toBe('важное')
  })

  it('does not mistake an inherited property for a built-in', () => {
    expect(presetLabel('toString', en)).toBe('toString')
    expect(isBuiltInPreset('constructor')).toBe(false)
  })
})

describe('the label table and the shipped presets agree', () => {
  // 🔴 The failure this guards: a typo in one of the three keys would leave
  // that preset untranslated while everything still looked correct — the same
  // preset would read "обычное" in an otherwise English interface and nobody
  // would know which half was wrong.
  it('every shipped preset has a label', () => {
    for (const name of BUILT_IN_PRESET_NAMES) {
      expect(isBuiltInPreset(name), `no label for shipped preset ${name}`).toBe(true)
    }
  })
})
