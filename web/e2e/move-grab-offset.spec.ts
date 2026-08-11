import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc, localWeekday } from './fixtures/localTime'

/**
 * Drag plan step 4 — a move must keep its grip on the point that was grabbed.
 *
 * The commit used to be `targetDay + cursorMinute`, which puts the event's
 * START wherever the cursor is. Grabbing a block by its middle therefore yanked
 * it upwards by the grab offset the instant the drag began — for a two-hour
 * event, a full hour of jump before the pointer had moved at all.
 *
 * Unit tests cover the arithmetic (moveCoords, dragHandlers). What they cannot
 * see is whether the grab offset reaches the hook from the mousedown handler,
 * which is exactly the class of defect MD2 was
 * (`learning-md2-lived-in-untested-producers`). Asserted against the API.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const HOUR_PX = 44
const HOUR = 3600 * 1000

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('move keeps its grab offset', () => {
  let createdId: string | undefined
  let token: string | undefined

  test.afterEach(async () => {
    if (createdId && token) {
      const ctx = await apiContext(token)
      await ctx.delete(`/api/events/${createdId}`)
      await ctx.dispose()
      createdId = undefined
    }
  })

  test('grabbing a block by its middle moves it by the cursor delta, not to the cursor', async ({
    authedPage,
    session,
  }) => {
    test.skip(test.info().project.name === 'mobile', 'the single-day mobile view is covered separately')
    token = session.token

    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'

    // Same Sunday guard as crossday-resize: keep a column to the right free in
    // case the layout shifts, and stay in the week the calendar opens on.
    const dayOffset = localWeekday(timeZone) === 0 ? -1 : 0
    const dayStart = localMidnightUtc(timeZone, dayOffset)

    // Two hours tall, so its middle is a comfortable grab point well clear of
    // both resize handles (8px each).
    const startMs = dayStart + 3 * HOUR
    const endMs = dayStart + 5 * HOUR
    const title = `E2E move ${Date.now()}`
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

    const box = await block.boundingBox()
    expect(box, 'the event block must be on screen').not.toBeNull()

    // Dead centre: one hour into a two-hour event, and far from either handle.
    const grabX = box!.x + box!.width / 2
    const grabY = box!.y + box!.height / 2

    await authedPage.mouse.move(grabX, grabY)
    await authedPage.mouse.down()
    // Past the 5px threshold, then exactly one hour down.
    await authedPage.mouse.move(grabX, grabY + 10, { steps: 3 })
    await authedPage.mouse.move(grabX, grabY + HOUR_PX, { steps: 10 })
    await authedPage.mouse.up()

    await expect(
      authedPage.getByRole('dialog'),
      'the drag must have landed on our own one-off event'
    ).toBeHidden()

    await expect
      .poll(
        async () => {
          const check = await apiContext(session.token)
          const res = await check.get(`/api/events/${createdId}`)
          const body = await res.json()
          await check.dispose()
          return new Date(body.data.starts_at ?? body.data.startsAt).getTime()
        },
        { timeout: 15_000, message: 'the move should reach the API' }
      )
      .not.toBe(startMs)

    const verify = await apiContext(session.token)
    const after = (await (await verify.get(`/api/events/${createdId}`)).json()).data
    await verify.dispose()

    const newStart = new Date(after.starts_at ?? after.startsAt).getTime()
    const newEnd = new Date(after.ends_at ?? after.endsAt).getTime()

    // THE ASSERTION. Dragged down one hour, so 03:00 → 04:00. The old path put
    // the start at the cursor, which sat one hour into the block: 05:00.
    expect(newStart - startMs, 'the event should follow the cursor delta, not jump to it').toBe(HOUR)
    expect(newEnd - newStart, 'a move must never change the duration').toBe(endMs - startMs)
  })
})
