import { describe, it, expect } from 'vitest'

/**
 * Tapping a task in the calendar must DO something.
 *
 * `onSelectTask` and `onEditTask` were `console.log` stubs wired into both the
 * desktop sidebar and the mobile panel, and they were the only two left in
 * `src/pages/`. From the outside a stub is indistinguishable from a slow
 * network — Denis tapped, nothing happened, and there was nothing to report
 * beyond "не работает".
 *
 * The deep-link format has its own unit tests; those would all pass with these
 * handlers still logging, because nothing would ever build a link. This checks
 * that the handlers are wired to it — the half a pure test cannot see.
 *
 * Source-scanned for the same reason as the sibling scans in this repo: these
 * are large hook-using components and there is no renderer in the test
 * dependencies.
 */
describe('the calendar does something when a task is tapped', () => {
  const sources = import.meta.glob('./Calendar.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const source = Object.values(sources)[0]

  it('found the page to check', () => {
    expect(source, 'Calendar.tsx was not found — this test guards nothing').toBeTruthy()
    expect(source).toContain('handleSelectTask')
    expect(source).toContain('handleEditTask')
  })

  it('has no console.log stub anywhere on the page', () => {
    // Deliberately the whole file, not just these two handlers: the next stub
    // will be written somewhere else, and a placeholder that logs is the shape
    // every one of them takes.
    expect(source).not.toMatch(/console\.log\(/)
  })

  it('sends both handlers to the tasks page', () => {
    expect(source).toMatch(/handleSelectTask[\s\S]{0,300}taskDeepLinkTo\(task\.id, false\)/)
    expect(source).toMatch(/handleEditTask[\s\S]{0,300}taskDeepLinkTo\(task\.id, true\)/)
  })
})
