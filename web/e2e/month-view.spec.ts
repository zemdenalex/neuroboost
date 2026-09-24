import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'
import { shiftByDays } from '../src/lib/calendar/shiftDays'

/**
 * The web month (spec docs/team/architecture/V003-20260924-arc-web-month-view.md §4).
 *
 * Desktop only: in v1 the phone has no month (spec §3). The drag is asserted
 * against the API, not the screen, like calendar-move.spec.ts.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

/** YYYY-MM-DD of a UTC instant in a zone. */
function localDay(ms: number, timeZone: string): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).format(
    new Date(ms),
  )
}

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'the month is desktop-only in v1')
})

test('the week/month switch opens the month and is remembered after a reload', async ({ authedPage }) => {
  await authedPage.goto('/calendar')
  await authedPage.getByTestId('view-month').click({ timeout: 15_000 })
  await expect(authedPage.getByTestId('month-view')).toBeVisible()
  await expect(authedPage.getByTestId('month-day')).toHaveCount(42)

  await authedPage.reload()
  await expect(authedPage.getByTestId('month-view')).toBeVisible({ timeout: 15_000 })

  await authedPage.getByTestId('view-week').click()
  await expect(authedPage.getByTestId('month-view')).toHaveCount(0)
})

test('a click on a day opens its week; a double click opens a new event', async ({ authedPage }) => {
  await authedPage.goto('/calendar')
  await authedPage.getByTestId('view-month').click({ timeout: 15_000 })
  const cells = authedPage.getByTestId('month-day')
  await expect(cells).toHaveCount(42)

  await cells.nth(20).dblclick()
  // Past the single-click wait (CLICK_WAIT_MS = 250): a double click that
  // also let its first click through would have left the month by now.
  await authedPage.waitForTimeout(400)
  await expect(authedPage.getByText(/^(New Event|Новое событие)$/)).toBeVisible()
  await expect(authedPage.getByTestId('month-view')).toBeVisible()
  // Escape (Denis 24.09): untouched closes at once; after typing, the first
  // press warns and the second closes.
  await authedPage.keyboard.press('Escape')
  await expect(authedPage.getByTestId('event-editor')).toHaveCount(0)
  await cells.nth(20).dblclick()
  await expect(authedPage.getByTestId('event-editor')).toBeVisible()
  await authedPage.keyboard.type('e2e escape draft')
  await authedPage.keyboard.press('Escape')
  await expect(authedPage.getByTestId('escape-hint')).toBeVisible()
  await expect(authedPage.getByTestId('event-editor')).toBeVisible()
  await authedPage.keyboard.press('Escape')
  await expect(authedPage.getByTestId('event-editor')).toHaveCount(0)

  const clicked = await cells.nth(20).getAttribute('data-day')
  await cells.nth(20).click()
  await expect(authedPage.getByTestId('month-view')).toHaveCount(0, { timeout: 5_000 })
  await expect(authedPage.getByTestId('view-week')).toHaveAttribute('aria-selected', 'true')
  // And it is THAT day's week, not this week.
  await expect(authedPage.locator(`[data-testid="week-day-header"][data-day="${clicked}"]`)).toBeVisible()
})

test('Enter on a day opens its week, and Month then opens the month of that week', async ({ authedPage }) => {
  await authedPage.goto('/calendar')
  await authedPage.getByTestId('view-month').click({ timeout: 15_000 })
  // The last row is always next month's days.
  const cell = authedPage.getByTestId('month-day').nth(38)
  const day = await cell.getAttribute('data-day')
  const title = await authedPage.getByTestId('month-title').textContent()
  await cell.focus()
  await authedPage.keyboard.press('Enter')
  await expect(authedPage.locator(`[data-testid="week-day-header"][data-day="${day}"]`)).toBeVisible()
  await authedPage.getByTestId('view-month').click()
  await expect(authedPage.getByTestId('month-title')).not.toHaveText(title ?? '')
  await expect(authedPage.locator(`[data-testid="month-day"][data-day="${day}"]`)).toBeVisible()
})

