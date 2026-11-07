import { test, expect } from './fixtures/auth'

/**
 * Opening the app while the server blinks must not log the person out (07.11,
 * found by the A5 offline review: AuthContext cleared the token on ANY getMe
 * failure, so a 502 during a redeploy — staging did exactly that mid-run on
 * 07.11 — or a dropped connection sent people to the login screen).
 *
 * The first two answers to GET /api/auth/me are 502; the third is the real one.
 * The app must ask again and land in the calendar, with the token still stored.
 */
test('a 502 on start is asked again; the person stays logged in', async ({ authedPage }) => {
  let blinks = 0
  await authedPage.route('**/api/auth/me', async (route) => {
    if (route.request().method() === 'GET' && blinks < 2) {
      blinks++
      return route.fulfill({ status: 502, contentType: 'text/html', body: '<html>502 Bad Gateway</html>' })
    }
    return route.fallback()
  })
  await authedPage.goto('/calendar')
  await expect(authedPage.getByTestId('calendar-period-title')).toBeVisible({ timeout: 20_000 })
  expect(new URL(authedPage.url()).pathname, 'a blink must not end on the login screen').not.toBe('/login')
  expect(blinks, 'both 502s were served').toBe(2)
  expect(await authedPage.evaluate(() => localStorage.getItem('nb_token')), 'the token is kept').not.toBeNull()
})
