import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * docs/tasks-web-cleanup.md 4.11: the circle in a Tasks row wrote status=DONE
 * for every task, so ticking «зарядка, каждый день» switched the whole series
 * off. A tick on a running series answers today's day of it; the series stays
 * TODO. Asserted through the API, not the pixels.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('tick on a repeating task', () => {
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

  test('closes today, not the series', async ({ authedPage, session }) => {
    token = session.token
    const ctx = await apiContext(session.token)
    const title = `E2E series tick ${Date.now()}`
    const created = await ctx.post('/api/tasks', {
      data: { title, priority: 3, rrule: 'FREQ=DAILY', contexts: [], tags: [] },
    })
    expect(created.ok(), `create failed: ${created.status()} ${await created.text()}`).toBe(true)
    createdId = (await created.json()).data.id

    await authedPage.goto('/tasks')
    const row = authedPage.locator('[id^="task-"]').filter({ hasText: title }).first()
    await expect(row).toBeVisible({ timeout: 15_000 })

    const answered = authedPage.waitForResponse(
      (r) => r.url().includes(`/api/tasks/${createdId}/occurrences`) && r.request().method() === 'POST',
    )
    await row.getByTestId('task-tick').click()
    expect((await answered).ok()).toBe(true)

    const list = await ctx.get('/api/tasks')
    const task = ((await list.json()).data as Array<{ id: string; status: string; occurrence_state?: string }>).find(
      (t) => t.id === createdId,
    )
    await ctx.dispose()
    expect(task?.status, 'the series must keep running').toBe('TODO')
    expect(task?.occurrence_state, "today's day is the one answered").toBe('done')
  })
})
