import {
  test,
  expect,
  request as playwrightRequest,
  type APIRequestContext,
  type Locator,
  type Page,
} from '@playwright/test'
import { applySession } from './fixtures/auth'

/**
 * The 👥 badge and the author's name, measured inside the block that holds
 * them, at 375px.
 *
 * 🔴 Why this is not covered by mobile-overflow.spec.ts. That spec asks whether
 * the DOCUMENT scrolls sideways, and the answer here is no whatever happens:
 * the event block sets `overflow-hidden`, so a name too long for it is clipped
 * rather than pushed off the page. The page stays honest while the label
 * becomes unreadable — invisible to every assertion this project had.
 *
 * 🔴 And why the guest's name is deliberately long. /profile passed its spec at
 * 375px for weeks while overflowing on a real account: the e2e user's email was
 * short enough not to break it. Fixture data that cannot exercise the failure
 * IS the failure. The name here is longer than any block can show, so
 * truncation is guaranteed and the question becomes "does truncation stay
 * inside the block" — which is the real one.
 *
 * TWO block sizes, because they take different paths through EventBlock:
 *   90 minutes → 66px → the time row renders, author sits beside the time
 *   30 minutes → 22px → no time row at all; author moves onto the title line
 * The second case is why this spec exists. Its first run, on 19.08, found the
 * badge visible and the author element absent entirely: HOUR_PX is 44, so
 * anything under ~48 minutes fell below the `height > 35` gate and showed
 * "shared" without ever saying by whom.
 *
 * The events are created BY THE GUEST and read BY THE OWNER, because
 * author_name is answered per viewer: an event you wrote yourself is never
 * labelled with your own name, so a spec where one account did both would
 * assert on an element that is correctly absent.
 *
 * 🔴 Both accounts are registered by this spec rather than taken from the
 * Telegram fixture. That fixture skips when E2E_TG_BOT_TOKEN is absent, which
 * is every developer machine — and a spec that skips reports green while
 * proving nothing (learning-green-because-skipped-proves-nothing). This one
 * runs anywhere the API is reachable.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

// Long enough to overflow any block, and recognisable in a failure screenshot.
const LONG_NAME = 'Александра Константинопольская-Верещагина'

const SHORT_EVENT = 'Обед'
const LONG_EVENT = 'Созвон по проекту'

interface Account {
  api: APIRequestContext
  email: string
  token: string
  expiresAt: number
}

async function register(label: string, name: string): Promise<Account> {
  const email = `e2e-badge-${label}-${Date.now()}@example.com`
  const anon = await playwrightRequest.newContext({ baseURL: API_BASE })
  const res = await anon.post('/api/auth/register', {
    data: { email, password: 'e2e-Password-1234', name },
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok()) throw new Error(`register ${label}: ${res.status()} ${await res.text()}`)
  const body = (await res.json()).data
  await anon.dispose()

  return {
    email,
    token: body.token,
    expiresAt: body.expires_at,
    api: await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: { Authorization: `Bearer ${body.token}`, 'Content-Type': 'application/json' },
    }),
  }
}

/** Today at a given local hour and minute. */
function todayAt(hour: number, minutes = 0): Date {
  const d = new Date()
  d.setHours(hour, minutes, 0, 0)
  return d
}

/**
 * Asserts that everything drawn inside one event block stays inside it.
 *
 * Takes the block rather than finding it, so the caller decides WHICH event is
 * under test — with two on screen, a `.first()` here would silently measure
 * whichever happened to paint first.
 */
