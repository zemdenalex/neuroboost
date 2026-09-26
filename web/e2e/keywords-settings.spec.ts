import { test, expect } from './fixtures/auth'

/**
 * Gap list row 17: the bot's own words (settings.bot.keywords) are listed,
 * added and deleted in Settings. Nothing is written to the account: the
 * account's settings answer is replaced by a fake blob here, the settings
 * write is caught and becomes the next answer, every other write is refused.
 *
 * 🔴 The point of the spec is the bot word added AFTER the page rendered: a
 * write built from the tab's copy would erase it, and only a write built from
 * what the server holds at write time keeps it.
 */
type Settings = { work_start?: string; bot?: { lang?: string; priority_style?: string; keywords?: Record<string, unknown> } }

const LONG = 'я'.repeat(28)

test('own words are listed, added and deleted without losing anything else', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one project is enough: the 375px check sets its own viewport')

  let state: Settings = {
    work_start: '09:00',
    bot: {
      lang: 'ru',
      priority_style: 'dot',
      keywords: {
        созвон: { field: 'calendar', value: 'Работа' },
        спорт: 'Спорт', // the first shape the bot shipped: a bare string is a tag
        странное: { field: 'mood', value: 'x' }, // a field this web does not know
        [LONG]: { field: 'task', value: '' },
      },
    },
  }
  const writes: Settings[] = []

  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  await page.route('**/api/auth/me', async (route) => {
    const req = route.request()
    if (req.method() === 'PATCH') {
      const body = req.postDataJSON() as { settings?: Settings }
      writes.push(body.settings ?? {})
      state = body.settings ?? state
    } else if (req.method() !== 'GET') return route.abort()
    const res = await route.fetch({ method: 'GET', postData: undefined })
    const body = await res.json()
    const user = body.data ?? body
    user.settings = state
    await route.fulfill({ response: res, json: body })
  })

  await page.goto('/settings')
  const section = page.getByTestId('settings-keywords')
  await expect(section.getByTestId('keyword-row-созвон')).toContainText('Работа', { timeout: 15_000 })
  await expect(section.getByTestId('keyword-row-спорт')).toContainText('спорт')
  await expect(section.getByTestId('keyword-row-странное')).toContainText('mood')

  // The bot adds a word while the tab is open; the page never shows it.
  state = { ...state, bot: { ...state.bot, keywords: { ...state.bot?.keywords, ботслово: { field: 'tag', value: 'x' } } } }

  // Two words are refused inline and nothing is written.
  await section.getByTestId('keyword-word').fill('два слова')
  await section.getByTestId('keyword-add').click()
  await expect(section.getByTestId('keyword-error')).toBeVisible()
  expect(writes).toHaveLength(0)

  await section.getByTestId('keyword-word').fill('Зал')
  await section.getByTestId('keyword-field').selectOption('repeat')
  await section.getByTestId('keyword-repeat').selectOption('FREQ=WEEKLY')
  await section.getByTestId('keyword-add').click()
  await expect.poll(() => writes.length, { timeout: 10_000 }).toBe(1)

  const added = writes[0]
  expect(added.work_start, 'other settings are kept').toBe('09:00')
  expect(added.bot?.lang, 'other bot keys are kept').toBe('ru')
  expect(added.bot?.priority_style).toBe('dot')
  expect(added.bot?.keywords).toEqual({
    созвон: { field: 'calendar', value: 'Работа' },
    спорт: 'Спорт',
    странное: { field: 'mood', value: 'x' },
    [LONG]: { field: 'task', value: '' },
    ботслово: { field: 'tag', value: 'x' },
    зал: { field: 'repeat', value: 'FREQ=WEEKLY' },
  })
  await expect(section.getByTestId('keyword-row-зал')).toBeVisible()
  await expect(section.getByTestId('keyword-row-ботслово')).toBeVisible()

  await section.getByTestId('keyword-delete-созвон').click()
  await expect.poll(() => writes.length, { timeout: 10_000 }).toBe(2)
  const rest = Object.fromEntries(Object.entries(added.bot?.keywords ?? {}).filter(([w]) => w !== 'созвон'))
  expect(writes[1].bot?.keywords).toEqual(rest)
  expect(writes[1].bot?.lang).toBe('ru')
  expect(writes[1].work_start).toBe('09:00')
  await expect(section.getByTestId('keyword-row-созвон')).toHaveCount(0)

  // 375px: the 28-letter word and the form fit without a horizontal scroll.
  await page.setViewportSize({ width: 375, height: 800 })
  await expect(section.getByTestId(`keyword-row-${LONG}`)).toBeVisible()
  await section.getByTestId('keyword-field').selectOption('colour')
  await expect(section.getByTestId('keyword-colour-slate')).toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(overflow, 'no horizontal scroll at 375px').toBeLessThanOrEqual(0)

  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
