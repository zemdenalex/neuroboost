/**
 * Reminder offsets are minutes before an event's start or a task's due date.
 * The backend stores them as an INTEGER[] on the row itself rather than as
 * pre-generated reminder rows, so this is the only representation the UI needs.
 */

export const MINUTE = 1
export const HOUR = 60
export const DAY = 1440
export const WEEK = 10080
export const MONTH = 43200

/**
 * The Go scan looks ahead by at most MONTH minutes (`maxOffsetMinutes` in
 * api-go/internal/reminders/scan.go). An offset beyond that would be stored
 * happily and then never fire, which is worse than refusing it here.
 */
export const MAX_OFFSET_MINUTES = MONTH

export type ReminderPresets = Record<string, number[]>

/** Mirrors DefaultSettings() in api-go/internal/reminders/settings.go. */
export const DEFAULT_REMINDER_PRESETS: ReminderPresets = {
  'важное': [MONTH, WEEK, 3 * DAY, DAY, HOUR],
  'обычное': [DAY, HOUR],
  'без': [],
}

export interface ReminderSettings {
  presets: ReminderPresets
  default_event_preset: string
  default_task_preset: string
  digest_at: string
  digest_enabled: boolean
  quiet_hours_respected: boolean
}

export const DEFAULT_REMINDER_SETTINGS: ReminderSettings = {
  presets: DEFAULT_REMINDER_PRESETS,
  default_event_preset: 'обычное',
  default_task_preset: 'обычное',
  digest_at: '08:00',
  digest_enabled: true,
  quiet_hours_respected: true,
}

/**
 * Sorted furthest-out first, deduplicated, and stripped of anything that is not
 * a usable offset.
 *
 * Negative values are refused on purpose: -1 and -2 are sentinels in the
 * reminder journal (snooze and digest), so a negative offset arriving from the
 * UI would collide with a real reminder's dedupe key.
 */
export function normalizeOffsets(offsets: number[]): number[] {
  const seen = new Set<number>()
  for (const raw of offsets) {
    if (!Number.isInteger(raw)) continue
    if (raw < 0 || raw > MAX_OFFSET_MINUTES) continue
    seen.add(raw)
  }
  return [...seen].sort((a, b) => b - a)
}

export function addOffset(offsets: number[], value: number): number[] {
  return normalizeOffsets([...offsets, value])
}

export function removeOffset(offsets: number[], value: number): number[] {
  return normalizeOffsets(offsets.filter(o => o !== value))
}

const UNIT_MINUTES: Record<string, number> = {
  m: MINUTE, м: MINUTE,
  h: HOUR, ч: HOUR,
  d: DAY, д: DAY,
  w: WEEK, н: WEEK,
}

/**
 * Reads "30", "2h", "3д", "1w" as a number of minutes. Returns null rather
 * than a guess: a misread offset is a reminder at the wrong time, which is
 * harder to notice than no reminder at all.
 */
export function parseOffsetInput(input: string): number | null {
  const cleaned = input.trim().toLowerCase().replace(/\s+/g, '')
  if (!cleaned) return null

  const match = /^(\d+)([a-zа-я]?)$/.exec(cleaned)
  if (!match) return null

  const amount = Number.parseInt(match[1], 10)
  if (!Number.isFinite(amount)) return null

  const suffix = match[2]
  const unit = suffix ? UNIT_MINUTES[suffix] : MINUTE
  if (unit === undefined) return null

  const minutes = amount * unit
  if (minutes > MAX_OFFSET_MINUTES) return null
  return minutes
}

export type OffsetUnit = 'atStart' | 'minute' | 'hour' | 'day' | 'week' | 'month'

export interface FormattedOffset {
  value: number
  unit: OffsetUnit
}

/**
 * Picks the largest unit that divides the offset evenly, so 1440 reads as
 * "1 day" and 90 stays "90 minutes" instead of becoming "1.5 hours".
 *
 * Returns a value+unit pair rather than a string: the caller owns
 * pluralisation, which differs between en and ru.
 */
export function formatOffset(minutes: number): FormattedOffset {
  if (minutes === 0) return { value: 0, unit: 'atStart' }
  const scale: Array<[number, OffsetUnit]> = [
    [MONTH, 'month'],
    [WEEK, 'week'],
    [DAY, 'day'],
    [HOUR, 'hour'],
  ]
  for (const [size, unit] of scale) {
    if (minutes % size === 0) return { value: minutes / size, unit }
  }
  return { value: minutes, unit: 'minute' }
}

/**
 * Names the preset a list corresponds to, or null when it is custom.
 * Order-insensitive, because a list built by clicking is not sorted the same
 * way as one typed into settings.
 */
export function matchPreset(offsets: number[], presets: ReminderPresets): string | null {
  const mine = normalizeOffsets(offsets).join(',')
  for (const [name, preset] of Object.entries(presets)) {
    if (normalizeOffsets(preset).join(',') === mine) return name
  }
  return null
}

function isPositiveIntArray(value: unknown): value is number[] {
  return Array.isArray(value) && value.every(v => Number.isInteger(v) && v >= 0)
}

function isClockTime(value: unknown): value is string {
  return typeof value === 'string' && /^([01]\d|2[0-3]):[0-5]\d$/.test(value)
}

/**
 * Reads the `reminders` section out of the user's settings blob.
 *
 * Never throws and falls back per field, matching the Go ParseSettings
 * contract: the blob is user-writable, and one bad digest_at must not cost the
 * user their presets.
 */
export function resolveReminderSettings(settings: unknown): ReminderSettings {
  const resolved: ReminderSettings = {
    ...DEFAULT_REMINDER_SETTINGS,
    presets: { ...DEFAULT_REMINDER_PRESETS },
  }
  if (!settings || typeof settings !== 'object') return resolved

  const section = (settings as Record<string, unknown>).reminders
  if (!section || typeof section !== 'object') return resolved
  const r = section as Record<string, unknown>

  if (r.presets && typeof r.presets === 'object' && !Array.isArray(r.presets)) {
    const presets: ReminderPresets = {}
    for (const [name, value] of Object.entries(r.presets as Record<string, unknown>)) {
      if (isPositiveIntArray(value)) presets[name] = normalizeOffsets(value)
    }
    if (Object.keys(presets).length > 0) resolved.presets = presets
  }
  if (typeof r.default_event_preset === 'string') resolved.default_event_preset = r.default_event_preset
  if (typeof r.default_task_preset === 'string') resolved.default_task_preset = r.default_task_preset
  if (isClockTime(r.digest_at)) resolved.digest_at = r.digest_at
  if (typeof r.digest_enabled === 'boolean') resolved.digest_enabled = r.digest_enabled
  if (typeof r.quiet_hours_respected === 'boolean') resolved.quiet_hours_respected = r.quiet_hours_respected

  return resolved
}
