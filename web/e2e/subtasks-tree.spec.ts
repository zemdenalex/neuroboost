import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * Subtasks on the web task list (N2 of pass 3). A subtask made anywhere, the
 * bot included, is drawn right under its parent, indented, and the parent
 * shows its progress. Created through the API and deleted afterwards.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

test.describe('subtasks tree', () => {
  const created: string[] = []
  let token: string | undefined

  test.afterEach(async () => {
    if (!token) return
    const ctx = await apiContext(token)
    for (const id of created.reverse()) await ctx.delete(`/api/tasks/${id}`)
    await ctx.dispose()
    created.length = 0
  })

  test('a subtask sits under its parent, and the parent counts it', async ({ authedPage, session }) => {
    token = session.token
    const ctx = await apiContext(session.token)
    const stamp = Date.now()
    const make = async (data: Record<string, unknown>) => {
      const res = await ctx.post('/api/tasks', { data: { contexts: [], tags: [], ...data } })
      expect(res.ok(), `create failed: ${res.status()} ${await res.text()}`).toBe(true)
      const id = (await res.json()).data.id as string
      created.push(id)
      return id
    }
    // Different priorities on purpose: the subtask still follows its parent.
    const parent = await make({ title: `E2E parent ${stamp}`, priority: 3 })
    const child = await make({ title: `E2E child ${stamp}`, priority: 1, parent_id: parent })
    await ctx.dispose()

    await authedPage.goto('/tasks')
    const parentRow = authedPage.locator(`#task-${parent}`)
    await expect(parentRow).toBeVisible({ timeout: 15_000 })
    await expect(parentRow.getByTestId('subtask-progress')).toHaveText('✓ 0/1')

    // The child is the row right after the parent, and indented.
    const next = parentRow.locator('xpath=following-sibling::*[1]')
    await expect(next).toHaveAttribute('id', `task-${child}`)
    await expect(next.getByTestId('subtask-indent')).toHaveCount(1)
    await expect(parentRow.getByTestId('subtask-indent')).toHaveCount(0)
  })
})
