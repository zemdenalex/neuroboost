import { describe, expect, it } from 'vitest'
import {
  DEFAULT_REMINDER_PRESETS,
  addOffset,
  formatOffset,
  matchPreset,
  normalizeOffsets,
  parseOffsetInput,
  removeOffset,
  resolveReminderSettings,
} from './offsets'

describe('normalizeOffsets', () => {
  it('sorts furthest-out first, the order a person reads them in', () => {
    // "month, week, day, hour" is how Denis described the set; showing 60
    // before 43200 would read as a different schedule.
    expect(normalizeOffsets([60, 43200, 1440])).toEqual([43200, 1440, 60])
  })

  it('drops duplicates so one reminder is one row', () => {
    expect(normalizeOffsets([60, 60, 1440])).toEqual([1440, 60])
  })

  it('drops negatives — they are sentinels in the journal, not offsets', () => {
    // -1 is snooze and -2 is digest on the backend; letting either in from
    // the UI would collide with a real reminder's dedupe key.
    expect(normalizeOffsets([-1, -2, 60])).toEqual([60])
  })

  it('drops non-finite and non-integer values', () => {
    expect(normalizeOffsets([Number.NaN, Infinity, 10.5, 30])).toEqual([30])
  })

  it('keeps zero — "at the moment it starts" is a real choice', () => {
    expect(normalizeOffsets([0, 60])).toEqual([60, 0])
  })

  it('returns a new array rather than mutating its argument', () => {
    const input = [60, 1440]
    const out = normalizeOffsets(input)
    expect(input).toEqual([60, 1440])
    expect(out).not.toBe(input)
  })
})

describe('addOffset / removeOffset', () => {
  it('adding an existing offset is a no-op, not a duplicate', () => {
    expect(addOffset([1440, 60], 60)).toEqual([1440, 60])
  })

  it('adds and re-sorts', () => {
    expect(addOffset([60], 1440)).toEqual([1440, 60])
  })

  it('removes by value', () => {
    expect(removeOffset([1440, 60], 60)).toEqual([1440])
  })

  it('removing something absent leaves the list alone', () => {
    expect(removeOffset([1440], 60)).toEqual([1440])
  })

  it('removing the last one yields an explicit empty list', () => {
    // Empty means "deliberately no reminders" on the wire — distinct from
    // omitting the field, which means "use my default preset".
    expect(removeOffset([60], 60)).toEqual([])
  })
})

describe('parseOffsetInput', () => {
  it('reads a bare number as minutes', () => {
    expect(parseOffsetInput('30')).toBe(30)
  })

  it('reads unit suffixes in both languages', () => {
    expect(parseOffsetInput('2h')).toBe(120)
    expect(parseOffsetInput('2ч')).toBe(120)
    expect(parseOffsetInput('3d')).toBe(4320)
    expect(parseOffsetInput('3д')).toBe(4320)
    expect(parseOffsetInput('1w')).toBe(10080)
    expect(parseOffsetInput('1н')).toBe(10080)
  })

  it('tolerates spacing and case', () => {
    expect(parseOffsetInput('  2 H ')).toBe(120)
  })

  it('rejects nonsense rather than guessing', () => {
    expect(parseOffsetInput('')).toBeNull()
    expect(parseOffsetInput('soon')).toBeNull()
    expect(parseOffsetInput('-5')).toBeNull()
    expect(parseOffsetInput('2x')).toBeNull()
  })

  it('rejects offsets beyond a month — the backend scan looks no further', () => {
    // maxOffsetMinutes in the Go scan is 43200. An offset larger than that
    // would be stored and then silently never fire.
    expect(parseOffsetInput('43200')).toBe(43200)
    expect(parseOffsetInput('43201')).toBeNull()
  })
})

