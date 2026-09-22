import { test, expect } from './fixtures/auth'

/**
 * The collapsed task sidebar is a tab at the left edge. It used to be
 * `position: fixed` over the page, so on desktop it sat on top of the first
 * day column — Monday — around 04:00–05:30, and a real user could not grab an
 * event there: the press landed on the tab. Found 22.09 when e2e drags on a
 * Monday grabbed «Tasks (0)» instead of the resize handle.
 *
 * Asserted on geometry, not on a class name: the defect is two boxes sharing
 * pixels, and any way of reintroducing it — fixed, absolute, a negative margin —
 * shows up here the same way.
 */
test('the collapsed task tab takes its own room instead of covering Monday', async ({ authedPage }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'the sidebar is desktop-only (hidden below lg)')

  await authedPage.addInitScript(() => localStorage.setItem('nb-sidebar-open', 'false'))
  await authedPage.goto('/calendar')

  const tab = authedPage.getByTestId('task-sidebar-tab')
  const grid = authedPage.getByTestId('calendar-main')
  await expect(tab).toBeVisible()
  await expect(grid).toBeVisible()

  const t = await tab.boundingBox()
  const g = await grid.boundingBox()
  expect(t, 'tab has no box').not.toBeNull()
  expect(g, 'calendar has no box').not.toBeNull()
  // Positive control: both are real, non-empty boxes on screen, so «no
  // overlap» below cannot be two zero-size boxes agreeing.
  expect(t!.width).toBeGreaterThan(0)
  expect(g!.width).toBeGreaterThan(600)

  expect(t!.x + t!.width, `tab ends at ${t!.x + t!.width}px, calendar starts at ${g!.x}px`).toBeLessThanOrEqual(g!.x + 0.5)

  // And it still opens the sidebar.
  await tab.click()
  await expect(tab).toHaveCount(0)
})
