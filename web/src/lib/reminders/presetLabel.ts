import { DEFAULT_REMINDER_PRESETS } from './offsets'

/**
 * How a preset's name is SHOWN. Its stored name is left alone.
 *
 * 🔴 The three built-in presets are Russian words — `важное`, `обычное`, `без` —
 * and they are KEYS, not labels. `default_event_preset` and
 * `default_task_preset` reference them by string, and a reference that no
 * longer resolves produces `reminder_offsets = {}`, which the reminder scanner
 * skips outright (`cardinality(reminder_offsets) > 0`). The reminder then never
 * arrives and nothing reports it. Renaming them in code would orphan every
 * existing user's reference at once; migrating the data would do the same to
 * anyone whose row the migration missed.
 *
 * So nothing is renamed. Only the display is translated, and only for names
 * that are still exactly the built-in ones. A preset the user renamed — or
 * created — is their text and is shown verbatim: translating it would be
 * inventing a name they never chose.
 *
 * ⚠ Deliberately NOT used by the presets editor. There the field IS the key
 * being edited, and showing a translated value would mean a focus-and-blur
 * silently renaming `важное` to `Important`.
 */

/** Translation keys, by stored preset name. */
const BUILT_IN_LABELS: Record<string, string> = {
  'важное': 'presetName.important',
  'обычное': 'presetName.normal',
  'без': 'presetName.none',
}

/**
 * @param name  the stored preset name
 * @param t     a translator for the `reminders` namespace
 */
export function presetLabel(name: string, t: (key: string) => string): string {
  const key = BUILT_IN_LABELS[name]
  if (!key) return name
  const translated = t(key)
  // i18next returns the key itself when a translation is missing. Showing
  // "presetName.important" to a user would be worse than showing the Russian.
  return translated === key ? name : translated
}

/** True when the name is one of the three shipped presets. */
export function isBuiltInPreset(name: string): boolean {
  return Object.prototype.hasOwnProperty.call(BUILT_IN_LABELS, name)
}

// A typo in BUILT_IN_LABELS would silently stop translating one preset while
// looking correct. This keeps the two lists tied together.
export const BUILT_IN_PRESET_NAMES = Object.keys(DEFAULT_REMINDER_PRESETS)
