import type { UserSettings } from '../../api/auth'

export type DefaultDue = 'today' | 'tomorrow' | 'none'

export interface QuickTaskKeys {
  submit: string
  submit_expanded: string
  expand: string
  global_capture: string
  indent: string
  outdent: string
}

export interface QuickTaskSettings {
  default_due: DefaultDue
  default_priority: number
  /** null means "do not set an estimate". */
  default_estimate_minutes: number | null
  inherit_filters: boolean
  keys: QuickTaskKeys
}

export const QUICK_TASK_DEFAULTS: QuickTaskSettings = {
  default_due: 'tomorrow',
  default_priority: 3,
  default_estimate_minutes: 15,
  inherit_filters: false,
  keys: {
    submit: 'Enter',
    submit_expanded: 'Ctrl+Enter',
    expand: 'Ctrl+E',
    global_capture: 'Ctrl+K',
    // Not Tab: Tab is the browser's focus key, and capturing it would trap
    // keyboard users inside the input (WCAG 2.1.2).
    indent: 'Alt+ArrowRight',
    outdent: 'Alt+ArrowLeft',
  },
}

const DUE_VALUES: readonly DefaultDue[] = ['today', 'tomorrow', 'none']

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/**
 * Read quick-task preferences out of the user settings JSONB blob.
 *
 * The blob is user-writable and may hold anything, so every field falls back
 * independently and nothing here may throw: one bad value must not cost the
 * user their other preferences.
 */
export function resolveQuickTaskSettings(settings: UserSettings | null | undefined): QuickTaskSettings {
  const raw: unknown = settings?.quick_task
  if (!isRecord(raw)) return QUICK_TASK_DEFAULTS

  const due = raw.default_due
  const priority = raw.default_priority
  const estimate = raw.default_estimate_minutes
  const inherit = raw.inherit_filters
  const keys = isRecord(raw.keys) ? raw.keys : {}

  const validKeys: QuickTaskKeys = { ...QUICK_TASK_DEFAULTS.keys }
  for (const name of Object.keys(QUICK_TASK_DEFAULTS.keys) as (keyof QuickTaskKeys)[]) {
    const bound = keys[name]
    if (typeof bound === 'string' && bound.trim() !== '') validKeys[name] = bound
  }

  return {
    default_due: DUE_VALUES.includes(due as DefaultDue) ? (due as DefaultDue) : QUICK_TASK_DEFAULTS.default_due,
    default_priority:
      typeof priority === 'number' && Number.isInteger(priority) && priority >= 0 && priority <= 5
        ? priority
        : QUICK_TASK_DEFAULTS.default_priority,
    default_estimate_minutes:
      estimate === null
        ? null
        : typeof estimate === 'number' && Number.isInteger(estimate) && estimate > 0 && estimate <= 1440
          ? estimate
          : QUICK_TASK_DEFAULTS.default_estimate_minutes,
    inherit_filters: typeof inherit === 'boolean' ? inherit : QUICK_TASK_DEFAULTS.inherit_filters,
    keys: validKeys,
  }
}
