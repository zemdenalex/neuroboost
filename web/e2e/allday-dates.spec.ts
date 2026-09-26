import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'

/**
 * Gap list row 9 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): the
 * editor hid every date field of an all-day event. Now it shows the first and
 * the last day, and saves midnight to the midnight after the last day — the
 * bot's convention (draftflow.go draftBounds).
 *
 * Nothing is written to the account: the event is added to the page's own
 * events answer and its PATCH is caught here; every other write is refused.
 */
const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const ID = 'e2e-allday-span'
const TITLE = 'e2e отпуск'
const DAY = 24 * 60 * 60 * 1000

test('an all-day event can be stretched to more days', async ({ authedPage: page, session }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: the same form on both')

  const ctx = await playwrightRequest.newContext({ baseURL: API_BASE, extraHTTPHeaders: { Authorization: `Bearer ${session.token}` } })
  const timeZone: string = (await (await ctx.get('/api/auth/me')).json()).data.timezone || 'Europe/Moscow'
  await ctx.dispose()

  const start = localMidnightUtc(timeZone, 0)
  const event = {
    id: ID, title: TITLE, all_day: true, tags: [],
    starts_at: new Date(start).toISOString(), ends_at: new Date(start + DAY).toISOString(),
  }

  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  await page.route('**/api/events?*', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const list = Array.isArray(body.data) ? body.data : body.data?.events ?? []
    list.push(event)
    await route.fulfill({ response: res, json: Array.isArray(body.data) ? { ...body, data: list } : { ...body, data: { ...body.data, events: list } } })
  })
  let sent: { starts_at?: string; ends_at?: string } | undefined
  await page.route(`**/api/events/${ID}**`, async (route) => {
    if (route.request().method() === 'PATCH') sent = route.request().postDataJSON()
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: event }) })
  })

  await page.goto('/calendar')
  const block = page.getByText(TITLE).first()
  await block.waitFor({ timeout: 20_000 })
  await block.dblclick()

  const first = page.getByTestId('event-start-date')
  const last = page.getByTestId('event-end-date')
  await expect(last, 'the last day is shown, not the midnight after it').toHaveValue(await first.inputValue())
  const lastDay = new Date(start + 2 * DAY + 12 * 60 * 60 * 1000).toLocaleDateString('en-CA', { timeZone })
  await last.fill(lastDay)
  await page.getByRole('button', { name: /^(Save|Сохранить)$/ }).click()

  await expect.poll(() => sent?.ends_at, { timeout: 10_000 }).toBeTruthy()
  expect(new Date(sent!.starts_at!).getTime()).toBe(start)
  expect(new Date(sent!.ends_at!).getTime(), 'three days: to the midnight after the last one').toBe(start + 3 * DAY)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