async function expectContained(block: Locator, label: string) {
  const badge = block.getByTestId('shared-badge')
  const author = block.getByTestId('event-author')

  await expect(badge, `${label}: no 👥 badge — is_shared never reached the grid`).toBeVisible()
  await expect(
    author,
    `${label}: no author label. On a short block this is the ~48-minute gate: HOUR_PX is 44, ` +
      'so the time row that used to carry the author is not drawn at all.',
  ).toBeVisible()

  // Exactly one of each. The author has two possible homes in EventBlock and
  // they must be exclusive; two elements with the same testid read as a
  // rendering bug on screen and make every bound below ambiguous.
  await expect(author, `${label}: the author is printed twice`).toHaveCount(1)

  const blockBox = await block.boundingBox()
  const badgeBox = await badge.boundingBox()
  const authorBox = await author.boundingBox()
  expect(blockBox && badgeBox && authorBox, `${label}: something has no layout box`).toBeTruthy()
  if (!blockBox || !badgeBox || !authorBox) return

  // The badge is drawn, not collapsed to nothing by flex pressure.
  expect(badgeBox.width, `${label}: the 👥 badge has no width`).toBeGreaterThan(4)

  // Nothing paints past the right edge. One pixel of tolerance for sub-pixel
  // layout at deviceScaleFactor 2.
  const right = blockBox.x + blockBox.width
  expect(badgeBox.x + badgeBox.width, `${label}: the badge is painted outside its block`)
    .toBeLessThanOrEqual(right + 1)
  expect(authorBox.x + authorBox.width, `${label}: the author name is painted outside its block`)
    .toBeLessThanOrEqual(right + 1)

  // Vertically the test is about READABILITY, not containment, and the
  // difference is not pedantic.
  //
  // 🔴 This assertion was `authorBottom <= blockBottom + 1` and it cost two CI
  // rounds. A span's bounding box is its font's ascent plus descent — a
  // platform fact, not a layout one. The same page measured +5px of headroom on
  // Windows and −2px on Linux: a 7px swing from the mono fallback alone, before
  // any phone with its own fonts is considered. And the block sets
  // overflow-hidden, so a descender past the edge is CLIPPED, not painted
  // outside: demanding exact containment asked the layout for something the
  // renderer never promised, on a property the user cannot see.
  //
  // What the user can see is how much of the label is inside. Most of it must
  // be, and it must START inside — an element pushed onto a second row lands
  // wholly below and fails both halves, which is the failure this guards.
  const blockBottom = blockBox.y + blockBox.height
  expect(authorBox.y, `${label}: the author starts below the block entirely`)
    .toBeLessThan(blockBottom)
  const visibleHeight = Math.min(authorBox.y + authorBox.height, blockBottom) - authorBox.y
  expect(
    visibleHeight / authorBox.height,
    `${label}: only ${Math.round((visibleHeight / authorBox.height) * 100)}% of the author label is inside the block`,
  ).toBeGreaterThan(0.7)

  // Truncated is fine; erased is not. A name clipped to "· " tells the viewer
  // nothing and would satisfy every bound above.
  expect(authorBox.width, `${label}: the author label is too narrow to read anything`)
    .toBeGreaterThan(28)
}

/** The event block carrying a given title. */
function blockFor(page: Page, title: string): Locator {
  return page.locator('div.absolute.rounded').filter({ hasText: title }).first()
}

test.describe('shared-event badge at 375px', () => {
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'mobile viewport only')
  })

  test('the badge and the author stay inside the event block', async ({ page }) => {
    const owner = await register('owner', 'E2E Owner')
    const guest = await register('guest', LONG_NAME)
    await applySession(page, { token: owner.token, expiresAt: owner.expiresAt })

    let calendarId: string | undefined
    const eventIds: string[] = []

    try {
      const created = await owner.api.post('/api/calendars', {
        data: { name: `E2E Badge ${Date.now()}`, color: '#7c3aed' },
      })
      expect(created.status(), await created.text()).toBe(201)
      calendarId = (await created.json()).data.id

      const invited = await owner.api.post(`/api/calendars/${calendarId}/invites`, {
        data: { email: guest.email, role: 'editor' },
      })
      expect(invited.status(), await invited.text()).toBe(201)

      const accepted = await guest.api.post(`/api/calendars/${calendarId}/invitation`, {
        data: { accept: true },
      })
      expect(accepted.status(), await accepted.text()).toBe(204)

      const events: Array<[string, Date, number]> = [
        [SHORT_EVENT, todayAt(10), 30],
        [LONG_EVENT, todayAt(12), 90],
      ]
      for (const [title, start, minutes] of events) {
        const res = await guest.api.post('/api/events', {
          data: {
            title,
            starts_at: start.toISOString(),
            ends_at: new Date(start.getTime() + minutes * 60_000).toISOString(),
            calendar_id: calendarId,
          },
        })
        expect(res.status(), await res.text()).toBe(201)
        eventIds.push((await res.json()).data.id)
      }

      await page.goto('/calendar')
      await page.waitForLoadState('networkidle')

      // The positive control comes first: if the grid drew no shared event at
      // all, every containment assertion below would pass vacuously.
      await expect(
        page.getByTestId('shared-badge').first(),
        'no shared event rendered — the rest of this spec would prove nothing',
      ).toBeVisible({ timeout: 15_000 })

      await expectContained(blockFor(page, LONG_EVENT), '90-minute block (time row)')
      await expectContained(blockFor(page, SHORT_EVENT), '30-minute block (title row)')

      // The time survives on the tall block. It is flex-shrink-0, so the author
      // yields first by design — this pins that design rather than trusting it.
      const timeText = await blockFor(page, LONG_EVENT).textContent()
      expect(timeText ?? '', 'the time was squeezed out of a 90-minute block').toMatch(/\d{1,2}:\d{2}/)

      // And the page still does not scroll sideways, which is what a fix
      // reaching for min-width instead of truncation would cause.
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, 'the calendar overflows 375px with shared events on screen').toBe(0)
    } finally {
      for (const id of eventIds) await guest.api.delete(`/api/events/${id}`).catch(() => {})
      if (calendarId) await owner.api.delete(`/api/calendars/${calendarId}`).catch(() => {})
      await guest.api.dispose()
      await owner.api.dispose()
    }
  })
})
