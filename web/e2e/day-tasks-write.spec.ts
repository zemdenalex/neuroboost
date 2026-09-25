import { test, expect } from './fixtures/auth'
import type { Page, Route } from '@playwright/test'

/**
 * Day tasks on the web: taking a day and ticking a task (docs/agents/queue.md,
 * «e2e для /day-tasks сейчас только read-only»).
 *
 * 🔴 The e2e account is Denis's real staging account, so nothing here reaches
 * the server: the day, the proposal and the task list are answered by the test,
 * every write is caught and answered here, and any other write is aborted.
 * What this proves is the page's side of the door: what it sends when you
 * press «Take it» and the tick, and that it shows the answer.
 */

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
})

const TASKS = [
  { id: 't-1', title: 'E2E позвонить в банк', status: 'TODO', priority: 2 },
  { id: 't-2', title: 'E2E купить хлеб', status: 'TODO', priority: 3 },
].map((t) => ({ ...t, tags: [], contexts: [], created_at: '2026-09-01T00:00:00Z', updated_at: '2026-09-01T00:00:00Z' }))

const json = (route: Route, data: unknown, status = 200) => route.fulfill({ status, json: { data } })

async function mockDayTasks(page: Page) {
  const sent: { confirm?: { day: string; task_ids: string[] }; patched?: { id: string; body: unknown } } = {}
  let day = ''
  let confirmed = false
  let done = false
  const state = () => ({
    day,
    target: 5,
    confirmed,
    items: confirmed ? TASKS.map((t) => ({ task_id: t.id, title: t.title, done: t.id === 't-1' && done })) : [],
    done: done ? 1 : 0,
    level: done ? 1 : 0,
    before_start: false,
  })

  // Registered first, so matched last: any write nothing below answered.
  await page.route('**/api/**', (route) => (route.request().method() === 'GET' ? route.fallback() : route.abort()))
  // Day tasks on for this page, whatever the account says; never saved.
  await page.addInitScript(() => {
    const orig = window.fetch.bind(window)
    window.fetch = async (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      const res = await orig(input, init)
      if (!url.includes('/api/auth/me') || (init?.method && init.method !== 'GET')) return res
      const body = await res.clone().json()
      body.data.settings = { ...(body.data.settings ?? {}), day_tasks_enabled: true }
      return new Response(JSON.stringify(body), { status: res.status, headers: { 'Content-Type': 'application/json' } })
    }
  })
  await page.route(/\/api\/tasks(\?.*)?$/, (route) => (route.request().method() === 'GET' ? json(route, TASKS) : route.fallback()))
  await page.route(/\/api\/day-tasks\?/, (route) => {
    day = new URL(route.request().url()).searchParams.get('from') ?? day
    return json(route, [state()])
  })
  await page.route(/\/api\/day-tasks\/proposal/, (route) =>
    json(route, TASKS.map((t) => ({ task_id: t.id, title: t.title, done: false }))),
  )
  await page.route(/\/api\/day-tasks\/confirm$/, (route) => {
    sent.confirm = route.request().postDataJSON()
    confirmed = true
    return json(route, state())
  })
  await page.route(/\/api\/tasks\/t-1$/, (route) => {
    if (route.request().method() !== 'PATCH') return route.fallback()
    sent.patched = { id: 't-1', body: route.request().postDataJSON() }
    done = true
    return json(route, { ...TASKS[0], status: 'DONE' })
  })
  return sent
}

test('taking the proposed day sends its tasks, and a tick closes one', async ({ authedPage }) => {
  const sent = await mockDayTasks(authedPage)
  await authedPage.goto('/day-tasks')

  const take = authedPage.getByRole('button', { name: 'Take it' })
  await expect(take).toBeVisible({ timeout: 15_000 })
  await take.click()
  await expect.poll(() => sent.confirm, { timeout: 10_000 }).toBeTruthy()
  expect(sent.confirm!.task_ids).toEqual(['t-1', 't-2'])
  expect(sent.confirm!.day).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  await expect(authedPage.getByTestId('day-progress')).toBeVisible()

  await authedPage.getByRole('button', { name: 'Mark done' }).first().click()
  await expect.poll(() => sent.patched, { timeout: 10_000 }).toBeTruthy()
  // A one-off task is closed by status (api/dayTasks.ts markDayTaskDone).
  expect(sent.patched!.body).toMatchObject({ status: 'DONE' })
  await expect(authedPage.getByTestId('day-progress')).toContainText('1')

  await authedPage.unrouteAll({ behavior: 'ignoreErrors' })
})
