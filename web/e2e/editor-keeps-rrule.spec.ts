import type { Page } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * Gap list F2 and row 11 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md).
 * F2: the event editor rebuilt the repeat rule from its three fields, so saving
 * a bot-made «every 3 days» made it daily — HTTP 200, nothing on the screen.
 * Row 11: the form sets the interval and «every year» (MONTHLY;INTERVAL=12,
 * the bot's own form of it) itself.
 *
 * Nothing is written to the account: the event is added to the page's own
 * events answer, and its PATCH is caught and answered here. The event sits at
 * the current hour, which is where the grid opens (now − 1), in any zone.
 */
const SAVE_BUTTON = /^(Save|Сохранить)$/

const CASES = [
  { name: 'keeps «every 3 days»', rrule: 'FREQ=DAILY;INTERVAL=3', shows: 'daily', act: async () => {}, sends: 'FREQ=DAILY;INTERVAL=3' },
  { name: 'keeps a yearly birthday', rrule: 'FREQ=MONTHLY;INTERVAL=12', shows: 'yearly', act: async () => {}, sends: 'FREQ=MONTHLY;INTERVAL=12' },
  {
    name: 'changes the interval', rrule: 'FREQ=DAILY;INTERVAL=3', shows: 'daily',
    act: async (page: Page) => page.getByTestId('event-repeat-interval').fill('5'),
    sends: 'FREQ=DAILY;INTERVAL=5',
  },
]

for (const [i, c] of CASES.entries()) {
  test(`the event editor ${c.name}`, async ({ authedPage: page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: the rule is built the same way')

    const id = `e2e-rrule-${i}`
    const title = `e2e повтор ${i}`
    const start = new Date()
    start.setMinutes(0, 0, 0)
    const event = {
      id, title, starts_at: start.toISOString(),
      ends_at: new Date(start.getTime() + 60 * 60_000).toISOString(), rrule: c.rrule, tags: [],
    }

    // Every other write is refused, so a wrong path cannot touch the account.
    // Registered first: Playwright tries the newest route first, so the two below win.
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
    let sent: { rrule?: string } | undefined
    await page.route(`**/api/events/${id}**`, async (route) => {
      if (route.request().method() === 'PATCH') sent = route.request().postDataJSON()
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: event }) })
    })

    await page.goto('/calendar')
    const block = page.locator('div.absolute.rounded').filter({ hasText: title }).first()
    await block.waitFor({ timeout: 20_000 })
    await block.dblclick()
    // The repeat fields are under «Advanced».
    await page.getByRole('button', { name: /^(Advanced|Подробнее|Расширенные) \+$/ }).click()
    await expect(page.getByTestId('event-repeat')).toHaveValue(c.shows)
    await c.act(page)
    await page.getByRole('button', { name: SAVE_BUTTON }).click()
    // A repeating event asks which ones to change; all of them is the rule's own save.
    const all = page.getByRole('button', { name: /^(All events|Все события)$/ })
    if (await all.isVisible({ timeout: 3_000 }).catch(() => false)) await all.click()

    await expect.poll(() => sent?.rrule, { timeout: 10_000 }).toBe(c.sends)
    // The grid keeps refetching events; let in-flight ones end with the test.
    await page.unrouteAll({ behavior: 'ignoreErrors' })
  })
}
