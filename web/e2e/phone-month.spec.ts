import { test, expect } from './fixtures/auth'

/**
 * Gap list row 3 (Denis 26.09: «let people choose, build all of these, okay,
 * A + C + D»): on a phone the calendar has a Week/Month switch, and «Month» is
 * the person's choice of three. Nothing is written: the account's own settings
 * answer is read and given a phone variant here, every write refused.
 */
for (const variant of ['split', 'strip', 'heat'] as const) {
  test(`phone month variant ${variant}`, async ({ authedPage: page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'a phone-only view')

    await page.route('**/api/**', (route) =>
      ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
    )
    await page.route('**/api/auth/me', async (route) => {
      if (route.request().method() !== 'GET') return route.abort()
      const res = await route.fetch()
      const body = await res.json()
      const user = body.data ?? body
      user.settings = { ...(user.settings ?? {}), phone_month_variant: variant }
      await route.fulfill({ response: res, json: body })
    })

    await page.goto('/calendar')
    await page.getByTestId('view-month').click()

    if (variant === 'strip') {
      const strip = page.getByTestId('week-strip')
      await expect(strip).toBeVisible({ timeout: 15_000 })
      const days = strip.getByTestId('week-strip-day')
      await expect(days).toHaveCount(7)
      // A tap shows that day below: the pressed day follows the tap.
      await days.nth(0).click()
      await expect(days.nth(0)).toHaveAttribute('aria-pressed', 'true')
      // ← from Monday: the grid goes to last week, and the strip goes with it
      // rather than keeping a week the shown day is not in (review of 3fe9456).
      const monday = await days.nth(0).getAttribute('data-day')
      await page.getByTitle('Previous period').click()
      await expect(days.nth(0)).not.toHaveAttribute('data-day', monday ?? '')
      await expect(strip.locator('[aria-pressed="true"]')).toHaveCount(1)
    } else {
      await expect(page.getByTestId('month-view')).toHaveAttribute('data-variant', variant, { timeout: 15_000 })
      await expect(page.getByTestId('month-day')).toHaveCount(42)
    }
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
    expect(overflow, 'no horizontal scroll at phone width').toBeLessThanOrEqual(0)
    await page.screenshot({ path: testInfo.outputPath(`phone-month-${variant}.png`) })
    await page.unrouteAll({ behavior: 'ignoreErrors' })
  })
}
