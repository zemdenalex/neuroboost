import { test, expect } from './fixtures/auth'

/**
 * Gap list row 19: the web shows the bot's «Что нового» (bot/release through
 * GET /api/release-notes), reached from «Ещё» on a phone and the profile menu
 * on a computer. The notes answer is served here so the spec does not depend
 * on which release is out; every write is refused.
 */
test('«Что нового» opens from the menu and shows the bot notes', async ({ authedPage: page }, testInfo) => {
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  await page.route('**/api/release-notes', (route) =>
    route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ data: [{ version: 'v9.9.9', released: 'e2e', ru: '• проверка e2e', en: '• e2e check' }] }),
    }),
  )

  await page.goto('/calendar')
  if (testInfo.project.name === 'mobile') {
    await page.getByTestId('tab-more').click()
  } else {
    await page.getByTestId('profile-menu').click()
  }
  await page.getByRole('link', { name: /^(Что нового|What's new)$/ }).or(page.getByRole('button', { name: /^(Что нового|What's new)$/ })).first().click()

  await expect(page).toHaveURL(/\/whats-new/)
  await expect(page.getByTestId('whats-new-note')).toContainText('v9.9.9')
  await expect(page.getByTestId('whats-new-note')).toContainText(/проверка e2e|e2e check/)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
