import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'

/**
 * Gap list row 21 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md):
 * the bot opens an event from a list straight in its editor; the web's «Что
 * дальше» rows did nothing, so on a phone an event a few weeks away was dozens
 * of day swipes off. Now a tap opens the calendar on that day with the editor.
 *
 * Nothing is written to the account: the event is added to the page's own
 * events answers, and every write is refused.
 */
const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const ID = 'e2e-agenda-open'
const TITLE = 'e2e стоматолог из списка'
const HOUR = 60 * 60 * 1000

test('an event in «Что дальше» opens in the editor', async ({ authedPage: page, session }) => {
  const ctx = await playwrightRequest.newContext({ baseURL: API_BASE, extraHTTPHeaders: { Authorization: `Bearer ${session.token}` } })
  const timeZone: string = (await (await ctx.get('/api/auth/me')).json()).data.timezone || 'Europe/Moscow'
  await ctx.dispose()

  // Tomorrow 11:00–12:00 in the account's zone.
  const start = localMidnightUtc(timeZone, 1) + 11 * HOUR
  const event = {
    id: ID, title: TITLE, all_day: false, tags: [],
    starts_at: new Date(start).toISOString(), ends_at: new Date(start + HOUR).toISOString(),
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

  await page.goto('/agenda')
  const row = page.getByTestId('agenda-item').filter({ hasText: TITLE })
  await row.getByRole('link').click()

  await expect(page).toHaveURL(/\/calendar/)
  // The editor's title field holds the event (the first text inputs are times).
  const editor = page.getByTestId('event-editor')
  await expect(editor).toBeVisible({ timeout: 15_000 })
  await expect
    .poll(() => editor.locator('input[type="text"]').evaluateAll((els) => els.map((e) => (e as HTMLInputElement).value)))
    .toContain(TITLE)
  // The parameters are dropped: a reload does not reopen the editor.
  await expect(page).not.toHaveURL(/event=/)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
