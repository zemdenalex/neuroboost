import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

/**
 * Day tasks on the web (spec docs/superpowers/specs/2026-09-24-web-day-tasks-design.md).
 *
 * Read-only on purpose: settings-race.spec.ts writes and restores the same
 * account's settings in parallel, and a spec that flipped day_tasks_enabled
 * would race with it. What this proves is that the page and its entrance
 * exist on the deployed build and talk to the real API. The rules are unit
 * tested in src/lib/dayTasks.
 */

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough for a read-only check')
})

test('the day tasks page opens and shows today in one of its states', async ({ authedPage }) => {
  await authedPage.goto('/day-tasks')
  // 15s: a local vite compiles the lazy page on first visit.
  await expect(authedPage.getByRole('heading', { name: /Day tasks/ })).toBeVisible({ timeout: 15_000 })

  // Whatever state the test account is in, the page must land in exactly one
  // of the four the spec names, not in an error or an endless «Loading».
  const states = authedPage
    .getByTestId('day-progress')
    .or(authedPage.getByRole('button', { name: /Take it|Take an empty day/ }))
    .or(authedPage.getByText('Day tasks are off.'))
  await expect(states.first()).toBeVisible({ timeout: 15_000 })
  await expect(authedPage.getByRole('alert')).toHaveCount(0)
})

test('day tasks have a way in from the header while they are on', async ({ authedPage, session }) => {
  // The account may have them switched off (it is a real person's staging
  // account): then the entrance must be gone, not present.
  const ctx = await playwrightRequest.newContext()
  const off = await (async () => {
    try {
      const res = await ctx.get(`${API_BASE}/api/auth/me`, { headers: { Authorization: `Bearer ${session.token}` } })
      expect(res.ok()).toBe(true)
      const body = await res.json()
      return (body.data ?? body).settings?.day_tasks_enabled === false
    } finally {
      await ctx.dispose()
    }
  })()
  await authedPage.goto('/home')
  const link = authedPage.getByRole('link', { name: /Day tasks/ })
  if (off) await expect(link).toHaveCount(0)
  else await expect(link.first()).toBeVisible({ timeout: 15_000 })
})
