import { describe, it, expect } from 'vitest'

/**
 * The swipe decision is only as good as what the grid records.
 *
 * `isHorizontalSwipe` is unit-tested next to itself, and every one of those
 * assertions would keep passing with the caller still storing `clientX` alone —
 * the function would simply be handed `dy = 0` forever and agree with the old,
 * broken rule. A test that cannot fail for the defect it was written against is
 * not a control, so this checks the wiring rather than the arithmetic.
 *
 * Source-scanned because the grid is a large hook-using component and this
 * project has no renderer in its test dependencies. The same idiom as
 * pages/Tasks/calendarPickerScope.test.ts, for the same reason.
 */
describe('the week grid feeds the swipe decision both coordinates', () => {
  const sources = import.meta.glob('./WeekGrid.tsx', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>

  const source = Object.values(sources)[0]

  it('found the grid to check', () => {
    // The floor: a rename would otherwise leave this scanning an empty string
    // and reporting success.
    expect(source, 'WeekGrid.tsx was not found — this test guards nothing').toBeTruthy()
    expect(source).toContain('handleTouchStartSwipe')
  })

  it('records clientY on touchstart, not clientX alone', () => {
    expect(source).toMatch(/handleTouchStartSwipe[\s\S]{0,400}clientY/)
  })

  it('decides with isHorizontalSwipe rather than an inline threshold', () => {
    expect(source).toContain('isHorizontalSwipe(dx, dy)')
    // The rule that let a scroll change the day. If it comes back — anywhere in
    // this file — the swipe is deciding on distance alone again.
    expect(source).not.toMatch(/Math\.abs\(dx\)\s*>\s*50/)
  })
})
