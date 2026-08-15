import { test, expect } from './fixtures/auth'
import type { Page } from '@playwright/test'

/**
 * Nothing runs off the right edge of a 375px screen.
 *
 * 🔴 Why a spec and not a look. The site audit on 12.08 was done by eye, and
 * the two defects it found at 375px came back within three days because
 * nothing was watching: the colour controls added on 15.08 pushed a
 * calendar's rename and delete buttons off-screen, and that was caught by
 * accident. An eyeball pass proves the screen was right once; this proves it
 * on every deploy.
 *
 * The measurement is `scrollWidth - clientWidth` on the document element, not
 * `getBoundingClientRect` on suspects. Two reasons, both learned the hard way:
 * a rect is layout, and an element can be laid out inside the viewport while
 * a descendant paints outside it; and enumerating suspects only finds the
 * overflow you already thought of. The document either scrolls sideways or it
 * does not, and no element can hide from that.
 *
 * `window.innerWidth` is deliberately NOT the comparison — it is the viewport
 * and stays 375 whatever overflows, so `scrollWidth > innerWidth` would be
 * true for the same reason `scrollWidth > clientWidth` is, but only by luck.
 * clientWidth says what the reader can see, which is the actual claim.
 */

const VIEWPORT_WIDTH = 375

test.describe('375px layout', () => {
  // Via beforeEach, not `test.skip(callback)` at describe level: that form's
  // callback receives the fixtures object ONLY, so reading testInfo.project
  // from it threw "Cannot read properties of undefined" on every desktop test
  // — the spec's own first CI run. beforeEach is passed (fixtures, testInfo).
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'mobile viewport only')
  })

  /** How many pixels the page can be scrolled sideways. Zero is the contract. */
  async function horizontalOverflow(page: Page): Promise<number> {
    return page.evaluate(() => {
      const el = document.documentElement
      return el.scrollWidth - el.clientWidth
    })
  }

  /** Every route reachable without picking something first. */
  const ROUTES = [
    '/home',
    '/calendar',
    '/tasks',
    '/planning',
    '/reflections',
    '/tools',
    '/tools/pomodoro',
    '/tools/kanban',
    '/tools/eisenhower',
    '/tools/time-blocking',
    '/settings',
    '/profile',
  ]

  for (const route of ROUTES) {
    test(`${route} does not scroll sideways`, async ({ authedPage }) => {
      await authedPage.goto(route)
      await authedPage.waitForLoadState('networkidle')
      expect(await horizontalOverflow(authedPage), `${route} overflows its 375px viewport`).toBe(0)
    })
  }

  test('the calendar filter panel stays on screen, open and expanded', async ({ authedPage }) => {
    await authedPage.goto('/calendar')
    await authedPage.waitForLoadState('networkidle')

    const toggle = authedPage.getByTestId('calendar-filter-toggle')
    await expect(toggle).toBeVisible()
    await toggle.click()

    const panel = authedPage.getByTestId('calendar-filter-panel')
    await expect(panel).toBeVisible()

    // The panel is absolutely positioned, so it can sit outside the viewport
    // without the document scrolling — the one case the overflow check above
    // cannot see, which is why it is asserted separately.
    const box = await panel.boundingBox()
    const anchor = await toggle.boundingBox()
    expect(box, 'the filter panel has no box').not.toBeNull()
    // The anchor is reported alongside the panel because the first two
    // failures looked identical (-13.0 both times) and only the button's
    // position distinguished "the panel is too wide" from "the panel is
    // anchored too far left". A number that does not move between runs is
    // saying the change missed.
    const where = `panel x=${box!.x} w=${box!.width}, button right=${anchor ? anchor.x + anchor.width : 'n/a'}`
    expect(box!.x, `the filter panel starts left of the screen — ${where}`).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width, 'the filter panel runs past the right edge').toBeLessThanOrEqual(
      VIEWPORT_WIDTH,
    )

    // Expanding management is where the width actually gets tested: it mounts
    // the full calendars section, with a colour field and rename/delete rows.
    await authedPage.getByRole('button', { name: /manage calendars/i }).click()
    await expect(authedPage.getByTestId('calendars-section')).toBeVisible()
    expect(await horizontalOverflow(authedPage), 'managing calendars overflows the screen').toBe(0)
  })

  test('every reminder preset keeps its delete button on screen', async ({ authedPage }) => {
    await authedPage.goto('/settings')
    await authedPage.waitForLoadState('networkidle')

    const deletes = authedPage.getByRole('button', { name: /delete preset/i })
    // A count of zero would make the loop below pass while asserting nothing —
    // the shape of a control that cannot fail.
    expect(await deletes.count(), 'no preset rows rendered').toBeGreaterThan(0)

    for (let i = 0; i < (await deletes.count()); i++) {
      const box = await deletes.nth(i).boundingBox()
      expect(box, `preset ${i} delete button has no box`).not.toBeNull()
      expect(box!.x + box!.width, `preset ${i} delete button is off-screen`).toBeLessThanOrEqual(
        VIEWPORT_WIDTH,
      )
    }
  })
})