describe('formatOffset', () => {
  it('uses the largest whole unit that fits', () => {
    expect(formatOffset(43200)).toEqual({ value: 1, unit: 'month' })
    expect(formatOffset(10080)).toEqual({ value: 1, unit: 'week' })
    expect(formatOffset(1440)).toEqual({ value: 1, unit: 'day' })
    expect(formatOffset(60)).toEqual({ value: 1, unit: 'hour' })
    expect(formatOffset(30)).toEqual({ value: 30, unit: 'minute' })
  })

  it('falls back to a smaller unit when the division is not whole', () => {
    expect(formatOffset(90)).toEqual({ value: 90, unit: 'minute' })
    expect(formatOffset(4320)).toEqual({ value: 3, unit: 'day' })
  })

  it('has a distinct shape for zero', () => {
    expect(formatOffset(0)).toEqual({ value: 0, unit: 'atStart' })
  })
})

describe('matchPreset', () => {
  it('names a list that equals a preset', () => {
    expect(matchPreset([1440, 60], DEFAULT_REMINDER_PRESETS)).toBe('обычное')
  })

  it('matches regardless of the order it was given in', () => {
    expect(matchPreset([60, 1440], DEFAULT_REMINDER_PRESETS)).toBe('обычное')
  })

  it('names the empty list', () => {
    expect(matchPreset([], DEFAULT_REMINDER_PRESETS)).toBe('без')
  })

  it('returns null for a custom list, so the UI can say "custom"', () => {
    expect(matchPreset([7], DEFAULT_REMINDER_PRESETS)).toBeNull()
  })

  // 🔴 Pinning the rule that made the preset selector look broken on staging.
  //
  // When several presets hold the same offsets, this returns the FIRST one in
  // key order. The account had {"без": [1440,60], "важное": [1440,60],
  // "обычное": [1440,60]} — all flattened by the preset picker that used to sit
  // inside the presets EDITOR — so the dropdown read "без" whatever was chosen,
  // and choosing "обычное" wrote the identical list and snapped back.
  //
  // The cause is fixed at the source (ReminderOffsets takes showPresetPicker,
  // and the presets editor passes false). This test states what happens when
  // duplicates exist anyway, so nobody has to rediscover it from a screenshot.
  it('names the first preset when several hold the same offsets', () => {
    const duplicated = { 'без': [1440, 60], 'важное': [1440, 60], 'обычное': [1440, 60] }
    expect(matchPreset([1440, 60], duplicated)).toBe('без')
  })

  it('is unambiguous as long as the presets differ', () => {
    // The negative control for the case above: with distinct presets the same
    // list resolves to the one that actually holds it.
    expect(matchPreset([1440, 60], DEFAULT_REMINDER_PRESETS)).toBe('обычное')
    expect(matchPreset([], DEFAULT_REMINDER_PRESETS)).toBe('без')
  })
})

describe('resolveReminderSettings', () => {
  it('falls back per field on a partial blob', () => {
    const s = resolveReminderSettings({ reminders: { digest_at: '09:30' } })
    expect(s.digest_at).toBe('09:30')
    expect(s.default_event_preset).toBe('обычное')
    expect(s.presets['важное']).toEqual([43200, 10080, 4320, 1440, 60])
  })

  it('survives garbage without throwing — settings are user-writable', () => {
    // Mirrors the Go ParseSettings contract: never throw, fall back per field.
    const s = resolveReminderSettings({ reminders: { digest_at: 12345, presets: 'nope' } })
    expect(s.digest_at).toBe('08:00')
    expect(s.presets).toEqual(DEFAULT_REMINDER_PRESETS)
  })

  it('handles undefined and null', () => {
    expect(resolveReminderSettings(undefined).digest_at).toBe('08:00')
    expect(resolveReminderSettings(null).digest_enabled).toBe(true)
  })

  it('keeps an explicit false rather than treating it as absent', () => {
    const s = resolveReminderSettings({ reminders: { digest_enabled: false } })
    expect(s.digest_enabled).toBe(false)
  })

  it('rejects a digest time that is not a real clock time', () => {
    expect(resolveReminderSettings({ reminders: { digest_at: '25:00' } }).digest_at).toBe('08:00')
  })
})
