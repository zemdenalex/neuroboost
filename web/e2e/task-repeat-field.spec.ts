import { test, expect } from './fixtures/auth'

/**
 * Gap list row 2 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md): a
 * task made in the web can repeat. Through the real form; nothing is written
 * to the account — the create is caught and answered here, every other write
 * is refused.
 */
test('a new task can be made to repeat every week', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: the same form on both')

  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  let sent: { title?: string; rrule?: string; nag_minutes?: number } | undefined
  await page.route('**/api/tasks', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    sent = route.request().postDataJSON()
    await route.fulfill({
      status: 201, contentType: 'application/json',
      body: JSON.stringify({ data: { id: 'e2e-repeat', title: sent?.title, status: 'TODO', priority: 3, tags: [], contexts: [], rrule: sent?.rrule, created_at: new Date().toISOString(), updated_at: new Date().toISOString() } }),
    })
  })

  await page.goto('/tasks')
  await page.getByRole('button', { name: /^(\+ )?(New Task|Новая задача)$/ }).first().click()
  // The title field has autofocus in the editor.
  await expect(page.getByTestId('task-repeat')).toBeVisible()
  await page.keyboard.type('e2e повтор по неделям')
  await page.getByTestId('task-repeat').selectOption('weekly')
  // Gap list row 13: an unanswered reminder can come back, per task, as in the bot.
  await expect(page.getByTestId('task-nag')).toBeDisabled()
  await page.locator('input[type="datetime-local"]').first().fill('2026-12-01T10:00')
  await page.getByTestId('task-nag').selectOption('15')
  await page.getByRole('button', { name: /^(Save|Сохранить)$/ }).click()

  await expect.poll(() => sent?.rrule, { timeout: 10_000 }).toBe('FREQ=WEEKLY')
  expect(sent?.nag_minutes).toBe(15)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
