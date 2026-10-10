import { test, expect } from './fixtures/auth'
import type { Page } from '@playwright/test'

/**
 * Settings → work hours: an end at or before the start is said on the spot
 * (the API counts such a day as 0 hours, usersettings/workhours.go Hours).
 * The account is Denis's real one: settings are shown from a mocked read and
 * no PATCH leaves the page.
 */

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
})

async function withWorkHours(page: Page, work_start: string, work_end: string) {
  await page.route('**/api/auth/me', (route) => (route.request().method() === 'PATCH' ? route.abort() : route.fallback()))
  await page.addInitScript(
    (extra) => {
      const orig = window.fetch.bind(window)
      window.fetch = async (input, init) => {
        const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
        const res = await orig(input, init)
        if (!url.includes('/api/auth/me') || (init?.method && init.method !== 'GET')) return res
        const body = await res.clone().json()
        body.data.settings = { ...(body.data.settings ?? {}), ...extra }
        return new Response(JSON.stringify(body), { status: res.status, headers: { 'Content-Type': 'application/json' } })
      }
    },
    { work_start, work_end },
  )
}

test('an end before the start is called out', async ({ authedPage }) => {
  await withWorkHours(authedPage, '18:00', '09:00')
  await authedPage.goto('/settings')
  await expect(authedPage.getByTestId('work-hours-invalid')).toBeVisible({ timeout: 15_000 })
})

test('a normal day says nothing', async ({ authedPage }) => {
  await withWorkHours(authedPage, '09:00', '18:00')
  await authedPage.goto('/settings')
  await expect(authedPage.getByTestId('work-hours-section').locator('input[type="time"]').first()).toHaveValue('09:00', { timeout: 15_000 })
  await expect(authedPage.getByTestId('work-hours-invalid')).toHaveCount(0)
})
