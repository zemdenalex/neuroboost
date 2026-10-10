import { test, expect } from '@playwright/test'

/**
 * Two pages for people who are not signed in yet (queue: pages without any
 * test). No account: these routes exist precisely for a visitor without one.
 * The one server call (redeeming a bot link) is answered by the test, so no
 * real token is spent.
 */

test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
})

test('an invite opened while signed out goes to the login and keeps the invite as next', async ({ page }) => {
  await page.goto('/i/e2e-invite-token')
  await expect(page).toHaveURL(/\/login\?next=%2Fi%2Fe2e-invite-token$/, { timeout: 15_000 })
})

test('a spent bot link says so and leads back to the login', async ({ page }) => {
  await page.route('**/api/auth/login-link/redeem', (route) =>
    route.fulfill({ status: 410, json: { error: { code: 'LINK_USED', message: 'used' } } }),
  )
  await page.goto('/login/link?t=e2e-spent')
  await expect(page.getByRole('heading', { name: 'Ссылка не сработала' })).toBeVisible({ timeout: 15_000 })
  await page.getByRole('button', { name: 'Ко входу' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('a good bot link signs in, and the token does not stay in the address', async ({ page }) => {
  let sent: unknown
  await page.route('**/api/auth/login-link/redeem', (route) => {
    sent = route.request().postDataJSON()
    return route.fulfill({
      status: 200,
      json: { data: { token: 'e2e.fake.jwt', expires_at: Math.floor(Date.now() / 1000) + 3600, user: { id: 'u' } } },
    })
  })
  // Stop at the page it goes to: the fake token is not meant to load anything.
  await page.route('**/profile', (route) => route.fulfill({ status: 200, contentType: 'text/html', body: '<p>landed</p>' }))
  await page.goto('/login/link?t=e2e-good')
  await expect(page).toHaveURL(/\/profile$/, { timeout: 15_000 })
  expect(sent).toEqual({ token: 'e2e-good' })
  expect(page.url()).not.toContain('e2e-good')
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
