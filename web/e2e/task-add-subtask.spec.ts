import { test, expect } from './fixtures/auth'

/**
 * Gap list row 7 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): a
 * subtask could be made only with Alt+→ in the quick-add row, which a phone
 * does not have. The row's «⋯» menu now has «Подзадача»: the task form opens
 * with the parent set, in the parent's calendar (bot pass 3.7).
 *
 * Nothing is written to the account: the parent is added to the page's own
 * task list, the create is caught here, every other write is refused.
 */
const PARENT = {
  id: 'e2e-subtask-parent', title: 'e2e ремонт', status: 'TODO', priority: 2,
  calendar_id: 'e2e-cal-home', actual_minutes: 0, tags: [], contexts: [], reminder_offsets: [],
  user_id: 'e2e', created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
}

test('a subtask is added from the row menu on a phone', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', 'the «⋯» menu is the phone path')

  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  await page.route('**/api/auth/me', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const user = body.data ?? body
    user.settings = { ...(user.settings ?? {}), task_row_actions: 'menu' }
    await route.fulfill({ response: res, json: body })
  })
  let sent: { title?: string; parent_id?: string; calendar_id?: string } | undefined
  await page.route(/\/api\/tasks(\?.*)?$/, async (route) => {
    const req = route.request()
    if (req.method() === 'POST') {
      sent = req.postDataJSON()
      return route.fulfill({
        status: 201, contentType: 'application/json',
        body: JSON.stringify({ data: { ...PARENT, id: 'e2e-subtask-child', title: sent?.title, parent_id: PARENT.id } }),
      })
    }
    if (req.method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const list = Array.isArray(body.data) ? body.data : Array.isArray(body) ? body : []
    list.unshift(PARENT)
    await route.fulfill({ response: res, json: Array.isArray(body) ? list : { ...body, data: list } })
  })

  await page.goto('/tasks')
  await page.getByRole('button', { name: new RegExp(`: ${PARENT.title}$`) }).first().click()
  await page.getByTestId('task-add-subtask').click()
  await expect(page.getByTestId('task-repeat')).toBeVisible()
  await page.keyboard.type('e2e купить краску')
  await page.getByRole('button', { name: /^(Save|Сохранить)$/ }).click()

  await expect.poll(() => sent?.parent_id, { timeout: 10_000 }).toBe(PARENT.id)
  expect(sent?.calendar_id, 'a subtask of a shared task is filed in the same calendar').toBe(PARENT.calendar_id)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
