import { test, expect } from './fixtures/auth'

/**
 * Settings → feature toggles: seven of eight switched nothing on — no screen is
 * imported behind Dreams, Goals, Projects, Timeline, Opportunities, Needs or Graph
 * (friction audit 26.10 #10). Denis 27.10: keep them, mark «soon». A «soon»
 * switch must not save anything; Tools (a real page) still does.
 *
 * Denis's real account: the «soon» click is the harmless one, and Tools is only
 * checked for being enabled, never clicked.
 */
test('a «soon» toggle cannot be switched and saves nothing; Tools stays a real switch', async ({ authedPage }) => {
  await authedPage.goto('/settings')
  const soon = authedPage.getByTestId('feature-opportunities')
  await expect(soon).toBeVisible({ timeout: 15_000 })
  await expect(soon).toBeDisabled()
  await expect(soon).toContainText(/скоро|soon/i)

  let saved = 0
  authedPage.on('request', (r) => {
    if (r.method() === 'PATCH' && r.url().includes('/api/auth/me')) saved++
  })
  await soon.click({ force: true })
  await authedPage.waitForTimeout(1500) // the page's saver is debounced
  expect(saved, 'clicking a «soon» toggle must not write settings').toBe(0)

  // Positive control: the one real feature is still a live switch.
  await expect(authedPage.getByTestId('feature-tools')).toBeEnabled()
})