test.describe('dragging in the month', () => {
  let cleanup: (() => Promise<void>) | undefined

  test.afterEach(async () => {
    if (cleanup) {
      await cleanup()
      cleanup = undefined
    }
  })

  test('an event dropped on the next day moves one calendar day, same local time', async ({ authedPage, session }) => {
    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone: string = me.timezone || 'Europe/Moscow'

    // The 21:00 band, two and three days out: always inside this month's grid.
    const start = new Date(localMidnightUtc(timeZone, 2) + 21 * 3600 * 1000)
    const end = new Date(start.getTime() + 30 * 60 * 1000)
    const title = `e2e month drag ${Date.now()}`
    const created = await ctx.post('/api/events', {
      data: { title, starts_at: start.toISOString(), ends_at: end.toISOString(), timezone: timeZone },
    })
    expect(created.status()).toBe(201)
    const event = (await created.json()).data as { id: string }
    cleanup = async () => {
      await ctx.delete(`/api/events/${event.id}`)
      await ctx.dispose()
    }

    // One calendar day in the account's zone, not 24 h (R8).
    const moved = shiftByDays(start.toISOString(), end.toISOString(), 1, timeZone)
    const fromDay = localDay(start.getTime(), timeZone)
    const toDay = localDay(Date.parse(moved.startsAt), timeZone)

    // The account is a real person's: whatever variant they chose, drag in the list.
    await withVariant(authedPage, 'list')
    await authedPage.goto('/calendar')
    await authedPage.getByTestId('view-month').click({ timeout: 15_000 })
    const item = authedPage.locator(`[data-day="${fromDay}"] [data-event-id="${event.id}"]`)
    await expect(item, 'the seeded event is not in its month cell').toBeVisible({ timeout: 15_000 })

    const from = await item.boundingBox()
    const to = await authedPage.locator(`[data-day="${toDay}"]`).boundingBox()
    expect(from && to).toBeTruthy()
    // First a drag that ends on its own day: nothing moves, and the click the
    // browser fires after the drop must not open the week.
    await authedPage.mouse.move(from!.x + 5, from!.y + from!.height / 2)
    await authedPage.mouse.down()
    await authedPage.mouse.move(from!.x + 40, from!.y + from!.height / 2 + 20, { steps: 6 })
    await authedPage.mouse.up()
    await authedPage.waitForTimeout(400)
    await expect(authedPage.getByTestId('month-view')).toBeVisible()
    const still = (await (await ctx.get(`/api/events/${event.id}`)).json()).data as { starts_at: string }
    expect(new Date(still.starts_at).toISOString()).toBe(start.toISOString())

    await authedPage.mouse.move(from!.x + 5, from!.y + from!.height / 2)
    await authedPage.mouse.down()
    await authedPage.mouse.move(to!.x + to!.width / 2, to!.y + to!.height / 2, { steps: 12 })
    await authedPage.mouse.up()
    await authedPage.waitForTimeout(400)
    await expect(authedPage.getByTestId('month-view'), 'the drop opened a week').toBeVisible()

    await expect
      .poll(
        async () => {
          const res = await ctx.get(`/api/events/${event.id}`)
          const body = (await res.json()).data as { starts_at: string; ends_at: string }
          return [body.starts_at, body.ends_at].map((s) => new Date(s).toISOString())
        },
        { timeout: 15_000, message: 'the event was not moved one day on' },
      )
      .toEqual([moved.startsAt, moved.endsAt])
  })
})

/**
 * The five variants and their setting, without writing to the account: the
 * e2e account is a real person's staging account and settings-race.spec.ts
 * writes the same blob in parallel. /auth/me is answered with the variant
 * patched in, and the settings PATCH is caught and inspected, not sent.
 */
async function withVariant(
  page: import('@playwright/test').Page,
  variant: string,
  opts: { settings?: Record<string, unknown>; today?: string } = {},
) {
  // A settings write from these tests would carry the fake variant into the
  // real account: block it.
  await page.route('**/api/auth/me', (route) =>
    route.request().method() === 'PATCH' ? route.abort() : route.fallback(),
  )
  await page.addInitScript(
    ({ v, settings, today }) => {
      const orig = window.fetch.bind(window)
      const json = (data: unknown, status = 200) =>
        new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })
      window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
        const get = !init?.method || init.method === 'GET'
        // One taken day today, 2 of 5 done: what the day-tasks month draws.
        if (today && get && url.includes('/api/day-tasks?')) {
          // The day before today: after the start, not taken, i.e. missed.
          const y = new Date(Date.parse(today + 'T12:00:00Z') - 86_400_000).toISOString().slice(0, 10)
          return json({
            data: [
              { day: y, target: 5, confirmed: false, done: 0, level: 0, before_start: false, items: [] },
              {
                day: today,
                target: 5,
                confirmed: true,
                done: 2,
                level: 2,
                before_start: false,
                items: [
                  { task_id: 'a', title: 'e2e one', done: true },
                  { task_id: 'b', title: 'e2e two', done: true },
                  { task_id: 'c', title: 'e2e three', done: false },
                ],
              },
            ],
          })
        }
        const res = await orig(input, init)
        if (!url.includes('/api/auth/me') || !get) return res
        const body = await res.clone().json()
        const user = body.data ?? body
        user.settings = { ...(user.settings ?? {}), ...settings, month_view_variant: v }
        return json(body.data ? { ...body, data: user } : user, res.status)
      }
    },
    { v: variant, settings: opts.settings ?? {}, today: opts.today ?? '' },
  )
}

