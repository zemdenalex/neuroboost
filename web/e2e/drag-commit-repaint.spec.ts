import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc, localWeekday } from './fixtures/localTime'

/**
 * What does the grid show BETWEEN mouseup and the refetch landing?
 *
 * `Calendar.tsx` says, above its `await loadEvents()`: "the grid has already
 * drawn the event at its dropped position, and only a reload puts it back".
 * Reading the code suggests the opposite — `onUp` calls `setDrag(null)`, so the
 * ghost is gone and the block's position comes only from the `events` array,
 * which still holds the old times until the refetch replaces it.
 *
 * One of those is wrong, and a decision rests on it: whether the cancel branch
 * needs its reload, and whether an optimistic update is a fix or a regression.
 * The same shape of unverified comment cost this session an hour on MD1, so
 * this is an observation, not an argument.
 *
 * The API is deliberately slowed so the in-between state is long enough to
 * measure; without that the window is a few hundred milliseconds.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'
const HOUR_PX = 44
const HOUR = 3600 * 1000
const API_DELAY_MS = 2500

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('what the grid shows between drop and refetch', () => {
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

  test('the block does not snap back to its old position after the drop', async ({
    authedPage,
    session,
  }) => {
    test.skip(test.info().project.name === 'mobile', 'measured against a desktop column layout')
    token = session.token

    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'

    const dayOffset = localWeekday(timeZone) === 0 ? -1 : 0
    const dayStart = localMidnightUtc(timeZone, dayOffset)
    const startMs = dayStart + 3 * HOUR
    const endMs = dayStart + 5 * HOUR

    const title = `E2E repaint ${Date.now()}`
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

    const before = await block.boundingBox()
    expect(before, 'the event block must be on screen').not.toBeNull()

    // Slow every events call from here on, so the window between the drop and
    // the refetch is wide enough to measure. Installed AFTER the initial load,
    // which would otherwise be delayed too.
    await authedPage.route('**/api/events**', async route => {
      await new Promise(resolve => setTimeout(resolve, API_DELAY_MS))
      await route.continue()
    })

    const grabX = before!.x + before!.width / 2
    const grabY = before!.y + before!.height / 2
    await authedPage.mouse.move(grabX, grabY)
    await authedPage.mouse.down()
    await authedPage.mouse.move(grabX, grabY + 10, { steps: 3 })
    await authedPage.mouse.move(grabX, grabY + HOUR_PX, { steps: 10 })
    await authedPage.mouse.up()

    // Measured inside the delay window: the PATCH and the refetch are still in
    // flight, so this is what the user actually looks at after letting go.
    const during = await block.boundingBox()
    expect(during, 'the block must still be rendered while the request is in flight').not.toBeNull()

    const movedBy = during!.y - before!.y
    // eslint-disable-next-line no-console
    console.log(`[observation] before.y=${before!.y} during.y=${during!.y} delta=${movedBy}px (one hour = ${HOUR_PX}px)`)

    // The claim under test: the grid has ALREADY drawn the event where it was
    // dropped. If it has, the block sits an hour lower while the request is in
    // flight. If it has not, delta is 0 and the user sees it snap back — the
    // flicker.
    expect(
      movedBy,
      'between drop and refetch the block should stay where it was dropped, not jump back an hour'
    ).toBeGreaterThan(HOUR_PX / 2)
  })
})
