import { test, expect } from './fixtures/auth'

/**
 * Profile shows real numbers only (Denis 25.09: «real numbers, remove mock
 * data»). Until then every account saw XP 1250, level 5 and a 7-day streak.
 */
test('the profile shows counts from real data and none of the old invented ones', async ({ authedPage }, info) => {
  test.skip(info.project.name !== 'desktop', 'one viewport is enough')
  await authedPage.goto('/profile')
  const grid = authedPage.getByTestId('profile-stats')
  await expect(grid).toBeVisible({ timeout: 15_000 })
  // Loaded: the placeholders are replaced by numbers.
  await expect(grid).not.toContainText('—', { timeout: 15_000 })
  // XP is real now (Denis 25.09): shown with its rule, never the old invented 1250.
  await expect(authedPage.getByTestId('profile-xp')).toBeVisible()
  await expect(authedPage.getByTestId('profile-xp')).toContainText('+25')
  await expect(authedPage.getByText(/(^|[^0-9])1250([^0-9]|$)/)).toHaveCount(0)
  await expect(authedPage.getByText(/78%|68%/)).toHaveCount(0)
})