const VARIANT_MARK: Record<string, string> = {
  list: 'month-item',
  classic: 'month-item',
  heat: 'month-heat',
  split: 'month-dots',
}

for (const variant of Object.keys(VARIANT_MARK)) {
  test(`the ${variant} variant draws its own month`, async ({ authedPage }) => {
    await withVariant(authedPage, variant)
    await authedPage.goto('/calendar')
    await authedPage.getByTestId('view-month').click({ timeout: 30_000 })
    await expect(authedPage.getByTestId('month-view')).toHaveAttribute('data-variant', variant)
    await expect(authedPage.getByTestId(VARIANT_MARK[variant]).first()).toBeAttached({ timeout: 15_000 })
    if (variant === 'split') await expect(authedPage.getByTestId('month-day-list')).toBeVisible()
  })
}

test('the day-tasks variant draws the taken day, and says when day tasks are off', async ({ authedPage, session }) => {
  const ctx = await apiContext(session.token)
  const me = (await (await ctx.get('/api/auth/me')).json()).data
  await ctx.dispose()
  const today = localDay(Date.now(), me.timezone || 'Europe/Moscow')

  await withVariant(authedPage, 'commit', { settings: { day_tasks_enabled: true }, today })
  await authedPage.goto('/calendar')
  await authedPage.getByTestId('view-month').click({ timeout: 30_000 })
  const cell = authedPage.locator(`[data-day="${today}"] [data-testid="month-commit"]`)
  await expect(cell).toBeVisible({ timeout: 15_000 })
  await expect(cell).toContainText('2/5')
  await expect(cell).toContainText('e2e three')
  // Day tasks, not event rows; events only as dots (Denis 24.09: «at least show dots for events»).
  await expect(authedPage.getByTestId('month-item')).toHaveCount(0)
  await expect(authedPage.getByTestId('month-dots').first()).toBeAttached()
  // A missed day after the start is an empty bar, not a blank cell.
  const y = new Date(Date.parse(today + 'T12:00:00Z') - 86_400_000).toISOString().slice(0, 10)
  const missed = authedPage.locator(`[data-day="${y}"] [data-testid="month-commit"]`)
  if ((await authedPage.locator(`[data-day="${y}"]`).count()) > 0) await expect(missed).toContainText('0/5')
})

test('the day-tasks variant with day tasks off shows the way to turn them on', async ({ authedPage }) => {
  await withVariant(authedPage, 'commit', { settings: { day_tasks_enabled: false } })
  await authedPage.goto('/calendar')
  await authedPage.getByTestId('view-month').click({ timeout: 30_000 })
  await expect(authedPage.getByText(/Day tasks are off|Задачи дня выключены/)).toBeVisible()
  await expect(authedPage.getByRole('link', { name: /Turn on in settings|Включить в настройках/ })).toHaveAttribute('href', '/settings')
})

test('choosing a variant in settings saves month_view_variant', async ({ authedPage }) => {
  const sent: Array<{ settings?: Record<string, unknown> }> = []
  await authedPage.route('**/api/auth/me', async (route) => {
    if (route.request().method() === 'PATCH') {
      const body = route.request().postDataJSON() as { settings?: Record<string, unknown> }
      sent.push(body)
      // Not sent on: the account is shared. Answered the way the server
      // would, the real user with the sent settings, so the page stays in a
      // state that can happen.
      const res = await route.fetch({ method: 'GET', postData: undefined })
      const got = await res.json()
      const user = { ...(got.data ?? got), settings: body.settings }
      return route.fulfill({ status: 200, json: got.data ? { ...got, data: user } : user })
    }
    return route.fallback()
  })
  await authedPage.goto('/settings')
  await authedPage.getByTestId('month-variant-heat').click({ timeout: 30_000 })
  await expect(authedPage.getByTestId('month-variant-heat')).toHaveAttribute('aria-checked', 'true')
  await expect.poll(() => sent.length, { timeout: 10_000 }).toBeGreaterThan(0)
  expect(sent[sent.length - 1].settings?.month_view_variant).toBe('heat')
  // Still chosen after the save came back.
  await expect(authedPage.getByTestId('month-variant-heat')).toHaveAttribute('aria-checked', 'true')
})
