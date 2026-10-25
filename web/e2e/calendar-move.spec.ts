import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'

/**
 * Moving an event to another calendar, through the editor a person actually uses.
 *
 * 🔴 Why this exists. Denis reported on 10.09 that an event "saves" and does not
 * move, and reading the code produced two confident wrong answers before a
 * browser produced the right one: `handleSave` in `useEditorForm.ts` omitted
 * `calendarId` from its `useCallback` deps, so changing ONLY the calendar left
 * the callback closed over a stale value. It computed `calendarChanged = false`,
 * deleted the field from the body, and sent a PATCH the server had nothing to do
 * with. HTTP 200, nothing moved.
 *
 * Everything that could have caught it, didn't:
 * - the server is fine, and answers 200 either way — proved separately over
 *   plain HTTP before this test existed;
 * - `api/index.ts` maps `calendarId` → `calendar_id` correctly, and a unit test
 *   on that mapping passes while the field never arrives;
 * - ESLint named the missing dependency, by file and line, on every CI run.
 *   Warnings do not fail the build, so it was counted and never read.
 *
 * The only control that could have failed for the right reason is this one: a
 * real editor, a change to the calendar and NOTHING else, and an assertion
 * against the API rather than against the screen.
 *
 * 🔴 The "and nothing else" is the whole test. Changing the calendar together
 * with the title passes even with the defect present — the title IS in the deps,
 * so the callback is rebuilt and picks the calendar up on the way. A test that
 * edits two fields is the test that could not fail.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.space'

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

/** The editor's save button, in either interface language. */
const SAVE_BUTTON = /^(Save|Сохранить)$/

test.describe('moving an event between calendars', () => {
  let cleanup: (() => Promise<void>) | undefined

  test.afterEach(async () => {
    if (cleanup) {
      await cleanup()
      cleanup = undefined
    }
  })

  test('changing only the calendar actually moves the event', async ({ authedPage, session }) => {
    const ctx = await apiContext(session.token)

    // The picker hides itself when there is only one writable calendar, so the
    // test has to create the second one it is going to select.
    const calendars = (await (await ctx.get('/api/calendars')).json()).data as Array<{
      id: string
      kind: string
      role: string
      status: string
    }>
    const home = calendars.find(c => c.kind === 'personal' && c.role === 'owner')
    expect(home, 'the account has no personal calendar to move the event out of').toBeTruthy()

    const targetResp = await ctx.post('/api/calendars', {
      data: { name: `e2e move target ${Date.now()}`, color: '#ff4444' },
    })
    expect(targetResp.status()).toBe(201)
    const target = (await targetResp.json()).data as { id: string }

    // A plain, non-recurring event: the scope dialog is a separate question and
    // a separate spec. This one is about the field reaching the request at all.
    //
    // 🔴 TODAY, in the account's own zone. The first version put the event 26
    // hours out and failed on the mobile project only: at 375px the grid shows
    // ONE day, so tomorrow's block is not on screen and never becomes findable.
    // Desktop passed, which is exactly how a viewport-shaped test defect hides.
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const timeZone: string = me.timezone || 'Europe/Moscow'
    const title = `e2e calendar move ${Date.now()}`
    const start = new Date(localMidnightUtc(timeZone, 0) + 10 * 3600 * 1000)
    const end = new Date(start.getTime() + 3600 * 1000)

    const evResp = await ctx.post('/api/events', {
      data: {
        title,
        starts_at: start.toISOString(),
        ends_at: end.toISOString(),
        timezone: timeZone,
      },
    })
    expect(evResp.status()).toBe(201)
    const event = (await evResp.json()).data as { id: string; calendar_id: string }
    expect(event.calendar_id).toBe(home!.id)

    cleanup = async () => {
      await ctx.delete(`/api/events/${event.id}`)
      await ctx.delete(`/api/calendars/${target.id}`)
      await ctx.dispose()
    }

    // The event is today, so it is on the first screen in every viewport: the
    // week on desktop, the single day on mobile.
    await authedPage.goto('/calendar')
    await authedPage.reload()

    const block = authedPage.locator(`[title^="${title}"]`).first()
    await expect(block, 'the created event never appeared on the grid').toBeVisible({ timeout: 15000 })
    await block.dblclick()

    const picker = authedPage.locator('#event-calendar')
    await expect(picker, 'the calendar field is not rendered — fewer than two writable calendars?').toBeVisible()
    // The editor must open showing the calendar the event is really in. If this
    // is wrong, the select is lying about the current value and a user "moving"
    // the event to the entry already displayed would produce no change event.
    await expect(picker).toHaveValue(home!.id)

    await picker.selectOption(target.id)
    await expect(picker).toHaveValue(target.id)

    // 🔴 Nothing else is touched. See the note at the top.
    await authedPage.getByRole('button', { name: SAVE_BUTTON }).click()

    // The assertion is against the API, not the grid: a repaint can lag, be
    // optimistic, or be filtered, and none of those tell us whether the move
    // was persisted.
    await expect
      .poll(
        async () => {
          const fresh = await ctx.get(`/api/events?start=${start.toISOString()}&end=${end.toISOString()}`)
          const list = (await fresh.json()).data as Array<{ id: string; calendar_id: string }>
          return list.find(e => e.id === event.id)?.calendar_id
        },
        {
          timeout: 15000,
          message: 'the event never reached the target calendar — the PATCH probably carried no calendar_id',
        },
      )
      .toBe(target.id)
  })
})
