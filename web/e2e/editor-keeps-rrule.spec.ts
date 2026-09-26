import { test, expect } from './fixtures/auth'

/**
 * Gap list F2 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): the
 * event editor rebuilt the repeat rule from its three fields, so saving a
 * bot-made «every 3 days» made it daily — HTTP 200, nothing on the screen.
 *
 * Nothing is written to the account: the event is added to the page's own
 * events answer, and its PATCH is caught and answered here. The event sits at
 * the current hour, which is where the grid opens (now − 1), in any zone.
 */
const ID = 'e2e-rrule-interval'
const TITLE = 'e2e каждые 3 дня'
const SAVE_BUTTON = /^(Save|Сохранить)$/

test('saving an event keeps the interval of its repeat rule', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: the rule is built the same way')

  const start = new Date()
  start.setMinutes(0, 0, 0)
  const event = {
    id: ID, title: TITLE, starts_at: start.toISOString(),
    ends_at: new Date(start.getTime() + 60 * 60_000).toISOString(), rrule: 'FREQ=DAILY;INTERVAL=3', tags: [],
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
  await page.route(`**/api/events/${ID}**`, async (route) => {
    const req = route.request()
    if (req.method() === 'PATCH') sent = req.postDataJSON()
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: event }) })
  })
  await page.goto('/calendar')
  const block = page.locator('div.absolute.rounded').filter({ hasText: TITLE }).first()
  await block.waitFor({ timeout: 20_000 })
  await block.dblclick()
  await page.getByRole('button', { name: SAVE_BUTTON }).click()
  // A repeating event asks which ones to change; all of them is the rule's own save.
  const all = page.getByRole('button', { name: /^(All events|Все события)$/ })
  if (await all.isVisible({ timeout: 3_000 }).catch(() => false)) await all.click()

  await expect.poll(() => sent?.rrule, { timeout: 10_000 }).toBe('FREQ=DAILY;INTERVAL=3')
  // The grid keeps refetching events; let in-flight ones end with the test.
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
