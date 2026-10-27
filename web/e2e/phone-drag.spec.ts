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
    // Runs that died before afterEach (a dropped connection) leave their events
    // behind, and five of them in this band squeeze each block to a fifth of the
    // column. Only this spec's own titles are removed (the account is a real one),
    // and only stale ones: the tests run in parallel, and a sibling's event made a
    // second ago is in use. The title carries its creation time.
    const listed = await ctx.get(`/api/events?start=${new Date(dayStart).toISOString()}&end=${new Date(dayStart + 24 * HOUR).toISOString()}`)
    for (const ev of ((await listed.json()).data ?? []) as { id: string; title: string }[]) {
      const made = Number(/^E2E phone drag (\d+)$/.exec(ev.title)?.[1])
      if (made && Date.now() - made > 10 * 60 * 1000) await ctx.delete(`/api/events/${ev.id}`)
    }
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

  test('a tapped event shows a circle at its bottom edge; dragging it one hour down makes the event an hour longer', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { block, box, startMs } = await seedEventOnScreen(authedPage, session.token)
    // A tap selects the event (no hold, no move).
    const cdp = await authedPage.context().newCDPSession(authedPage)
    const c = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [c] })
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
    await cdp.detach()

    const handle = block.getByTestId('resize-handle-bottom')
    await expect(handle, 'a selected event offers a circle to stretch it').toBeVisible({ timeout: 5000 })
    const hb = (await handle.boundingBox())!
    const from = { x: hb.x + hb.width / 2, y: hb.y + hb.height / 2 }
    // Held past the lift delay on purpose: a circle stretches, it never picks the
    // event up (TouchHoldBinder skips touches that start on a circle).
    await fingerDrag(authedPage, from, { x: from.x, y: from.y + HOUR_PX }, 600)

    await authedPage.waitForTimeout(1500)
    const check = await apiContext(session.token)
    const after = (await (await check.get(`/api/events/${createdId}`)).json()).data
    await check.dispose()
    const start = new Date(after.starts_at ?? after.startsAt).getTime()
    const end = new Date(after.ends_at ?? after.endsAt).getTime()
    expect(start, 'stretching the bottom edge must not move the start').toBe(startMs)
    expect(end - start, 'the event was 2 hours, the circle was dragged one hour down').toBe(3 * HOUR)
  })

  /**
   * A free 3-hour window today (09:00–20:00, on the hour), centred on screen.
   * The account is a real one: a hold that lands on someone's event lifts that
   * event instead of creating. Returns the finger's y for the window's start.
   */
  async function freeWindowOnScreen(authedPage: Page, token: string) {
    const { block } = await seedEventOnScreen(authedPage, token)
    const ctx = await apiContext(token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const dayStart = localMidnightUtc(me.timezone || 'Europe/Moscow', 0)
    const listed = await ctx.get(`/api/events?start=${new Date(dayStart).toISOString()}&end=${new Date(dayStart + 24 * HOUR).toISOString()}`)
    const busy = (((await listed.json()).data ?? []) as { starts_at: string; ends_at: string; all_day?: boolean }[])
      .filter((e) => !e.all_day)
      .map((e) => [new Date(e.starts_at).getTime(), new Date(e.ends_at).getTime()])
    await ctx.dispose()
    const hour = [9, 10, 11, 12, 13, 14, 15, 16, 17].find((h) => {
      const a = dayStart + h * HOUR, b = a + 3 * HOUR
      return busy.every(([s, e]) => e <= a || s >= b)
    })
    expect(hour, 'no free 3-hour window today between 09:00 and 20:00').toBeDefined()
    // The track is the seeded block's parent; its top is 00:00 on screen.
    const y = await block.evaluate((el, h) => {
      const track = el.parentElement!
      let sc: HTMLElement | null = track.parentElement
      while (sc && getComputedStyle(sc).overflowY !== 'auto') sc = sc.parentElement
      const want = track.getBoundingClientRect().top + h * 44
      const r = sc!.getBoundingClientRect()
      sc!.scrollTop += want - (r.top + r.height / 3)
      return track.getBoundingClientRect().top + h * 44
    }, hour!)
    await authedPage.evaluate(() => new Promise((r) => requestAnimationFrame(() => r(null))))
    const x = (await block.boundingBox())!.x + 20
    const settingMin = typeof me.settings?.new_event_minutes === 'number' ? me.settings.new_event_minutes : 30
    return { x, y: y + 2, hour: hour!, settingMin }
  }

  const hhmm = (h: number, m = 0) => `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`

  test('holding an empty slot and dragging two hours down opens a new event for those two hours', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { x, y, hour } = await freeWindowOnScreen(authedPage, session.token)
    await fingerDrag(authedPage, { x, y }, { x, y: y + 2 * HOUR_PX }, 600)

    await expect(authedPage.getByTestId('event-start-time'), 'the new-event form should open').toBeVisible({ timeout: 5000 })
    await expect(authedPage.getByTestId('event-start-time')).toHaveValue(hhmm(hour))
    await expect(authedPage.getByTestId('event-end-time'), 'the end is where the finger was lifted').toHaveValue(hhmm(hour + 2))
  })

  test('holding an empty slot without moving opens a new event of the length set in ⚙️', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { x, y, hour, settingMin } = await freeWindowOnScreen(authedPage, session.token)
    await fingerDrag(authedPage, { x, y }, { x, y }, 600)

    await expect(authedPage.getByTestId('event-start-time'), 'the new-event form should open').toBeVisible({ timeout: 5000 })
    await expect(authedPage.getByTestId('event-start-time')).toHaveValue(hhmm(hour))
    await expect(authedPage.getByTestId('event-end-time')).toHaveValue(hhmm(hour + Math.floor(settingMin / 60), settingMin % 60))
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
