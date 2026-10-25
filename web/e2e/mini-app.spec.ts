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

// MA3b (Denis 26.09: «everything from Telegram»): the person's Telegram colours
// paint the app, in a dark and a light Telegram theme alike.
const THEMES = {
  night: {
    params: { bg_color: '#212d3b', secondary_bg_color: '#1d2733', text_color: '#ffffff', hint_color: '#7d8b99',
      link_color: '#5eabe1', button_color: '#50a8eb', button_text_color: '#ffffff' },
    expected: { page: '33 45 59', button: '80 168 235' },
  },
  light: {
    params: { bg_color: '#ffffff', secondary_bg_color: '#efeff4', text_color: '#000000', hint_color: '#8e8e93',
      link_color: '#007aff', button_color: '#007aff', button_text_color: '#ffffff' },
    expected: { page: '255 255 255', button: '0 122 255' },
  },
}

for (const [name, theme] of Object.entries(THEMES)) {
  test(`a Mini App launch takes the Telegram theme colours (${name})`, async ({ page }, testInfo) => {
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
    const hash = `#tgWebAppData=${encodeURIComponent(initData)}&tgWebAppVersion=8.0&tgWebAppPlatform=android` +
      `&tgWebAppThemeParams=${encodeURIComponent(JSON.stringify(theme.params))}`

    await page.goto(`/tasks${hash}`)
    await expect(page).toHaveURL(/\/tasks/, { timeout: 20_000 })
    const vars = await page.evaluate(() => {
      const s = getComputedStyle(document.documentElement)
      return { page: s.getPropertyValue('--nb-zinc-950').trim(), button: s.getPropertyValue('--nb-blue-600').trim() }
    })
    expect(vars).toEqual(theme.expected)
    await page.waitForLoadState('networkidle')
    await expect(page.locator('.animate-spin')).toHaveCount(0, { timeout: 20_000 })
    await page.screenshot({ path: testInfo.outputPath(`telegram-${name}.png`) })
  })
}

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

// Full screen (Bot API 8.0): the choice lives in Settings, is kept on the device,
// and is asked of Telegram at the next launch. Telegram's script is replaced by a
// recorder, so this proves the page's side of the door, not Telegram's.
test('full screen is offered in Telegram, asked for, and asked again at the next launch', async ({ page }) => {
  const botToken = process.env.E2E_TG_BOT_TOKEN
  const tgId = process.env.E2E_TG_ID
  test.skip(!botToken || !tgId, 'E2E_TG_BOT_TOKEN and E2E_TG_ID are required')

  await page.route('https://telegram.org/**', (r) => r.abort())
  await page.addInitScript(() => {
    localStorage.setItem('neuroboost-onboarding-welcome-seen', 'true')
    localStorage.setItem('neuroboost-onboarding-checklist-dismissed', 'true')
    const calls: string[] = []
    ;(window as unknown as { __tgCalls: string[] }).__tgCalls = calls
    const handlers: Record<string, Array<() => void>> = {}
    const wa = {
      version: '8.0', initData: '', isFullscreen: false,
      ready() {}, expand() {}, disableVerticalSwipes() {},
      isVersionAtLeast: () => true,
      BackButton: { show() {}, hide() {}, onClick() {}, offClick() {} },
      onEvent(e: string, cb: () => void) { (handlers[e] ||= []).push(cb) },
      requestFullscreen() { calls.push('request'); wa.isFullscreen = true; handlers.fullscreenChanged?.forEach((f) => f()) },
      exitFullscreen() { calls.push('exit'); wa.isFullscreen = false; handlers.fullscreenChanged?.forEach((f) => f()) },
    }
    ;(window as unknown as { Telegram: unknown }).Telegram = { WebApp: wa }
  })
  const initData = signInitData(botToken!, {
    auth_date: String(Math.floor(Date.now() / 1000)),
    user: JSON.stringify({ id: Number(tgId), first_name: 'E2E' }),
  })
  const hash = `#tgWebAppData=${encodeURIComponent(initData)}&tgWebAppVersion=8.0&tgWebAppPlatform=android`
  const calls = () => page.evaluate(() => (window as unknown as { __tgCalls: string[] }).__tgCalls)

  // Settings is read-only for this test: no setting is written (the choice is localStorage).
  await page.goto(`/settings${hash}`)
  const box = page.getByTestId('telegram-fullscreen')
  await expect(box).toBeVisible({ timeout: 20_000 })
  expect(await calls(), 'nothing is requested until the person chooses').toEqual([])

  await box.check()
  await expect.poll(calls).toEqual(['request'])
  await expect(page.locator('html')).toHaveClass(/tg-fullscreen/)

  // The header is position:fixed, so padding on body would not move it: it has to
  // clear Telegram's buttons itself (the inset is what the client reports).
  await page.evaluate(() => document.documentElement.style.setProperty('--tg-safe-area-inset-top', '44px'))
  await expect
    .poll(async () => (await page.locator('header').first().boundingBox())?.y ?? -1, 'the fixed header sits below the inset')
    .toBeGreaterThanOrEqual(0)
  const headerText = page.locator('header').first().getByText('NeuroBoost')
  await expect.poll(async () => (await headerText.boundingBox())?.y ?? -1).toBeGreaterThanOrEqual(44)

  await box.uncheck()
  await expect.poll(calls).toEqual(['request', 'exit'])
  await expect(page.locator('html')).not.toHaveClass(/tg-fullscreen/)

  await box.check()
  await page.reload()
  await expect(page.getByTestId('telegram-fullscreen')).toBeChecked({ timeout: 20_000 })
  await expect.poll(calls, 'the saved choice is asked of Telegram at launch').toContain('request')
})
