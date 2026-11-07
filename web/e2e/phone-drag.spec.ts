import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext, type Page } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'
import { withSettings } from './fixtures/settings'

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

  // Step 5, the grip (variant 4: «hold / handle / both» in ⚙️). The setting is
  // laid over in this browser only: the e2e account is Denis's real one.
  test('with the grip on, the grip moves the event at once, without a hold', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    await withSettings(authedPage, { phone_drag_start: 'handle' })
    const { block, startMs } = await seedEventOnScreen(authedPage, session.token)
    const grip = block.getByTestId('event-grip')
    await expect(grip).toBeVisible()
    const gb = (await grip.boundingBox())!
    const from = { x: gb.x + gb.width / 2, y: gb.y + gb.height / 2 }
    await fingerDrag(authedPage, from, { x: from.x, y: from.y + HOUR_PX }, 0)
    await expect.poll(() => startOf(session.token, createdId!), { timeout: 5000 }).toBe(startMs + HOUR)
  })

  test('with the grip on, the bar at the bottom stretches the event at once', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    await withSettings(authedPage, { phone_drag_start: 'handle' })
    const { block, startMs } = await seedEventOnScreen(authedPage, session.token)
    const bar = block.getByTestId('event-grip-bottom')
    await expect(bar).toBeVisible()
    const bb = (await bar.boundingBox())!
    const from = { x: bb.x + bb.width / 2, y: bb.y + bb.height / 2 }
    await fingerDrag(authedPage, from, { x: from.x, y: from.y + HOUR_PX }, 0)
    await authedPage.waitForTimeout(1500)
    const check = await apiContext(session.token)
    const after = (await (await check.get(`/api/events/${createdId}`)).json()).data
    await check.dispose()
    const start = new Date(after.starts_at ?? after.startsAt).getTime()
    const end = new Date(after.ends_at ?? after.endsAt).getTime()
    expect(start, 'the bar stretches the end, the start stays').toBe(startMs)
    expect(end - start, 'the event was 2 hours, the bar was dragged one hour down').toBe(3 * HOUR)
  })

  test('grip only: holding the body no longer lifts the event, and no grip by default', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    await withSettings(authedPage, { phone_drag_start: 'handle' })
    const { box, startMs } = await seedEventOnScreen(authedPage, session.token)
    // On the left half: the grip sits on the right.
    const x = box.x + box.width / 4
    const y = box.y + box.height / 2
    await fingerDrag(authedPage, { x, y }, { x, y: y + HOUR_PX }, 600)
    await authedPage.waitForTimeout(1500)
    expect(await startOf(session.token, createdId!), 'with «handle» a hold on the body must not move it').toBe(startMs)

    // Default (no key): hold only, nothing new drawn on the events.
    await withSettings(authedPage, { phone_drag_start: undefined })
    await authedPage.reload()
    await expect(authedPage.getByTestId('event-block').first()).toBeVisible({ timeout: 15_000 })
    await expect(authedPage.getByTestId('event-grip')).toHaveCount(0)
  })

  // Step 7: a task from the phone's task panel onto the grid. Desktop does this
  // with HTML5 drag-and-drop, which a finger never fires.
  test('holding a task in the task panel and dropping it on a slot schedules it at that hour', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { hour } = await freeWindowOnScreen(authedPage, session.token)
    const ctx = await apiContext(session.token)
    const title = `E2E carry ${Date.now()}`
    const made = await ctx.post('/api/tasks', { data: { title, estimated_minutes: 60, priority: 3 } })
    expect(made.ok(), `task create failed: ${made.status()} ${await made.text()}`).toBe(true)
    const taskId = (await made.json()).data.id as string
    try {
      // The panel lists tasks loaded with the page.
      await authedPage.reload()
      await authedPage.waitForLoadState('networkidle')
      await authedPage.getByRole('button', { name: /^(Open tasks|Открыть задачи)$/ }).click()
      const row = authedPage.locator('[data-task-index]', { hasText: title })
      await expect(row).toBeVisible({ timeout: 10_000 })
      // The panel slides in: a box measured mid-animation put the finger off
      // screen. Wait until the row stops moving.
      let rb = (await row.boundingBox())!
      await expect.poll(async () => {
        const prev = rb.x
        await authedPage.waitForTimeout(100)
        rb = (await row.boundingBox())!
        return rb.x === prev
      }, { timeout: 3000 }).toBe(true)
      const from = { x: rb.x + rb.width / 2, y: rb.y + rb.height / 2 }
      // The reload put the grid back at its initial scroll: aim through the
      // grid's own coordinates, not the ones measured before it.
      const target = await authedPage.evaluate((h) => {
        const track = document.querySelector('[data-hint="calendar.grid"]') as HTMLElement
        let sc: HTMLElement | null = track
        while (sc && getComputedStyle(sc).overflowY !== 'auto') sc = sc.parentElement
        const col = document.querySelector('[data-testid="week-day-header"]')!.parentElement!.parentElement!
        const trackEl = col.querySelector('.relative.select-none') as HTMLElement
        const want = trackEl.getBoundingClientRect().top + h * 44
        const r = sc!.getBoundingClientRect()
        sc!.scrollTop += want - (r.top + r.height / 2)
        return { y: trackEl.getBoundingClientRect().top + h * 44 + 2, x: trackEl.getBoundingClientRect().left + 30 }
      }, hour)
      await fingerDrag(authedPage, from, target, 600)

      const zone = (await (await ctx.get('/api/auth/me')).json()).data.timezone || 'Europe/Moscow'
      await expect.poll(async () => {
        const t = (await (await ctx.get(`/api/tasks/${taskId}`)).json()).data
        if (!t?.event_id) return `not scheduled (${t?.status})`
        const ev = (await (await ctx.get(`/api/events/${t.event_id}`)).json()).data
        return new Intl.DateTimeFormat('en-GB', { timeZone: zone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(ev.starts_at))
      }, { timeout: 8000, message: 'the task should be scheduled where the finger let go' }).toBe(hhmm(hour))
    } finally {
      // The schedule made an event named after the task: remove both.
      const dayStart = localMidnightUtc(((await (await ctx.get('/api/auth/me')).json()).data.timezone) || 'Europe/Moscow', 0)
      const listed = await ctx.get(`/api/events?start=${new Date(dayStart - 24 * HOUR).toISOString()}&end=${new Date(dayStart + 48 * HOUR).toISOString()}`)
      for (const ev of ((await listed.json()).data ?? []) as { id: string; title: string }[]) {
        if (ev.title === title) await ctx.delete(`/api/events/${ev.id}`)
      }
      await ctx.delete(`/api/tasks/${taskId}`)
      await ctx.dispose()
    }
  })

  test('a task held in the panel and let go in place is put nowhere, and the panel comes back', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const ctx = await apiContext(session.token)
    const title = `E2E carry ${Date.now()}`
    const made = await ctx.post('/api/tasks', { data: { title, estimated_minutes: 60, priority: 3 } })
    expect(made.ok()).toBe(true)
    const taskId = (await made.json()).data.id as string
    try {
      await authedPage.goto('/calendar')
      await authedPage.waitForLoadState('networkidle')
      await authedPage.getByRole('button', { name: /^(Open tasks|Открыть задачи)$/ }).click()
      const row = authedPage.locator('[data-task-index]', { hasText: title })
      await expect(row).toBeVisible({ timeout: 10_000 })
      let rb = (await row.boundingBox())!
      await expect.poll(async () => {
        const prev = rb.x
        await authedPage.waitForTimeout(100)
        rb = (await row.boundingBox())!
        return rb.x === prev
      }, { timeout: 3000 }).toBe(true)
      const at = { x: rb.x + rb.width / 2, y: rb.y + rb.height / 2 }
      // Picked up (600 ms > the hold), not moved, let go.
      await fingerDrag(authedPage, at, at, 600)
      await authedPage.waitForTimeout(1500)
      const t = (await (await ctx.get(`/api/tasks/${taskId}`)).json()).data
      expect(t.event_id ?? null, 'a hold let go in place must not schedule the task').toBeNull()
      await expect(row, 'the panel comes back as it was').toBeVisible()
    } finally {
      await ctx.delete(`/api/tasks/${taskId}`)
      await ctx.dispose()
    }
  })

  // Step 9: a held event parked at the right edge of the day flips the day
  // after ~0.8 s and stays in the finger. The column it was lifted from leaves
  // the screen with the flip; a touch keeps targeting the node it started on even
  // after React removes it, so a drag that listened only on `document` would go
  // deaf right here and the drop would never be saved.
  test('a held event parked at the right edge flips to the next day and is dropped there, same time', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    const { box, startMs } = await seedEventOnScreen(authedPage, session.token)
    const x = box.x + box.width / 2
    const y = box.y + box.height / 2
    const width = authedPage.viewportSize()!.width
    const cdp = await authedPage.context().newCDPSession(authedPage)
    const touch = (type: 'touchStart' | 'touchMove' | 'touchEnd', p?: { x: number; y: number }) =>
      cdp.send('Input.dispatchTouchEvent', { type, touchPoints: p ? [p] : [] })
    await touch('touchStart', { x, y })
    await authedPage.waitForTimeout(600)
    const title = authedPage.getByTestId('calendar-period-title')
    const before = await title.textContent()
    const edge = width - 6
    for (let i = 1; i <= 6; i++) {
      await touch('touchMove', { x: x + ((edge - 40 - x) * i) / 6, y })
      await authedPage.waitForTimeout(16)
    }
    // Into the band in one move, so the timer starts at a known moment; wait for
    // the day the user sees to change, then let go. Waiting
    // on the header, not a fixed sleep: CDP round trips stretched a fixed 1.1 s
    // past the second flip at 1.6 s (07.11).
    await touch('touchMove', { x: edge, y })
    const parked = Date.now()
    await expect(title, 'parked at the right edge, the day flips').not.toHaveText(before ?? '', { timeout: 3000 })
    const flippedAfter = Date.now() - parked
    // Let go right there, without another move: the finger sent nothing since the
    // flip, so the drop is right only if the drag re-reads the new day by itself.
    await touch('touchEnd')
    expect(flippedAfter, 'the flip waits about 0.8 s, it is not instant').toBeGreaterThan(500)
    await cdp.detach()

    await expect.poll(() => startOf(session.token, createdId!).then((ms) => ms - startMs), {
      timeout: 8000,
      message: 'the event lands on the next day at the same time',
    }).toBe(24 * HOUR)
  })

  // Step 9, the 3-day view (a phone on its side, 480–767 px). A flip shifts the
  // view by ONE day, so the event steps on instead of jumping three.
  // (A second test, «a drop is not also a swipe», was dropped: it stayed green
  // with the swipe guard sabotaged — the lifted block's node leaves React's tree,
  // so the grid's swipe never sees that touchend. 07.11, test-audit.)
  test.describe('3-day view', () => {
    async function threeDay(authedPage: Page, token: string) {
      await authedPage.setViewportSize({ width: 600, height: 800 })
      const seeded = await seedEventOnScreen(authedPage, token)
      const columns = await authedPage.getByTestId('calendar-period-title').textContent()
      return { ...seeded, columns }
    }
    async function finger(page: Page) {
      const cdp = await page.context().newCDPSession(page)
      const touch = (type: 'touchStart' | 'touchMove' | 'touchEnd', p?: { x: number; y: number }) =>
        cdp.send('Input.dispatchTouchEvent', { type, touchPoints: p ? [p] : [] })
      return { touch, done: () => cdp.detach() }
    }

    test('parked at the right edge, the view moves on by one day and the event lands in its last column', async ({ authedPage, session }) => {
      test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
      const { box, startMs, columns } = await threeDay(authedPage, session.token)
      const x = box.x + box.width / 2
      const y = box.y + box.height / 2
      const { touch, done } = await finger(authedPage)
      await touch('touchStart', { x, y })
      await authedPage.waitForTimeout(600)
      await touch('touchMove', { x: x + 20, y })
      await touch('touchMove', { x: 600 - 6, y })
      const title = authedPage.getByTestId('calendar-period-title')
      await expect(title).not.toHaveText(columns ?? '', { timeout: 3000 })
      await touch('touchEnd')
      await done()
      // Today was the first of three columns; one flip makes the last column
      // today + 3. Three flips' worth (a whole-page jump) would be today + 5.
      await expect.poll(() => startOf(session.token, createdId!).then((ms) => (ms - startMs) / (24 * HOUR)), {
        timeout: 8000, message: 'days moved',
      }).toBe(3)
    })
  })

  // Step 8: the phone month («month and day», split). An event in the chosen
  // day's list, held and dragged onto another day's cell, moves there and keeps
  // its time. The month's mouse drag returns early on pointerType touch.
  test('in the phone month, an event held in the day list and dropped on another day moves there, same time', async ({ authedPage, session }) => {
    test.skip(test.info().project.name !== 'mobile', 'touch drag is a phone gesture')
    await withSettings(authedPage, { phone_header_variant: 'row', phone_month_variant: 'split' })
    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const zone = me.timezone || 'Europe/Moscow'
    const dayStart = localMidnightUtc(zone, 0)
    const ymd = (ms: number) => new Intl.DateTimeFormat('en-CA', { timeZone: zone }).format(new Date(ms))
    const today = ymd(dayStart + 12 * HOUR)
    // Another day of THIS month: tomorrow, or yesterday on a month's last day
    // (a fixed «tomorrow» went red by itself on 31.10).
    const tomorrow = ymd(dayStart + 36 * HOUR)
    const target = tomorrow.slice(0, 7) === today.slice(0, 7) ? tomorrow : ymd(dayStart - 12 * HOUR)
    const title = `E2E month drag ${Date.now()}`
    const startMs = dayStart + 10 * HOUR
    const made = await ctx.post('/api/events', {
      data: { title, starts_at: new Date(startMs).toISOString(), ends_at: new Date(startMs + HOUR).toISOString(), all_day: false },
    })
    expect(made.ok()).toBe(true)
    createdId = (await made.json()).data.id
    try {
      await authedPage.goto('/calendar')
      await authedPage.getByTestId('view-month').click()
      const list = authedPage.getByTestId('month-day-list')
      await expect(list).toBeVisible({ timeout: 15_000 })
      const item = list.locator('li', { hasText: title })
      await expect(item).toBeVisible({ timeout: 10_000 })
      const ib = (await item.boundingBox())!
      const cell = authedPage.locator(`[data-testid="month-day"][data-day="${target}"]`)
      const cb = (await cell.boundingBox())!
      await fingerDrag(
        authedPage,
        { x: ib.x + ib.width / 2, y: ib.y + ib.height / 2 },
        { x: cb.x + cb.width / 2, y: cb.y + cb.height / 2 },
        600,
      )
      await expect.poll(async () => {
        const ev = (await (await ctx.get(`/api/events/${createdId}`)).json()).data
        const at = new Date(ev.starts_at)
        return `${ymd(at.getTime())} ${new Intl.DateTimeFormat('en-GB', { timeZone: zone, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(at)}`
      }, { timeout: 8000, message: 'the event moves to the day it was dropped on, at the same time' }).toBe(`${target} 10:00`)
    } finally {
      await ctx.dispose()
    }
  })
})
