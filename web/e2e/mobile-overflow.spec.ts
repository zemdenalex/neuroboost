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

    // Expanding management is where the width actually gets tested: the rows
    // grow a colour select, a rename button and a delete button, and a create
    // field appears underneath.
    await authedPage.getByTestId('calendar-filter-manage').click()
    await expect(authedPage.getByTestId('calendar-create-input')).toBeVisible()
    expect(await horizontalOverflow(authedPage), 'managing calendars overflows the screen').toBe(0)

    // 🔴 The document check above CANNOT see this, and that is why Denis found
    // it and the suite did not. `horizontalOverflow` reads the document's
    // scrollWidth; the panel has `overflow-y-auto`, so its own overflow is
    // absorbed into a scrollbar inside the panel and the document stays calm.
    // What he saw on 16.08 was exactly that: a horizontal scrollbar inside the
    // panel and a create button clipped at its right edge, with the page
    // itself perfectly still. Measure the panel against ITSELF.
    const inner = await panel.evaluate((el) => ({
      scrollWidth: el.scrollWidth,
      clientWidth: el.clientWidth,
    }))
    expect(
      inner.scrollWidth - inner.clientWidth,
      `the panel's own content is wider than the panel (${inner.scrollWidth} > ${inner.clientWidth})`,
    ).toBeLessThanOrEqual(1)

    // And every control inside it must be reachable, not merely non-scrolling:
    // a button clipped by the panel's right edge still "exists" to a locator.
    const panelBox = await panel.boundingBox()
    for (const testid of ['calendar-create-submit', 'calendar-create-input']) {
      const box = await authedPage.getByTestId(testid).boundingBox()
      expect(box, `${testid} has no box`).not.toBeNull()
      expect(
        box!.x + box!.width,
        `${testid} is clipped by the panel's right edge`,
      ).toBeLessThanOrEqual(panelBox!.x + panelBox!.width + 1)
    }
  })

  /**
   * 🔴 The route sweep above passes or fails on WHOSE data it is looking at.
   *
   * /profile overflowed at 375px for months and this suite was green the whole
   * time: the CI account's email is short enough to fit, and the defect only
   * appears when a value has no break opportunity and no room. Running the same
   * spec against a real account (zemdenalex@gmail.com) failed it on the first
   * try — the address, the XP figures and the progress bar all ran off the
   * right edge. Same code, same assertion, different fixture data.
   *
   * So the length is supplied here rather than hoped for. The text is replaced
   * in the DOM, not on the server: this is a question about CSS — does the
   * layout shrink an unbreakable value or let it push — and mutating a shared
   * staging account to ask it would be both slower and worse.
   */
  test('the profile header survives a long email, whoever is logged in', async ({ authedPage }) => {
    await authedPage.goto('/profile')
    await authedPage.waitForLoadState('networkidle')

    const header = authedPage.locator('[data-hint="profile.identity"]')
    await expect(header).toBeVisible()

    const replaced = await header.evaluate((el) => {
      const target = Array.from(el.querySelectorAll('span, h1')).find((n) =>
        (n.textContent ?? '').includes('@'),
      )
      if (!target) return false
      target.textContent = 'a.very.long.address.that.does.not.break@some-long-domain.example.com'
      return true
    })
    // Without this the test would pass by having found nothing to lengthen.
    expect(replaced, 'no email-shaped text in the profile header to lengthen').toBe(true)

    expect(
      await horizontalOverflow(authedPage),
      'a long email pushes the profile header past the right edge',
    ).toBe(0)
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
