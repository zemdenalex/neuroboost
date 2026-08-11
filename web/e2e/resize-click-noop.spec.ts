import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc, localWeekday } from './fixtures/localTime'

/**
 * Drag plan step 6 — a click on a resize handle is a click, not a resize.
 *
 * Resize had no movement threshold (move had 5px). Pressing a handle and
 * releasing without moving therefore committed a PATCH with unchanged times.
 * Harmless to the data, but on a repeating event it raised the "this
 * occurrence / all occurrences" dialog out of nowhere.
 *
 * 🔴 This test asserts that something does NOT happen, which is the easiest
 * kind of test to pass for the wrong reason — a broken selector, a page that
 * never loaded, an event that was never created would all "pass". So the same
 * test carries a positive control: after the click proves inert, a real drag on
 * the same handle must move the event. If the control fails, the negative half
 * proved nothing.
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

test.describe('clicking a resize handle', () => {
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

  test('commits nothing, while a real drag on the same handle still does', async ({
    authedPage,
    session,
  }) => {
    token = session.token

    const ctx = await apiContext(session.token)
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone = me.timezone || 'Europe/Moscow'

    const dayOffset = localWeekday(timeZone) === 0 ? -1 : 0
    const dayStart = localMidnightUtc(timeZone, dayOffset)
    const startMs = dayStart + 3 * HOUR
    const endMs = dayStart + 5 * HOUR

    const title = `E2E clicknoop ${Date.now()}`
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

    const handle = block.locator('div.cursor-ns-resize').last()
    const handleBox = await handle.boundingBox()
    expect(handleBox, 'the event must expose a bottom resize handle').not.toBeNull()
    const x = handleBox!.x + handleBox!.width / 2
    const y = handleBox!.y + handleBox!.height / 2

    // ---- the negative: press and release without moving -------------------
    await authedPage.mouse.move(x, y)
    await authedPage.mouse.down()
    await authedPage.mouse.up()

    // Give any PATCH time to land before claiming none did.
    await authedPage.waitForTimeout(2_000)

    const afterClick = await apiContext(session.token)
    const clicked = (await (await afterClick.get(`/api/events/${createdId}`)).json()).data
    await afterClick.dispose()

    expect(new Date(clicked.starts_at ?? clicked.startsAt).getTime(), 'a click must not move the start').toBe(startMs)
    expect(new Date(clicked.ends_at ?? clicked.endsAt).getTime(), 'a click must not move the end').toBe(endMs)

    // ---- the positive control: the same handle, actually dragged ----------
    await authedPage.mouse.move(x, y)
    await authedPage.mouse.down()
    await authedPage.mouse.move(x, y + HOUR_PX, { steps: 10 })
    await authedPage.mouse.up()

    await expect
      .poll(
        async () => {
          const check = await apiContext(session.token)
          const res = await check.get(`/api/events/${createdId}`)
          const body = await res.json()
          await check.dispose()
          return new Date(body.data.ends_at ?? body.data.endsAt).getTime()
        },
        {
          timeout: 15_000,
          message: 'CONTROL FAILED — a real drag did not commit either, so the click assertion above proved nothing',
        }
      )
      .toBeGreaterThan(endMs)
  })
})
