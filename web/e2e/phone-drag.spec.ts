import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext, type Page } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'

/**
 * Phone drag (Denis, 25.10: «how the dragging works on the phone, it just
 * straight up doesn't right now»).
 *
 * The week grid's move / resize / create all listen to mousedown, mousemove and
 * mouseup. A finger produces none of the moves, so on a phone nothing can be
 * dragged. This spec holds the finger on an event and moves it one hour down
 * with real touch events (CDP Input.dispatchTouchEvent), and asserts against
 * the API, not the pixels.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.space'
const HOUR_PX = 44
const HOUR = 3600 * 1000

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

/** A finger: down, optional hold, then moves in small steps, then up. */
async function fingerDrag(
  page: Page,
  from: { x: number; y: number },
  to: { x: number; y: number },
  holdMs: number,
  end: 'up' | 'cancel' = 'up',
) {
  const cdp = await page.context().newCDPSession(page)
  const touch = (type: 'touchStart' | 'touchMove' | 'touchEnd' | 'touchCancel', p?: { x: number; y: number }) =>
    cdp.send('Input.dispatchTouchEvent', { type, touchPoints: p ? [{ x: p.x, y: p.y }] : [] })
  await touch('touchStart', from)
  if (holdMs > 0) await page.waitForTimeout(holdMs)
  const steps = 12
  for (let i = 1; i <= steps; i++) {
    await touch('touchMove', {
      x: from.x + ((to.x - from.x) * i) / steps,
      y: from.y + ((to.y - from.y) * i) / steps,
    })
    await page.waitForTimeout(16)
  }
  await touch(end === 'up' ? 'touchEnd' : 'touchCancel')
  await cdp.detach()
}

