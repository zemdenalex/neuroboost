import { test, expect } from './fixtures/auth'

// Denis 25.09 (docs/tasks-web-cleanup.md 4.10): «Бюджет времени» takes the day
// from the account's work hours and has no decorative week. Read only: the
// spec changes nothing, so nothing is saved to the (real) staging account.
test('the time budget reads the work hours and draws no week', async ({ authedPage }) => {
  await authedPage.goto('/tools/time-blocking')
  const hours = authedPage.getByTestId('time-budget-hours')
  await expect(hours).toBeVisible({ timeout: 15_000 })
  await expect(hours.locator('input')).toHaveCount(0)
  await expect(hours.getByRole('link')).toHaveAttribute('href', '/settings')
  await expect(authedPage.getByText(/weekly view|недельный вид/i)).toHaveCount(0)
})
