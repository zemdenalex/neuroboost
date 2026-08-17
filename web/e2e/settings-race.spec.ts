import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'

/**
 * A settings change must survive the arrival of an earlier save's response.
 *
 * 🔴 The defect this reproduces (docs/audit-2026-08-15.md §3). Nine of the
 * thirteen settings sections rebuild their local state from an effect keyed on
 * the `user` OBJECT's identity, and `updateSettings` calls setUser on every
 * save — so every server response overwrites local state, including a change
 * the user made while the previous save was still in flight. On its own that
 * reads as a 150ms flicker and self-corrects. It stops being cosmetic the
 * moment a third change is made from the reverted state: the next patch is
 * built from what the server sent back, and the middle change is gone for good.
 *
 * Made deterministic by delaying the PATCH response rather than by racing real
 * network timing. A timing-sensitive version of this test would be flaky, and a
 * flaky guard is worse than none — it gets retried until green and then
 * believed.
 *
 * Written BEFORE the fix and confirmed to fail against deployed staging, so
 * that the green run afterwards means something. Without that ordering this
 * would only be a test that happens to pass.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

/** The whole settings blob, so the account is left exactly as it was found. */
async function readSettings(token: string): Promise<unknown> {
  const ctx = await playwrightRequest.newContext()
  try {
    const res = await ctx.get(`${API_BASE}/api/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok()) throw new Error(`could not read settings: ${res.status()}`)
    const body = await res.json()
    return (body.data ?? body).settings ?? {}
  } finally {
    await ctx.dispose()
  }
}

async function writeSettings(token: string, settings: unknown): Promise<void> {
  const ctx = await playwrightRequest.newContext()
  try {
    const res = await ctx.patch(`${API_BASE}/api/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { settings },
    })
    // Loud: a silent failure here leaves the account in a state the next run
    // starts from, and the next failure would look unrelated.
    if (!res.ok()) throw new Error(`could not restore settings: ${res.status()} ${await res.text()}`)
  } finally {
    await ctx.dispose()
  }
}

// One viewport only. Both projects run fully in parallel against the SAME test
// account, and this spec writes settings and restores them — two copies would
// overwrite each other's restore and fail for a reason that has nothing to do
// with what is being tested.
test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'writes shared account settings; one viewport only')
})

test('a change made during a save is not overwritten when that save answers', async ({
  authedPage,
  session,
}) => {
  const original = await readSettings(session.token)

  try {
    await authedPage.goto('/settings')
    await authedPage.waitForLoadState('networkidle')

    const quietHours = authedPage
      .locator('label', { hasText: 'Respect quiet hours' })
      .locator('input[type=checkbox]')
    const digestAt = authedPage.locator('#rem-digest-at')

    await expect(quietHours).toBeVisible()
    await expect(digestAt).toBeVisible()
    // The digest time input is disabled while the digest is off, and a disabled
    // input cannot carry the second change. Assert rather than assume: without
    // this the fill below would throw and the failure would read as a timeout
    // on an unrelated element.
    await expect(digestAt).toBeEnabled()

    // Hold every settings save open long enough that its response is guaranteed
    // to arrive after the second change, instead of hoping it does.
    await authedPage.route('**/api/auth/me', async route => {
      if (route.request().method() !== 'PATCH') return route.continue()
      await new Promise(resolve => setTimeout(resolve, 1500))
      await route.continue()
    })

    // Change A — starts a save that will not answer for 1.5s.
    await quietHours.click()
    await authedPage.waitForTimeout(500) // past the 300ms debounce: A is in flight

    // Change B — local only, nothing has saved it yet.
    await digestAt.fill('07:11')
    await expect(digestAt).toHaveValue('07:11')

    // A's response lands here and, today, resets the section from the server.
    await authedPage.waitForTimeout(1800)

    expect(
      await digestAt.inputValue(),
      'the digest time typed while the previous save was in flight was overwritten by that save\'s response',
    ).toBe('07:11')
  } finally {
    await authedPage.unroute('**/api/auth/me')
    // Let anything still queued finish before putting the account back, or the
    // restore would be overwritten by a late save.
    await authedPage.waitForTimeout(2000)
    await writeSettings(session.token, original)
  }
})