test.describe('phone drag', () => {
  let createdId: string | undefined

  test.afterEach(async ({ session }) => {
    if (createdId) {
      const ctx = await apiContext(session.token)
      await ctx.delete(`/api/events/${createdId}`)
      await ctx.dispose()
      createdId = undefined
    }
  })

  /** An event 06:30–08:30 today, on screen and clear of the bottom bar. */
  async function seedEventOnScreen(authedPage: Page, token: string) {
    const ctx = await apiContext(token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'
    const dayStart = localMidnightUtc(timeZone, 0)
    // Same band as move-grab-offset: 06:30–08:30.
    const startMs = dayStart + 6 * HOUR + 30 * 60 * 1000
    const endMs = dayStart + 8 * HOUR + 30 * 60 * 1000
    const title = `E2E phone drag ${Date.now()}`
    const created = await ctx.post('/api/events', {
      data: {
        title,
        starts_at: new Date(startMs).toISOString(),
        ends_at: new Date(endMs).toISOString(),
        all_day: false,
      },
    })
    expect(created.ok(), `create failed: ${created.status()} ${await created.text()}`).toBe(true)
    createdId = (await created.json()).data.id
    await ctx.dispose()

    await authedPage.goto('/calendar')
    await authedPage.waitForLoadState('networkidle')
    const block = authedPage.locator(`[title^="${title}"]`).first()
    await expect(block).toBeVisible({ timeout: 15_000 })
    // 375x667: at 00:00 the 06:30 block sits under the bottom bar (its centre is
    // at y=628, the bar starts near 603), so a finger there would press a tab, not
    // the event. Bring it to the middle of the screen, without the smooth scroll
    // `html` asks for, which would hand back coordinates mid-animation.
    await block.evaluate((el) => el.scrollIntoView({ block: 'center', behavior: 'instant' }))
    await authedPage.evaluate(() => new Promise((r) => requestAnimationFrame(() => r(null))))
    const box = await block.boundingBox()
    expect(box, 'the event block must be on screen').not.toBeNull()
    return { block, box: box!, startMs }
  }

  async function startOf(token: string, id: string): Promise<number> {
    const check = await apiContext(token)
    const after = (await (await check.get(`/api/events/${id}`)).json()).data
    await check.dispose()
    return new Date(after.starts_at ?? after.startsAt).getTime()
  }

  test('holding an event and dragging it one hour down moves it one hour', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    await fingerDrag(authedPage, { x, y }, { x, y: y + HOUR_PX }, 600)

    await authedPage.waitForTimeout(1500)
    expect(
      (await startOf(session.token, createdId!)) - startMs,
      'a finger held on the event and moved one hour down must move it one hour'
    ).toBe(HOUR)
  })

  test('a quick swipe over an event scrolls the day and leaves the event alone', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { block, box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    // No hold: the finger starts moving at once, the way a thumb scrolls.
    await fingerDrag(authedPage, { x, y }, { x, y: y + 2 * HOUR_PX }, 0)
    await authedPage.waitForTimeout(1200)

    const moved = await block.boundingBox()
    expect(moved!.y, 'the day should have scrolled under the finger').not.toBe(box.y)
    expect(await startOf(session.token, createdId!), 'a scroll must not move the event').toBe(startMs)
  })

  test('a gesture the system takes away (touchcancel) commits nothing', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    await fingerDrag(authedPage, { x, y }, { x, y: y + HOUR_PX }, 600, 'cancel')
    await authedPage.waitForTimeout(1200)
    expect(await startOf(session.token, createdId!), 'a cancelled drag must not be saved').toBe(startMs)
  })

  test('a held event dragged to the bottom edge makes the day scroll and the event follow it', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { block, box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const scroller = () =>
      block.evaluate((el) => {
        let n: HTMLElement | null = el.parentElement
        while (n && getComputedStyle(n).overflowY !== 'auto') n = n.parentElement
        const r = n!.getBoundingClientRect()
        return { top: r.top, bottom: r.bottom, scrollTop: n!.scrollTop }
      })
    const before = await scroller()

    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    const cdp = await authedPage.context().newCDPSession(authedPage)
    const touch = (type: 'touchStart' | 'touchMove' | 'touchEnd' | 'touchCancel', yy?: number) =>
      cdp.send('Input.dispatchTouchEvent', { type, touchPoints: yy === undefined ? [] : [{ x, y: yy }] })
    await touch('touchStart', y)
    await authedPage.waitForTimeout(600)
    // Down to 10px above the bottom of the scrolling area, in steps, and stay there.
    const target = before.bottom - 10
    for (let i = 1; i <= 12; i++) {
      await touch('touchMove', y + ((target - y) * i) / 12)
      await authedPage.waitForTimeout(16)
    }
    await authedPage.waitForTimeout(700)
    const during = await scroller()
    await touch('touchEnd')
    await cdp.detach()

    expect(during.scrollTop, 'holding the finger at the bottom edge should scroll the grid down').toBeGreaterThan(before.scrollTop + 20)
    await authedPage.waitForTimeout(1500)
    // The finger stood still while the grid slid ~8 hours under it (about 3 hours of that
    // happen before its last move, so the bar is 6). The event has to
    // be where the finger is now, not where it was when the finger last moved.
    expect(
      (await startOf(session.token, createdId!)) - startMs,
      'the event should have followed the scrolling grid, not stayed at the time under the finger when it last moved'
    ).toBeGreaterThan(6 * HOUR)
  })

  test('after a move, «Отменить» in the toast puts the event back', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    await fingerDrag(authedPage, { x, y }, { x, y: y + HOUR_PX }, 600)

    const undo = authedPage.getByRole('status').getByRole('button', { name: /^(Отменить|Undo)$/ })
    await expect(undo, 'a finished move should offer to undo it').toBeVisible({ timeout: 5000 })
    // Positive control: the move really landed, or putting it back proves nothing.
    await expect.poll(() => startOf(session.token, createdId!), { timeout: 5000 }).toBe(startMs + HOUR)
    await undo.click()
    await expect.poll(() => startOf(session.token, createdId!), { timeout: 5000 }).toBe(startMs)
  })
})
