import { describe, it, expect } from 'vitest'
import { resolveQuickTaskSettings, QUICK_TASK_DEFAULTS } from './settings'

describe('resolveQuickTaskSettings', () => {
  it('returns the documented defaults when nothing is stored', () => {
    expect(resolveQuickTaskSettings(undefined)).toEqual(QUICK_TASK_DEFAULTS)
    expect(resolveQuickTaskSettings(null)).toEqual(QUICK_TASK_DEFAULTS)
    expect(resolveQuickTaskSettings({})).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('defaults to tomorrow, priority 3, 15 minutes, filters off', () => {
    expect(QUICK_TASK_DEFAULTS.default_due).toBe('tomorrow')
    expect(QUICK_TASK_DEFAULTS.default_priority).toBe(3)
    expect(QUICK_TASK_DEFAULTS.default_estimate_minutes).toBe(15)
    expect(QUICK_TASK_DEFAULTS.inherit_filters).toBe(false)
  })

  it('accepts valid stored values', () => {
    const r = resolveQuickTaskSettings({
      quick_task: {
        default_due: 'today',
        default_priority: 1,
        default_estimate_minutes: 5,
        inherit_filters: true,
      },
    })
    expect(r.default_due).toBe('today')
    expect(r.default_priority).toBe(1)
    expect(r.default_estimate_minutes).toBe(5)
    expect(r.inherit_filters).toBe(true)
  })

  it('accepts null estimate as "do not set an estimate"', () => {
    const r = resolveQuickTaskSettings({ quick_task: { default_estimate_minutes: null } })
    expect(r.default_estimate_minutes).toBeNull()
  })

  it('falls back per-field on garbage without throwing', () => {
    const r = resolveQuickTaskSettings({
      // The JSONB blob is user-writable; a bad field must not poison the others.
      quick_task: {
        default_due: 'yesterday',
        default_priority: 99,
        default_estimate_minutes: -5,
        inherit_filters: 'yes',
      },
    } as never)
    expect(r).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('rejects a non-object quick_task', () => {
    expect(resolveQuickTaskSettings({ quick_task: 'nope' } as never)).toEqual(QUICK_TASK_DEFAULTS)
  })

  it('merges stored keybindings over the defaults', () => {
    const r = resolveQuickTaskSettings({ quick_task: { keys: { expand: 'Ctrl+D' } } } as never)
    expect(r.keys.expand).toBe('Ctrl+D')
    expect(r.keys.submit).toBe(QUICK_TASK_DEFAULTS.keys.submit)
    expect(r.keys.indent).toBe('Alt+ArrowRight')
  })
})
