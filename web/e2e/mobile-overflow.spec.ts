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
  test('the profile header survives a long unbreakable value, whoever is logged in', async ({
    authedPage,
  }) => {
    await authedPage.goto('/profile')
    await authedPage.waitForLoadState('networkidle')

    const header = authedPage.locator('[data-hint="profile.identity"]')
    await expect(header).toBeVisible()

    // ⚠ The first version of this test lengthened the EMAIL, and failed in CI
    // with "nothing to lengthen" — the CI account signs in through Telegram and
    // has no email at all. Which is the same lesson one layer down: a test
    // written around the data in front of you inherits that data's shape. The
    // display name is the value every account has.
    const LONG = 'Wolfeschlegelsteinhausenbergerdorffvoralternwarengewissenhaftschaferswessen'
    const lengthened = await header.evaluate((el, long) => {
      // 🔴 LEAF elements only. The first version matched any span containing
      // "@", which included the WRAPPER around the icon and the truncating
      // span — and setting textContent on a wrapper deletes its children. The
      // test dismantled the exact structure it was checking and then reported
      // the page as broken: an overflow of 822px that the app never produces.
      // A control that fails for its own reason is worse than no control.
      const nodes = [
        el.querySelector('h1'),
        ...Array.from(el.querySelectorAll('span')).filter(
          (n) => n.childElementCount === 0 && (n.textContent ?? '').includes('@'),
        ),
      ].filter((n): n is HTMLElement => n !== null && n.childElementCount === 0)
      for (const n of nodes) n.textContent = long
      return nodes.length
    }, LONG)
    // Without this the test could pass by having found nothing to lengthen.
    expect(lengthened, 'nothing in the profile header to lengthen').toBeGreaterThan(0)

    expect(
      await horizontalOverflow(authedPage),
      'a long unbreakable value pushes the profile header past the right edge',
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

  // Tour 25.09 (MW1): the unscheduled list ran on under the capacity meter.
  // Its wrapper capped the height at 40vh but the list inside sized itself by
  // content, so the meter below painted over the lower tasks.
  test('planning: the unscheduled list ends above the capacity meter', async ({ authedPage }) => {
    await authedPage.goto('/planning')
    const list = authedPage.locator('[data-hint="planning.unscheduled"]')
    const meter = authedPage.getByTestId('capacity-meter')
    await expect(meter).toBeVisible()
    const a = await list.boundingBox()
    const b = await meter.boundingBox()
    expect(a && b, 'both boxes measured').toBeTruthy()
    expect(a!.y + a!.height, 'list bottom vs meter top').toBeLessThanOrEqual(b!.y + 1)
  })

  // Tour 25.09 (MW2): beside a 96px avatar the XP line had ~135px and broke
  // into four lines ("Level1" glued together). On a phone it goes under it.
  test('profile: the XP block gets the card width, not a side column', async ({ authedPage }) => {
    await authedPage.goto('/profile')
    const xp = authedPage.getByTestId('profile-xp')
    await expect(xp).toBeVisible()
    const box = await xp.boundingBox()
    expect(box!.width, 'XP block width at 375px').toBeGreaterThanOrEqual(260)
  })

  // Tour 25.09 (MW3): the counters broke mid-label ("To / Do:") and the
  // focused quick-add drew a second outline inside its own focus ring.
  test('tasks: each counter stays on one line, quick-add has one focus ring', async ({ authedPage }) => {
    await authedPage.goto('/tasks')
    const stats = authedPage.getByTestId('task-stats').locator(':scope > span')
    await expect(stats.first()).toBeVisible()
    for (const box of await stats.evaluateAll((els) => els.map((e) => e.getBoundingClientRect().height))) {
      expect(box, 'a counter wrapped onto two lines').toBeLessThan(30)
    }
    const input = authedPage.getByRole('textbox', { name: /new task/i }).first()
    await input.focus()
    await authedPage.keyboard.press('a')
    // Tailwind's outline-none is `2px solid transparent`, so the style alone
    // says nothing: the question is whether a painted outline is visible.
    const outline = await input.evaluate((e) => {
      const cs = getComputedStyle(e)
      return cs.outlineStyle === 'none' || cs.outlineColor === 'rgba(0, 0, 0, 0)' ? 'invisible' : cs.outlineColor
    })
    expect(outline, 'a second focus outline inside the row').toBe('invisible')
  })

  // Tour 25.09 (MW4): "Friday, September 25" broke onto two lines beside the
  // arrows and buttons, pushing the grid down another 30px.
  test('calendar: the date title fits on one line', async ({ authedPage }) => {
    await authedPage.goto('/calendar')
    const title = authedPage.getByTestId('calendar-period-title')
    await expect(title).toBeVisible()
    const box = await title.boundingBox()
    expect(box!.height, 'title wrapped').toBeLessThan(30)
  })

  // Tour 25.09 (MW5): the "vertical sidebar" layout is hidden below md, so a
  // phone with that setting (it syncs from the account, e.g. chosen on a
  // desktop) had no top bar at all: no avatar menu, no help.
  test('a phone keeps its top bar even with the sidebar layout saved', async ({ authedPage }) => {
    // The account's settings are applied over localStorage on load, so the
    // saved variant is substituted in the /auth/me answer; writes are blocked
    // so the real account never gets it (it is Denis's staging account).
    await authedPage.route('**/api/auth/me', (route) =>
      route.request().method() === 'PATCH' ? route.abort() : route.fallback(),
    )
    await authedPage.addInitScript(() => {
      localStorage.setItem('neuroboost-header-variant', 'vertical')
      const orig = window.fetch.bind(window)
      window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
        const res = await orig(input, init)
        if (!url.includes('/api/auth/me') || (init?.method && init.method !== 'GET')) return res
        const body = await res.clone().json()
        body.data.settings = { ...(body.data.settings ?? {}), header_variant: 'vertical' }
        return new Response(JSON.stringify(body), { status: res.status, headers: { 'Content-Type': 'application/json' } })
      }
    })
    await authedPage.goto('/calendar')
    await expect(authedPage.getByTestId('calendar-period-title')).toBeVisible({ timeout: 15_000 })
    await expect(authedPage.locator('header').first(), 'no top bar on the phone').toBeVisible()
    await authedPage.goto('/settings')
    // Wait for the page itself: a count of 0 is true before a lazy page renders.
    await expect(authedPage.getByTestId('settings-hint-section')).toBeVisible({ timeout: 15_000 })
    await expect(authedPage.getByTestId('settings-layout-section')).toHaveCount(0)
  })

  // Tour 25.09 (MW7): the logo sat 40px in from the edge for a hamburger
  // button that only exists when mobile_nav is 'hamburger'.
  test('the logo starts at the edge, and clears the hamburger when there is one', async ({ authedPage }) => {
    await authedPage.goto('/calendar')
    const logo = authedPage.locator('header a', { hasText: 'NeuroBoost' }).first()
    await expect(logo).toBeVisible({ timeout: 15_000 })
    expect((await logo.boundingBox())!.x, 'gap without a hamburger').toBeLessThan(24)

    await withSettings(authedPage, { mobile_nav: 'hamburger' })
    await authedPage.goto('/calendar')
    await expect(authedPage.getByTestId('calendar-period-title')).toBeVisible({ timeout: 15_000 })
    const burger = await authedPage.getByRole('button', { name: /menu/i }).first().boundingBox()
    const moved = await logo.boundingBox()
    expect(moved!.x, 'logo under the hamburger').toBeGreaterThanOrEqual(burger!.x + burger!.width)
  })

  // Tour 25.09 (MW6): page padding 24px plus card padding 20-24px took ~96px
  // of 375. On a phone the page keeps 16px a side.
  for (const [route, selector] of [
    ['/settings', '[data-testid="settings-hint-section"]'],
    ['/tools', 'main a[href="/tools/pomodoro"]'],
    ['/tasks', '[data-testid="task-stats"]'],
  ] as const) {
    test(`${route}: the first block spans the phone, not 24px in`, async ({ authedPage }) => {
      await authedPage.goto(route)
      const el = authedPage.locator(selector).first()
      await expect(el).toBeVisible({ timeout: 15_000 })
      const box = await el.boundingBox()
      expect(box!.x, `${route} left gutter`).toBeLessThanOrEqual(19)
    })
  }

  // Tour 25.09, second pass (MW10): the bulk-select checkbox is opacity-0
  // until hover, and a phone has no hover. It took the row's width for an
  // invisible control that a tap left of the circle silently ticked.
  test('tasks: no invisible select box in a phone row', async ({ authedPage }) => {
    await authedPage.goto('/tasks')
    const row = authedPage.locator('[id^="task-"]').first()
    await expect(row).toBeVisible({ timeout: 15_000 })
    await expect(row.locator('input[type="checkbox"]')).toBeHidden()
  })

  // Tour 25.09, second pass (MW11): the day opened at 00:00, a screen of empty
  // small hours with "00:00" half under the sticky header.
  test('calendar: the day opens near the current hour, not at midnight', async ({ authedPage }) => {
    const hour = Number(
      new Intl.DateTimeFormat('en-GB', { hour: 'numeric', hourCycle: 'h23', timeZone: 'Europe/Moscow' }).format(new Date()),
    )
    test.skip(hour < 2, 'before 02:00 an hour before now is the top anyway')
    await authedPage.goto('/calendar')
    await expect(authedPage.getByTestId('week-day-header').first()).toBeVisible({ timeout: 15_000 })
    const top = await authedPage.evaluate(() => {
      const header = document.querySelector('[data-testid="week-day-header"]')
      let el = header?.parentElement ?? null
      while (el && getComputedStyle(el).overflowY !== 'auto') el = el.parentElement
      return el?.scrollTop ?? -1
    })
    expect(top, 'opened at midnight').toBeGreaterThan(0)
  })

  // MA8b: /calendar?date= (a Mini App start link d-YYYY-MM-DD lands here)
  // opens that very day on a phone, not today or the week's Monday.
  test('calendar: ?date= opens that day on a phone', async ({ authedPage }) => {
    // Two days on, in Moscow terms; the week may roll over, which is the point.
    const target = new Intl.DateTimeFormat('en-CA', { timeZone: 'Europe/Moscow' }).format(new Date(Date.now() + 2 * 864e5))
    await authedPage.goto(`/calendar?date=${target}`)
    const header = authedPage.getByTestId('week-day-header').first()
    await expect(header).toHaveAttribute('data-day', target, { timeout: 15_000 })
    await expect(authedPage).not.toHaveURL(/date=/)
    // Review M4: another day opens at the start of the working day (08:00 = 8 * 44px),
    // not at "now minus an hour", which belongs to today.
    await expect
      .poll(() =>
        authedPage.evaluate(() => {
          let el = document.querySelector('[data-testid="week-day-header"]')?.parentElement ?? null
          while (el && getComputedStyle(el).overflowY !== 'auto') el = el.parentElement
          return el?.scrollTop ?? -1
        }),
      )
      .toBe(8 * 44)
  })

  // Denis 25.09, variant A: on a phone the tip shows on the first three opens,
  // and an empty all-day bar is a 20px strip. A far-future day is empty in any
  // account, so the bar's height does not depend on whose calendar this is.
  test('calendar: thin all-day strip on an empty day, tip only the first three times', async ({ authedPage }) => {
    const shown: boolean[] = []
    for (let i = 0; i < 4; i++) {
      await authedPage.goto('/calendar?date=2031-01-15')
      await expect(authedPage.getByTestId('week-day-header').first()).toHaveAttribute('data-day', '2031-01-15', {
        timeout: 15_000,
      })
      shown.push(await authedPage.getByTestId('calendar-hint').isVisible())
    }
    expect(shown, 'tip on opens 1-4').toEqual([true, true, true, false])
    const bar = authedPage.locator('[data-testid="week-day-header"]').first()
    const top = await bar.evaluate((el) => {
      const header = el.parentElement as HTMLElement
      return parseFloat(getComputedStyle(header).top)
    })
    expect(top, 'day header sits under a 20px all-day strip').toBe(20)
  })

  // Denis 25.09: task actions on a phone, all three variants, chosen in settings.
  test('task row: «⋯» by default, with the actions in its menu', async ({ authedPage }) => {
    await authedPage.goto('/tasks')
    const row = authedPage.locator('[id^="task-"]').first()
    await expect(row).toBeVisible({ timeout: 15_000 })
    await expect(row.getByRole('button', { name: /schedule|запланировать/i })).toHaveCount(0)
    await row.getByTestId('task-row-more').click()
    const menu = authedPage.getByTestId('task-row-menu')
    await expect(menu.getByRole('menuitem')).toHaveCount(3)
    await authedPage.keyboard.press('Escape')
    await expect(menu).toBeHidden()
  })

  test('task row: swipe left uncovers the actions', async ({ authedPage }) => {
    await withSettings(authedPage, { task_row_actions: 'swipe' })
    await authedPage.goto('/tasks')
    const swipe = authedPage.getByTestId('task-row-swipe').first()
    await expect(swipe).toBeVisible({ timeout: 15_000 })
    const box = (await swipe.boundingBox())!
    const y = box.y + box.height / 2
    await authedPage.mouse.move(box.x + box.width - 20, y)
    await authedPage.mouse.down()
    await authedPage.mouse.move(box.x + box.width - 160, y, { steps: 8 })
    await authedPage.mouse.up()
    await expect(swipe).toHaveAttribute('data-open', 'true')
    await expect(authedPage.getByRole('dialog')).toHaveCount(0)
  })

  test('task row: a tap opens the card with the actions', async ({ authedPage }) => {
    await withSettings(authedPage, { task_row_actions: 'card' })
    await authedPage.goto('/tasks')
    const row = authedPage.getByTestId('task-row-card').first()
    await expect(row).toBeVisible({ timeout: 15_000 })
    // The title, not the done circle: the circle keeps its own job.
    await row.locator('.font-mono').first().click()
    const sheet = authedPage.getByTestId('task-action-sheet')
    await expect(sheet).toBeVisible()
    await expect(sheet.getByRole('button')).toHaveCount(4)
    await authedPage.keyboard.press('Escape')
    await expect(sheet).toBeHidden()
  })
})

/**
 * Substitutes account settings in the /auth/me answer for this page only, and
 * blocks settings writes: the e2e account is a real person's staging account.
 */
async function withSettings(page: Page, settings: Record<string, unknown>) {
  await page.route('**/api/auth/me', (route) =>
    route.request().method() === 'PATCH' ? route.abort() : route.fallback(),
  )
  await page.addInitScript((extra) => {
    const orig = window.fetch.bind(window)
    window.fetch = async (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      const res = await orig(input, init)
      if (!url.includes('/api/auth/me') || (init?.method && init.method !== 'GET')) return res
      const body = await res.clone().json()
      body.data.settings = { ...(body.data.settings ?? {}), ...extra }
      return new Response(JSON.stringify(body), { status: res.status, headers: { 'Content-Type': 'application/json' } })
    }
  }, settings)
}
