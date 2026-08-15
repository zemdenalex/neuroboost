/**
 * Adding, renaming and deleting reminder presets.
 *
 * Kept out of the React section on purpose: every operation here has a branch
 * that is invisible in the UI and expensive in production, so each one needs a
 * test that can redden.
 *
 * 🔴 The branch that matters most: `default_event_preset` and
 * `default_task_preset` are plain strings naming a key in `presets`. Nothing
 * enforces that the key exists — not `resolveReminderSettings` here, which
 * checks only `typeof === 'string'`, and not `ParseReminders` in Go. A dangling
 * name reaches `usersettings.OffsetsForPreset`, which answers `[]int{}` for a
 * name it does not know, so a newly created event is stored with
 * `reminder_offsets = {}` — and the scanner skips those rows outright
 * (`cardinality(reminder_offsets) > 0`, reminders/scan.go). The event then
 * never reminds and says nothing about it.
 *
 * Therefore: renaming a preset repoints any default that named it, and
 * deleting one repoints those defaults to a survivor.
 */

import { normalizeOffsets, type ReminderPresets, type ReminderSettings } from './offsets'

export type PresetError =
  /** The name was blank or only whitespace. */
  | 'empty'
  /** Another preset already holds that name. */
  | 'duplicate'
  /** No preset by that name exists. */
  | 'unknown'
  /** Refusing to delete the last one — see deletePreset. */
  | 'last'

export type PresetResult =
  | { ok: true; settings: ReminderSettings }
  | { ok: false; reason: PresetError }

/**
 * Why a name is refused, or null when it is usable.
 *
 * `current` is the name being renamed, if any: renaming a preset to what it is
 * already called is not a collision with itself.
 *
 * Duplicate names are REFUSED rather than warned about, because `presets` is a
 * `Record<string, number[]>` — writing a name that already exists does not
 * create a second preset, it silently overwrites the first one's offsets.
 */
export function validatePresetName(
  raw: string,
  presets: ReminderPresets,
  current?: string,
): PresetError | null {
  const name = raw.trim()
  if (name === '') return 'empty'
  if (name !== current && Object.prototype.hasOwnProperty.call(presets, name)) return 'duplicate'
  return null
}

/** A new, empty preset appended after the existing ones. */
export function addPreset(settings: ReminderSettings, raw: string): PresetResult {
  const reason = validatePresetName(raw, settings.presets)
  if (reason) return { ok: false, reason }

  return {
    ok: true,
    settings: { ...settings, presets: { ...settings.presets, [raw.trim()]: [] } },
  }
}

/**
 * Rename in place, keeping the preset where it sits in the list.
 *
 * Rebuilding the object key by key rather than deleting and re-adding: object
 * key order is insertion order, so the shortcut would drop the renamed preset
 * to the bottom of the list while the user is still looking at it.
 */
export function renamePreset(settings: ReminderSettings, from: string, raw: string): PresetResult {
  if (!Object.prototype.hasOwnProperty.call(settings.presets, from)) {
    return { ok: false, reason: 'unknown' }
  }
  const reason = validatePresetName(raw, settings.presets, from)
  if (reason) return { ok: false, reason }

  const to = raw.trim()
  if (to === from) return { ok: true, settings }

  const presets: ReminderPresets = {}
  for (const [name, offsets] of Object.entries(settings.presets)) {
    presets[name === from ? to : name] = offsets
  }

  return {
    ok: true,
    settings: {
      ...settings,
      presets,
      default_event_preset: settings.default_event_preset === from ? to : settings.default_event_preset,
      default_task_preset: settings.default_task_preset === from ? to : settings.default_task_preset,
    },
  }
}

/**
 * Remove a preset, moving any default that named it onto a survivor.
 *
 * 🔴 The last preset cannot be deleted, and the reason is mechanical rather
 * than stylistic: both readers treat an empty map as "absent" and substitute
 * the three built-in presets — `resolveReminderSettings` assigns only when
 * `Object.keys(presets).length > 0`, and Go's `applyLooseReminders` only when
 * `len(presets) > 0`. Saving `{}` would therefore look like a reset to
 * defaults rather than a deletion, which is a confusing thing to hand someone
 * who just deleted their last preset.
 */
export function deletePreset(settings: ReminderSettings, name: string): PresetResult {
  if (!Object.prototype.hasOwnProperty.call(settings.presets, name)) {
    return { ok: false, reason: 'unknown' }
  }
  const remaining = Object.keys(settings.presets).filter(n => n !== name)
  if (remaining.length === 0) return { ok: false, reason: 'last' }

  const presets: ReminderPresets = {}
  for (const key of remaining) presets[key] = settings.presets[key]

  const survivor = remaining[0]
  return {
    ok: true,
    settings: {
      ...settings,
      presets,
      default_event_preset: settings.default_event_preset === name ? survivor : settings.default_event_preset,
      default_task_preset: settings.default_task_preset === name ? survivor : settings.default_task_preset,
    },
  }
}

/**
 * Groups of presets that hold the same offsets under different names.
 *
 * Not an error — two names for one schedule is a legitimate thing to want, and
 * refusing it would be removing something that works today. But it makes
 * `matchPreset` ambiguous: it answers with the FIRST name holding those
 * offsets, so choosing the second one leaves the dropdown reading the first,
 * which looks like the choice did not take. That was the bouncing selector
 * reported on staging on 14.08, and the honest fix is to say so beside the
 * rows rather than to forbid it.
 */
export function duplicateOffsetGroups(presets: ReminderPresets): string[][] {
  const byOffsets = new Map<string, string[]>()
  for (const [name, offsets] of Object.entries(presets)) {
    const key = normalizeOffsets(offsets).join(',')
    const group = byOffsets.get(key)
    if (group) group.push(name)
    else byOffsets.set(key, [name])
  }
  return [...byOffsets.values()].filter(group => group.length > 1)
}
