import { test, expect } from './fixtures/auth'

// docs/tasks-web-cleanup.md 4.1 and 4.2 (audit A2, A4). The feedback API is
// answered here, so nothing is written to the staging backlog: the Backlog's
// own request (it always sends sort_by) gets one item, the Overview's plain
// request gets the whole three.
const item = (id: string, type: string, status: string) => ({
  id, type, status, priority: 'medium', title: `e2e ${id}`, description: '', source: 'web',
  admin_notes: '', tags: [], created_at: '2026-09-25T10:00:00Z', updated_at: '2026-09-25T10:00:00Z',
})
const ALL = [item('a', 'bug', 'open'), item('b', 'idea', 'open'), item('c', 'bug', 'resolved')]

test('Overview counts the whole backlog; a row whose edit failed says so', async ({ authedPage }) => {
  await authedPage.route('**/api/feedback**', async (route) => {
    const req = route.request()
    if (req.method() === 'PATCH') return route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":{"code":"X","message":"x"}}' })
    const filtered = new URL(req.url()).searchParams.has('sort_by')
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: filtered ? [ALL[0]] : ALL }) })
  })
  // The page is shown only to an admin. CI's e2e account is not one (the local
  // account happened to be, which is how this spec first passed), so the
  // account's own answer is taken and marked admin: the gate is the page's,
  // and the feedback API it then reads is answered above.
  await authedPage.route('**/api/auth/me', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const user = body.data ?? body
    user.is_admin = true
    await route.fulfill({ response: res, json: body })
  })
  await authedPage.goto('/admin')
  const overview = authedPage.getByRole('button', { name: 'Overview' })
  await overview.waitFor({ timeout: 15_000 })
  await overview.click()
  await expect(authedPage.getByTestId('overview-total')).toHaveText('3')

  await authedPage.getByRole('button', { name: 'Backlog' }).click()
  const row = authedPage.getByText('e2e a').first()
  await expect(row).toBeVisible()
  // The status select lives in the expanded row.
  await row.click()
  await authedPage.locator('select').filter({ has: authedPage.locator('option[value="resolved"]') }).last().selectOption('resolved')
  await expect(authedPage.getByTestId('feedback-row-unsaved')).toBeVisible()
})
