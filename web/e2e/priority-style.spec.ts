import { test, expect } from './fixtures/auth'

/**
 * Gap list row 16 (Denis 26.09): the web follows the bot's priority symbol
 * (settings.bot.priority_style) and can set it. Nothing is written to the
 * account: the account's own settings answer is taken and changed here, the
 * settings write is caught, every other write refused.
 */
test('the priority symbol is set in settings and drawn on the task list', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: same setting, same marks')

  let style = 'circles'
  let written: { settings?: { bot?: Record<string, unknown> } } | undefined
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  await page.route('**/api/auth/me', async (route) => {
    const req = route.request()
    const res = await route.fetch({ method: 'GET', postData: undefined })
    const body = await res.json()
    const user = body.data ?? body
    const bot = { ...(user.settings?.bot ?? {}), keywords: { e2e: {} }, priority_style: style }
    user.settings = { ...(user.settings ?? {}), bot }
    if (req.method() === 'PATCH') {
      written = req.postDataJSON()
      style = String(written?.settings?.bot?.priority_style ?? style)
      user.settings = written?.settings ?? user.settings
    } else if (req.method() !== 'GET') return route.abort()
    await route.fulfill({ response: res, json: body })
  })

  await page.goto('/settings')
  await page.getByTestId('priority-style-dot').click()
  await expect.poll(() => written?.settings?.bot?.priority_style, { timeout: 10_000 }).toBe('dot')
  expect(written?.settings?.bot?.keywords, 'the rest of the bot section is kept').toEqual({ e2e: {} })

  await page.goto('/tasks')
  await expect(page.getByTestId('priority-mark-text').first()).toHaveText(/^[●○·]\d$/, { timeout: 15_000 })
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
