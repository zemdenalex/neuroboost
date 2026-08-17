import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * MD2, dragged with a real mouse.
 *
 * The unit tests around resizeCoords prove the mapping; they cannot prove the
 * hook is wired to it, and "green unit tests cannot see the screen" is the
 * reason this harness exists at all. Here the assertion is made against the
 * API — what the database ended up holding — rather than against the DOM, so
 * a ghost that merely looked right cannot pass it.
 *
 * Before the fix this event collapsed onto the grabbed day.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const HOUR_PX = 44

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

import { midnightInsideRenderedWeek } from './fixtures/localTime'

test.describe('multi-day resize', () => {
  // Desktop only, and not for convenience: at 375px the calendar shows a single
  // day, so the second segment is not on screen to be grabbed — the run there
  // failed with "Received: 1" segment. Multi-day resize on mobile is therefore
  // NOT covered by this suite, and should not be assumed to work: the mobile
  // calendar views (MV1) do not exist yet either.
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

  test('dragging the end of a night-spanning event keeps its start on the previous day', async ({
    authedPage,
    session,
  }) => {
    test.skip(test.info().project.name === 'mobile', 'the mobile calendar shows one day at a time')
    token = session.token

    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'

    // Two hours before local midnight to one hour after it: two segments, and
    // the second starts at the very top of its column, so its bottom handle is
    // on screen without scrolling.
    //
    // The window stops at 01:00 deliberately. The account carries a daily event
    // at 01:15, and an overlapping block shares the column in a narrower lane —
    // the first version of this test reached across into it and started a drag
    // on somebody else's repeating event instead.
    const midnight = midnightInsideRenderedWeek(timeZone)
    const startMs = midnight - 2 * 3600 * 1000
    const endMs = midnight + 1 * 3600 * 1000
    const title = `E2E multiday ${Date.now()}`
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

    // Segments render one block per day; the last one carries the bottom handle.
    const blocks = authedPage.locator(`[title^="${title}"]`)
    await expect(blocks.first()).toBeVisible({ timeout: 15_000 })
    expect(await blocks.count(), 'a night-spanning event should render as two segments').toBeGreaterThan(1)

    // Target the handle element itself rather than guessing at the block's
    // bottom edge. A first attempt used `box.height - 2` and landed on the
    // neighbouring event instead, which opened the recurring-scope dialog — the
    // screenshot was the only thing that said so.
    const lastSegment = blocks.last()
    const handle = lastSegment.locator('div.cursor-ns-resize').last()
    const box = await handle.boundingBox()
    expect(box, 'the last segment must expose a bottom resize handle on screen').not.toBeNull()

    const grabX = box!.x + box!.width / 2
    const grabY = box!.y + box!.height / 2
    await authedPage.mouse.move(grabX, grabY)
    await authedPage.mouse.down()
    await authedPage.mouse.move(grabX, grabY + HOUR_PX, { steps: 10 })
    await authedPage.mouse.up()

    // If we grabbed somebody else's repeating event, this dialog appears and the
    // assertions below would be measuring the wrong thing.
    await expect(
      authedPage.getByRole('dialog'),
      'the drag must have landed on our own one-off event'
    ).toBeHidden()

    // Assert on stored state, not on pixels.
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
      .toBeGreaterThan(endMs)

    const verify = await apiContext(session.token)
    const after = (await (await verify.get(`/api/events/${createdId}`)).json()).data
    await verify.dispose()

    const newStart = new Date(after.starts_at ?? after.startsAt).getTime()
    const newEnd = new Date(after.ends_at ?? after.endsAt).getTime()

    // The collapse this fixes: the start used to be rebuilt on the grabbed day.
    expect(newStart, 'the untouched start must not move').toBe(startMs)
    expect(newEnd - newStart, 'the event must still cross midnight').toBeGreaterThan(2 * 3600 * 1000)
  })
})
