import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * A task's reminders must be editable after the task exists.
 *
 * Until now the ONLY place reminder offsets could be set was quick-add's second
 * level. The task editor on /tasks never sent the field, and `Task` did not even
 * declare it, so the editor could not read back what quick-add had written: a
 * task created without reminders could never be given any, and one created with
 * them could not be changed.
 *
 * The language is pinned to English so the control can be addressed by its
 * accessible names — the account's own locale would otherwise decide whether
 * this test can find anything.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('task reminders', () => {
  let createdId: string | undefined
  let token: string | undefined

  test.afterEach(async () => {
    if (createdId && token) {
      const ctx = await apiContext(token)
      await ctx.delete(`/api/tasks/${createdId}`)
      await ctx.dispose()
      createdId = undefined
    }
  })

  test('an offset added in the task editor reaches the API', async ({ authedPage, session }) => {
    token = session.token

    await authedPage.addInitScript(() => {
      window.localStorage.setItem('neuroboost-locale', 'en')
    })

    const ctx = await apiContext(session.token)
    // A due date is required: reminders count back from it, and without one the
    // control renders its disabled hint instead.
    const due = new Date(Date.now() + 48 * 3600 * 1000).toISOString()
    const title = `E2E reminders ${Date.now()}`
    const created = await ctx.post('/api/tasks', {
      data: { title, priority: 3, due_date: due, reminder_offsets: [], contexts: [], tags: [] },
    })
    expect(created.ok(), `create failed: ${created.status()} ${await created.text()}`).toBe(true)
    createdId = (await created.json()).data.id
    await ctx.dispose()

    await authedPage.goto('/tasks')
    await authedPage.waitForLoadState('networkidle')

    // Open the editor for our task.
    const row = authedPage.getByText(title, { exact: false }).first()
    await expect(row).toBeVisible({ timeout: 15_000 })
    await row.click()

    const offsetInput = authedPage.getByLabel('New reminder offset')
    await expect(
      offsetInput,
      'the reminder control must be present and enabled — a task with a due date has something to count back from'
    ).toBeVisible({ timeout: 10_000 })

    await offsetInput.fill('30')
    await authedPage.getByRole('button', { name: 'Add' }).click()

    // The chip appearing proves the control accepted it; the API check below
    // proves the editor actually sends it.
    await expect(authedPage.getByText('30 minutes before')).toBeVisible()

    await authedPage.getByRole('button', { name: /save/i }).click()

    await expect
      .poll(
        async () => {
          const check = await apiContext(session.token)
          const res = await check.get(`/api/tasks/${createdId}`)
          const body = await res.json()
          await check.dispose()
          return JSON.stringify(body.data.reminder_offsets ?? [])
        },
        { timeout: 15_000, message: 'the offset should reach the API' }
      )
      .toBe('[30]')
  })
})
