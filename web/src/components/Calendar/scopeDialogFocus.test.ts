import { describe, it, expect } from 'vitest'

/**
 * The scope dialog must not default to the answer the server refuses.
 *
 * When a save moves an event to another calendar, `scope=occurrence` comes back
 * 400 CALENDAR_SCOPE_SERIES — deliberately. The dialog nevertheless carried a
 * bare `autoFocus` on "Только это событие", so pressing Enter chose the refused
 * path. The user gets an error message for an answer the dialog picked.
 *
 * planMutation's unit tests cover the case where a REMEMBERED answer would have
 * skipped the dialog. They say nothing about which button is focused once the
 * dialog is on screen — that lives in JSX, and this project has no renderer in
 * its test dependencies, so it is source-scanned like its siblings here.
 */
describe('the recurring-scope dialog focuses an answer the server will accept', () => {
  const sources = import.meta.glob('./RecurringScopeDialog.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const source = Object.values(sources)[0]

  it('found the dialog to check', () => {
    expect(source, 'RecurringScopeDialog.tsx was not found — this test guards nothing').toBeTruthy()
    expect(source).toContain('recurringScope.thisEvent')
  })

  it('never focuses unconditionally', () => {
    // A bare `autoFocus` attribute — the exact shape of the defect. Both
    // buttons carry a conditional one now, so any bare occurrence is a
    // regression regardless of which button grew it.
    expect(source).not.toMatch(/\n\s*autoFocus\s*\n/)
  })

  it('hands the focus to "all events" when a calendar move is in play', () => {
    expect(source).toContain('autoFocus={!calendarChanged}')
    expect(source).toContain('autoFocus={calendarChanged}')
  })

  it('does not merely move the focus — it takes the refused answer away', () => {
    // Focus alone would still leave the refused button one tap away, with
    // nothing saying why it fails.
    expect(source).toContain('disabled={calendarChanged}')
    expect(source).toContain('recurringScope.calendarSeriesOnly')
  })
})
