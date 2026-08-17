import { describe, it, expect } from 'vitest'
import {
  addPreset,
  deletePreset,
  duplicateOffsetGroups,
  renamePreset,
  validatePresetName,
} from './presets'
import { DAY, HOUR, matchPreset, type ReminderSettings } from './offsets'

/**
 * A settings object whose defaults BOTH name `обычное`.
 *
 * That is deliberate and load-bearing: a fixture whose defaults name some
 * other preset would let every "the defaults follow the rename" assertion pass
 * without the code doing anything. Checked by sabotage — dropping the
 * repointing lines from renamePreset and deletePreset reddens these tests.
 */
function fixture(over: Partial<ReminderSettings> = {}): ReminderSettings {
  return {
    presets: {
      'важное': [DAY, HOUR],
      'обычное': [HOUR],
      'без': [],
    },
    default_event_preset: 'обычное',
    default_task_preset: 'обычное',
    digest_at: '08:00',
    digest_enabled: true,
    quiet_hours_respected: true,
    ...over,
  }
}

describe('validatePresetName', () => {
  it('refuses a blank or whitespace-only name', () => {
    expect(validatePresetName('', fixture().presets)).toBe('empty')
    expect(validatePresetName('   ', fixture().presets)).toBe('empty')
  })

  it('refuses a name another preset already holds', () => {
    expect(validatePresetName('важное', fixture().presets)).toBe('duplicate')
    expect(validatePresetName('  важное  ', fixture().presets)).toBe('duplicate')
  })

  it('lets a preset keep its own name while being renamed', () => {
    expect(validatePresetName('важное', fixture().presets, 'важное')).toBeNull()
  })

  it('accepts a free name', () => {
    expect(validatePresetName('дедлайн', fixture().presets)).toBeNull()
  })

  // Object.prototype keys are not presets. Without hasOwnProperty this would
  // report 'duplicate' for a name nobody has used.
  it('does not mistake an inherited property for an existing preset', () => {
    expect(validatePresetName('toString', fixture().presets)).toBeNull()
    expect(validatePresetName('constructor', fixture().presets)).toBeNull()
  })
})

describe('addPreset', () => {
  it('appends an empty preset under the trimmed name', () => {
    const r = addPreset(fixture(), '  дедлайн ')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.settings.presets['дедлайн']).toEqual([])
    expect(Object.keys(r.settings.presets)).toEqual(['важное', 'обычное', 'без', 'дедлайн'])
  })

  it('refuses a duplicate rather than overwriting the existing preset', () => {
    // The failure this guards: `presets[name] = []` on an existing key does not
    // add a preset, it empties one — and an emptied preset is a schedule that
    // silently never fires.
    const r = addPreset(fixture(), 'важное')
    expect(r).toEqual({ ok: false, reason: 'duplicate' })
  })

  it('leaves the original settings untouched', () => {
    const before = fixture()
    addPreset(before, 'дедлайн')
    expect(Object.keys(before.presets)).toEqual(['важное', 'обычное', 'без'])
  })
})

describe('renamePreset', () => {
  it('renames in place, keeping the offsets and the position', () => {
    const r = renamePreset(fixture(), 'обычное', 'стандарт')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(Object.keys(r.settings.presets)).toEqual(['важное', 'стандарт', 'без'])
    expect(r.settings.presets['стандарт']).toEqual([HOUR])
  })

  // 🔴 The whole reason this module exists. A default naming a preset that no
  // longer exists resolves to no offsets in Go, so every event created
  // afterwards is stored with an empty offset array and is skipped by the
  // scanner — a reminder that never arrives and never reports anything.
  it('moves the defaults that named the old preset', () => {
    const r = renamePreset(fixture(), 'обычное', 'стандарт')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.settings.default_event_preset).toBe('стандарт')
    expect(r.settings.default_task_preset).toBe('стандарт')
  })

  it('leaves defaults that named a different preset alone', () => {
    const r = renamePreset(fixture(), 'важное', 'критично')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.settings.default_event_preset).toBe('обычное')
  })

  it('refuses a name another preset holds', () => {
    expect(renamePreset(fixture(), 'обычное', 'важное')).toEqual({ ok: false, reason: 'duplicate' })
  })

  it('refuses a blank name', () => {
    expect(renamePreset(fixture(), 'обычное', '  ')).toEqual({ ok: false, reason: 'empty' })
  })

  it('reports an unknown preset instead of inventing one', () => {
    expect(renamePreset(fixture(), 'нетакого', 'что-то')).toEqual({ ok: false, reason: 'unknown' })
  })

  it('accepts renaming a preset to what it is already called', () => {
    const r = renamePreset(fixture(), 'обычное', 'обычное')
    expect(r.ok).toBe(true)
  })
})

describe('deletePreset', () => {
  it('removes the preset and keeps the rest in order', () => {
    const r = deletePreset(fixture(), 'важное')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(Object.keys(r.settings.presets)).toEqual(['обычное', 'без'])
  })

  it('moves a default that named the deleted preset onto a survivor', () => {
    const r = deletePreset(fixture(), 'обычное')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(Object.keys(r.settings.presets)).toContain(r.settings.default_event_preset)
    expect(Object.keys(r.settings.presets)).toContain(r.settings.default_task_preset)
  })

  it('leaves untouched defaults alone', () => {
    const r = deletePreset(fixture(), 'без')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.settings.default_event_preset).toBe('обычное')
  })

  // Saving an empty map does not delete anything: both readers treat `{}` as
  // "no presets stored" and hand back the three built-ins, so the deletion
  // would appear to be a reset.
  it('refuses to delete the last preset', () => {
    const one = fixture({ presets: { 'обычное': [HOUR] } })
    expect(deletePreset(one, 'обычное')).toEqual({ ok: false, reason: 'last' })
  })

  it('reports an unknown preset', () => {
    expect(deletePreset(fixture(), 'нетакого')).toEqual({ ok: false, reason: 'unknown' })
  })
})

describe('duplicateOffsetGroups', () => {
  it('finds nothing when every preset differs', () => {
    expect(duplicateOffsetGroups(fixture().presets)).toEqual([])
  })

  it('names the presets that share one schedule', () => {
    const presets = { 'a': [HOUR], 'b': [HOUR], 'c': [DAY] }
    expect(duplicateOffsetGroups(presets)).toEqual([['a', 'b']])
  })

  it('ignores the order the offsets were written in', () => {
    const presets = { 'a': [HOUR, DAY], 'b': [DAY, HOUR] }
    expect(duplicateOffsetGroups(presets)).toEqual([['a', 'b']])
  })

  // The reported symptom, stated as a test so the warning's premise is not
  // just asserted in a comment: with two names on one schedule, matchPreset
  // answers with the first, so selecting the second reads back as the first.
  it('marks exactly the case where matchPreset becomes ambiguous', () => {
    const presets = { 'a': [HOUR], 'b': [HOUR] }
    expect(duplicateOffsetGroups(presets)).toEqual([['a', 'b']])
    expect(matchPreset(presets['b'], presets)).toBe('a')
  })
})
