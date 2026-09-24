import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { DAY_MS, localMidnightUtc } from './fixtures/localTime'

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
  await expect(authedPage.getByText(/^(New Event|Новое событие)$/)).toBeVisible()
  // The double click must not also have left the month.
  await expect(authedPage.getByTestId('month-view')).toBeVisible()
  // The editor does not close on Escape (a separate defect, docs/tasks-web-month.md); the backdrop does.
  await authedPage.mouse.click(5, 5)
  await expect(authedPage.getByText(/^(New Event|Новое событие)$/)).toHaveCount(0)

  await cells.nth(20).click()
  await expect(authedPage.getByTestId('month-view')).toHaveCount(0, { timeout: 5_000 })
  await expect(authedPage.getByTestId('view-week')).toHaveAttribute('aria-selected', 'true')
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

    const fromDay = localDay(start.getTime(), timeZone)
    const toDay = localDay(start.getTime() + DAY_MS, timeZone)

    await authedPage.goto('/calendar')
    await authedPage.getByTestId('view-month').click({ timeout: 15_000 })
    const item = authedPage.locator(`[data-day="${fromDay}"] [data-event-id="${event.id}"]`)
    await expect(item, 'the seeded event is not in its month cell').toBeVisible({ timeout: 15_000 })

    const from = await item.boundingBox()
    const to = await authedPage.locator(`[data-day="${toDay}"]`).boundingBox()
    expect(from && to).toBeTruthy()
    await authedPage.mouse.move(from!.x + 5, from!.y + from!.height / 2)
    await authedPage.mouse.down()
    await authedPage.mouse.move(to!.x + to!.width / 2, to!.y + to!.height / 2, { steps: 12 })
    await authedPage.mouse.up()

    await expect
      .poll(
        async () => {
          const res = await ctx.get(`/api/events/${event.id}`)
          const body = (await res.json()).data as { starts_at: string; ends_at: string }
          return [body.starts_at, body.ends_at].map((s) => new Date(s).toISOString())
        },
        { timeout: 15_000, message: 'the event was not moved one day on' },
      )
      .toEqual([new Date(start.getTime() + DAY_MS).toISOString(), new Date(end.getTime() + DAY_MS).toISOString()])
  })
})
