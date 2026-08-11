import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * Calendar create / rename / delete, driven through the settings UI and
 * exercised against deployed staging.
 *
 * Covers: the calendars section renders with the personal calendar already
 * listed; creating a shared calendar with a unique name makes its row
 * appear; renaming it through the rename controls swaps the displayed name
 * (old name gone, new name present); deleting it removes the row entirely.
 *
 * Deliberately NOT covered: the 409 CALENDAR_NOT_EMPTY refusal. CreateRequest
 * in api-go/internal/events/types.go has no calendar_id field, and
 * createEvent (handlers.go:667) unconditionally resolves to the personal
 * calendar via PersonalIDFor — so nothing reachable from the browser can put
 * an event inside a shared calendar in this slice, meaning a shared calendar
 * cannot be made non-empty from here. That refusal is covered by the Go
 * test TestDeleteRefusesNonEmpty instead; it returns with the next slice
 * once events can target a shared calendar.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

interface CalendarLike {
  id: string
  name: string
}

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('calendar CRUD', () => {
  let createdId: string | undefined
  let token: string | undefined

  // Cleanup by id, not by name: if the test fails after the rename step but
  // before delete, the calendar exists under the renamed name, and deleting
  // by id still finds it. This runs on both pass and failure.
  test.afterEach(async () => {
    if (createdId && token) {
      const ctx = await apiContext(token)
      await ctx.delete(`/api/calendars/${createdId}`)
      await ctx.dispose()
    }
    createdId = undefined
    token = undefined
  })

  test('create, rename and delete a shared calendar through settings', async ({ authedPage, session }) => {
    token = session.token

    // Timestamp-derived so reruns against the same staging account never collide.
    const originalName = `E2E Calendar ${Date.now()}`
    const renamedName = `${originalName} renamed`

    await authedPage.goto('/settings')
    await authedPage.waitForLoadState('networkidle')

    const section = authedPage.getByTestId('calendars-section')
    await expect(section).toBeVisible({ timeout: 15_000 })

    // A fresh account still owns a personal calendar, so the list is never
    // empty before this test creates anything.
    await expect(section.getByTestId('calendar-row').first()).toBeVisible({ timeout: 15_000 })

    // --- create ---
    await section.getByTestId('calendar-create-input').fill(originalName)
    await section.getByTestId('calendar-create-submit').click()

    const createdRow = section.locator(
      `[data-testid="calendar-row"][data-calendar-name="${originalName}"]`
    )
    await expect(createdRow, 'row for the newly created calendar should appear').toBeVisible({
      timeout: 15_000,
    })

    // The UI never surfaces the calendar's id, and cleanup needs it to be
    // resilient to a later rename — fetch it from the API instead of
    // asserting on list length.
    const listCtx = await apiContext(session.token)
    const listRes = await listCtx.get('/api/calendars')
    const calendars: CalendarLike[] = (await listRes.json()).data ?? []
    await listCtx.dispose()
    const created = calendars.find((c) => c.name === originalName)
    expect(created, 'created calendar should be present in the API list').toBeTruthy()
    createdId = created?.id

    // --- rename ---
    await createdRow.getByTestId('calendar-rename').click()
    const renameInput = createdRow.getByTestId('calendar-rename-input')
    await expect(renameInput).toBeVisible()
    await renameInput.fill(renamedName)
    await createdRow.getByTestId('calendar-rename-save').click()

    const renamedRow = section.locator(
      `[data-testid="calendar-row"][data-calendar-name="${renamedName}"]`
    )
    await expect(renamedRow, 'the new name should be visible after rename').toBeVisible({
      timeout: 15_000,
    })
    await expect(
      section.locator(`[data-testid="calendar-row"][data-calendar-name="${originalName}"]`),
      'the old name should no longer be present after rename'
    ).toHaveCount(0)

    // --- delete ---
    await renamedRow.getByTestId('calendar-delete').click()
    await expect(
      section.locator(`[data-testid="calendar-row"][data-calendar-name="${renamedName}"]`),
      'the row should disappear entirely after delete, not just relabel'
    ).toHaveCount(0, { timeout: 15_000 })

    // Deleted through the UI itself — nothing left for afterEach to clean up.
    createdId = undefined
  })
})
