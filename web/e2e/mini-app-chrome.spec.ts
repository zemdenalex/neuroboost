import { test, expect } from '@playwright/test'
import { createHmac } from 'node:crypto'

/**
 * Inside Telegram on a phone the app drew its own top bar (NeuroBoost, ?, avatar)
 * under Telegram's header: two headers (friction audit 26.10, #3). Denis 27.10:
 * hide ours in the Mini App, profile and help in «More».
 *
 * Signed the same way as mini-app.spec.ts (the Mini App scheme, the dev bot's token).
 */

function signInitData(botToken: string, fields: Record<string, string>): string {
  const check = Object.keys(fields)
    .sort()
    .map((k) => `${k}=${fields[k]}`)
    .join('\n')
  const secret = createHmac('sha256', 'WebAppData').update(botToken).digest()
  const hash = createHmac('sha256', secret).update(check).digest('hex')
  return new URLSearchParams({ ...fields, hash }).toString()
}

test('in the Mini App on a phone our top bar is gone and «More» offers help and the profile', async ({ page }) => {
  test.skip(test.info().project.name !== 'mobile', 'the top bar is hidden on a phone only')
  const botToken = process.env.E2E_TG_BOT_TOKEN
  const tgId = process.env.E2E_TG_ID
  test.skip(!botToken || !tgId, 'E2E_TG_BOT_TOKEN and E2E_TG_ID are required')

  await page.route('https://telegram.org/**', (r) => r.abort())
  await page.addInitScript(() => {
    localStorage.setItem('neuroboost-onboarding-welcome-seen', 'true')
    localStorage.setItem('neuroboost-onboarding-checklist-dismissed', 'true')
  })
  const initData = signInitData(botToken!, {
    auth_date: String(Math.floor(Date.now() / 1000)),
    user: JSON.stringify({ id: Number(tgId), first_name: 'E2E' }),
  })
  await page.goto(`/calendar#tgWebAppData=${encodeURIComponent(initData)}&tgWebAppVersion=8.0&tgWebAppPlatform=android`)
  await expect(page.getByTestId('week-day-header').first()).toBeVisible({ timeout: 20_000 })

  await expect(page.getByTestId('profile-menu'), 'our top bar (with the avatar) should not be drawn inside Telegram').toHaveCount(0)
  // And no room kept for it: the page starts at the top, not 56px below an empty strip.
  const padTop = await page.locator('main').evaluate((el) => parseFloat(getComputedStyle(el).paddingTop))
  expect(padTop, 'main should not reserve the top bar’s height').toBeLessThan(10)

  await page.getByTestId('tab-more').click()
  await expect(page.getByTestId('more-help'), '«More» should offer this page’s help').toBeVisible()
  await expect(page.getByRole('button', { name: /профиль|profile/i }), '«More» should keep the profile').toBeVisible()
})
