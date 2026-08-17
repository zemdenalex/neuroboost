import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc, localWeekday } from './fixtures/localTime'

/**
 * MD1 — a resize must follow the cursor into the neighbouring day column.
 *
 * Until now `cursorMs` was rebuilt on the column the drag STARTED on, so
 * dragging an end sideways changed only the time-of-day and never the date: the
 * end could not leave its own day. Three comments in the WeekGrid folder said
 * this was deliberate, because "the resize ghost renders on the start column
 * only" — a claim that stopped being true in the same commit that wrote it
 * (`aec55e3`), which is what `GhostPreview.test.ts` now pins down.
 *
 * This is the check the unit tests cannot make: that the hook passes the
 * cursor's column through, and that what the database ends up holding matches
 * what the ghost drew. The assertion is against the API, not the DOM.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const HOUR = 3600 * 1000

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('cross-day resize', () => {
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

  test('dragging the end into the next column moves the end to the next day', async ({
    authedPage,
    session,
  }) => {
    // Same reason as multiday-resize: at 375px the calendar shows one day, so
    // there is no neighbouring column to drag into. Cross-day resize on mobile
    // is NOT covered.
    test.skip(test.info().project.name === 'mobile', 'the mobile calendar shows one day at a time')
    token = session.token

    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'

    // The event needs a column to its RIGHT. The week runs Mon…Sun, so every day
    // has one except Sunday — on a Sunday, put the event on Saturday instead,
    // which is still inside the week the calendar opens on.
    const dayOffset = localWeekday(timeZone) === 0 ? -1 : 0
    const dayStart = localMidnightUtc(timeZone, dayOffset)

    // 04:00–05:00 local: early enough to be clear of a working day's events, so
    // the block keeps the full column width and its handle is not overlapped by
    // a neighbour sharing the lane.
    const startMs = dayStart + 4 * HOUR
    const endMs = dayStart + 5 * HOUR
    const title = `E2E crossday ${Date.now()}`
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

    // Column width the way the hook computes it: the grid container's width
    // divided by the number of visible days. Reading it from the DOM rather than
    // assuming keeps the test honest if the layout changes.
    const grid = authedPage.locator('[data-hint="calendar.grid"] > div.grid')
    const gridBox = await grid.boundingBox()
    expect(gridBox, 'the week grid must be on screen').not.toBeNull()
    const dayWidth = gridBox!.width / 7

    const handle = block.locator('div.cursor-ns-resize').last()
    const handleBox = await handle.boundingBox()
    expect(handleBox, 'the event must expose a bottom resize handle').not.toBeNull()

    const grabX = handleBox!.x + handleBox!.width / 2
    const grabY = handleBox!.y + handleBox!.height / 2

    // Straight sideways into the middle of the next column, holding Y so the
    // time-of-day is unchanged and the only thing that can move is the date.
    const dropX = grabX + dayWidth
    expect(dropX, 'the next column must be within the grid').toBeLessThan(gridBox!.x + gridBox!.width)

    await authedPage.mouse.move(grabX, grabY)
    await authedPage.mouse.down()
    await authedPage.mouse.move(dropX, grabY, { steps: 12 })
    await authedPage.mouse.up()

    // A dialog here means the drag landed on a recurring event that is not ours.
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
          return new Date(body.data.ends_at ?? body.data.endsAt).getTime()
        },
        { timeout: 15_000, message: 'the resize should reach the API' }
      )
      // Before MD1 this stayed at endMs: the end could not leave its own day.
      .toBeGreaterThan(startMs + 20 * HOUR)

    const verify = await apiContext(session.token)
    const after = (await (await verify.get(`/api/events/${createdId}`)).json()).data
    await verify.dispose()

    const newStart = new Date(after.starts_at ?? after.startsAt).getTime()
    const newEnd = new Date(after.ends_at ?? after.endsAt).getTime()

    expect(newStart, 'the anchored start must not move').toBe(startMs)
    // Y was held, so the end should land at the same time on the following day:
    // 04:00 → next-day 05:00 is 25 hours. Allow a slot of slack for snapping.
    expect(newEnd - newStart, 'the end should now sit on the next day').toBeGreaterThan(24 * HOUR)
    expect(newEnd - newStart, 'and only one day further, not more').toBeLessThan(26 * HOUR)
  })
})
