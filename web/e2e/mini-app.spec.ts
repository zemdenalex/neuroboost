import { test, expect } from '@playwright/test'
import { createHmac } from 'node:crypto'

/**
 * Telegram Mini App: opened from the bot with a signed identity in the hash,
 * the person lands in the calendar with no login screen.
 *
 * The launch data is signed here with the dev bot's token (the one staging's
 * API holds, checked 25.09), by the Mini App scheme: secret =
 * HMAC-SHA256("WebAppData", token), NOT the Login Widget's SHA256(token).
 * Telegram's own script is blocked: the sign-in must not depend on it.
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

test('a Mini App launch signs in without the login screen', async ({ page }) => {
  const botToken = process.env.E2E_TG_BOT_TOKEN
  const tgId = process.env.E2E_TG_ID
  test.skip(!botToken || !tgId, 'E2E_TG_BOT_TOKEN and E2E_TG_ID are required')

  await page.route('https://telegram.org/**', (r) => r.abort())
  // Skip the welcome overlay, as the other specs do; no token is seeded.
  await page.addInitScript(() => {
    localStorage.setItem('neuroboost-onboarding-welcome-seen', 'true')
    localStorage.setItem('neuroboost-onboarding-checklist-dismissed', 'true')
  })

  const initData = signInitData(botToken!, {
    auth_date: String(Math.floor(Date.now() / 1000)),
    user: JSON.stringify({ id: Number(tgId), first_name: 'E2E' }),
  })
  const hash = `#tgWebAppData=${encodeURIComponent(initData)}&tgWebAppVersion=8.0&tgWebAppPlatform=android`

  await page.goto(`/calendar${hash}`)
  await expect(page).toHaveURL(/\/calendar/, { timeout: 20_000 })
  // Either marker proves the calendar; on a phone both are on screen at once
  // (the tap hint shows on the first three opens), so take the first match.
  await expect(page.getByTestId('week-day-header').or(page.getByText(/tap: select|нажми/i)).first()).toBeVisible({
    timeout: 20_000,
  })
  expect(await page.evaluate(() => localStorage.getItem('nb_token')), 'a session was stored').toBeTruthy()
})

test('a forged launch falls back to the login screen', async ({ page }) => {
  const tgId = process.env.E2E_TG_ID
  test.skip(!tgId, 'E2E_TG_ID is required')
  await page.route('https://telegram.org/**', (r) => r.abort())
  const forged = signInitData('1:not-the-bot', {
    auth_date: String(Math.floor(Date.now() / 1000)),
    user: JSON.stringify({ id: Number(tgId), first_name: 'E2E' }),
  })
  await page.goto(`/calendar#tgWebAppData=${encodeURIComponent(forged)}`)
  await expect(page).toHaveURL(/\/login|\/$/, { timeout: 20_000 })
  expect(await page.evaluate(() => localStorage.getItem('nb_token'))).toBeFalsy()
})
