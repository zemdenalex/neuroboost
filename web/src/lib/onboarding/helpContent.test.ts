import { describe, it, expect } from 'vitest'
import { resolveHelpKey } from './helpContent'

describe('resolveHelpKey', () => {
  it('maps a known top-level route to its help key', () => {
    expect(resolveHelpKey('/tasks')).toBe('help.tasks')
    expect(resolveHelpKey('/calendar')).toBe('help.calendar')
  })

  it('ignores trailing segments and query/sub-paths', () => {
    expect(resolveHelpKey('/tasks/123')).toBe('help.tasks')
    expect(resolveHelpKey('/calendar/2026-06-17')).toBe('help.calendar')
  })

  it('falls back to the default key for the root path', () => {
    expect(resolveHelpKey('/')).toBe('help.default')
    expect(resolveHelpKey('')).toBe('help.default')
  })

  it('falls back to the default key for unknown routes', () => {
    expect(resolveHelpKey('/nope')).toBe('help.default')
  })
})
