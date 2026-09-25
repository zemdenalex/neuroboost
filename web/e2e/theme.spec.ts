import { test, expect } from './fixtures/auth'

/**
 * Light inside a light Telegram, dark everywhere else (Denis 25.09: «follow
 * Telegram's theme»; docs/tasks-light-theme.md).
 *
 * Telegram passes its theme in the launch hash as tgWebAppThemeParams. Without
 * tgWebAppData alongside it the Mini App sign-in does not start, so the theme
 * can be driven on its own, on the ordinary e2e session.
 */

const lightHash = '#tgWebAppThemeParams=' + encodeURIComponent(JSON.stringify({ bg_color: '#ffffff', text_color: '#000000' }))

async function bodyLuminance(page: import('@playwright/test').Page): Promise<number> {
  return page.evaluate(() => {
    const [r, g, b] = getComputedStyle(document.body).backgroundColor.match(/\d+/g)!.map(Number)
    return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255
  })
}

test('a light Telegram gets the light app', async ({ authedPage }) => {
  await authedPage.goto(`/settings${lightHash}`)
  await expect(authedPage.locator('html')).toHaveAttribute('data-theme', 'light')
  await expect.poll(() => bodyLuminance(authedPage), { message: 'page background is light' }).toBeGreaterThan(0.9)
  // Text is dark on it: the page title takes the inverted "white".
  const heading = authedPage.getByRole('heading', { level: 1 }).first()
  await expect(heading).toBeVisible({ timeout: 15_000 })
  const ink = await heading.evaluate((el) => getComputedStyle(el).color.match(/\d+/g)!.map(Number))
  expect(Math.max(...ink), 'heading text is dark').toBeLessThan(80)
})

test('without Telegram the app stays dark', async ({ authedPage }) => {
  await authedPage.goto('/settings')
  await expect(authedPage.locator('html')).toHaveAttribute('data-theme', 'dark')
  await expect.poll(() => bodyLuminance(authedPage)).toBeLessThan(0.1)
})
