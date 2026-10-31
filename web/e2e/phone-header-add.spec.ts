import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext, type Page } from '@playwright/test'
import { withSettings } from './fixtures/settings'

/**
 * Phone header, «Month» and «+» (Denis 31.10, spec
 * docs/team/architecture/V003-20261031-arc-shapka-dobavit.md): «header A + C
 * choice in settings, for creation I think it should have simple mode and
 * elaborate, and simple uses A, elaborate opens like the one in ticktick,
 * month b and c depending on what is the choice of header».
 *
 * Settings are laid over in the browser only (fixtures/settings); items made
 * by «+» carry a test prefix and are deleted after each test.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.space'
/** Where the time grid started on 375×667 before (two rows + hint), staging 27.10. */
const OLD_GRID_TOP = 298

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

async function accountZone(token: string): Promise<string> {
  const ctx = await apiContext(token)
  const me = await (await ctx.get('/api/auth/me')).json()
  await ctx.dispose()
  return (me.data ?? me).timezone || 'Europe/Moscow'
}

const gridTop = (page: Page) =>
  page.locator('[data-hint="calendar.grid"]').evaluate((el) => Math.round(el.getBoundingClientRect().top))

test.describe('phone header and «+»', () => {
  const made: { kind: 'events' | 'tasks'; id: string }[] = []

  test.beforeEach(async ({}, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', 'a phone-only header')
  })

  test.afterEach(async ({ session }) => {
    if (made.length === 0) return
    const ctx = await apiContext(session.token)
    for (const m of made.splice(0)) await ctx.delete(`/api/${m.kind}/${m.id}`)
    await ctx.dispose()
  })

  /** Remember what a POST made, so afterEach can delete it. */
  function track(page: Page, kind: 'events' | 'tasks') {
    return page.waitForResponse((r) => r.request().method() === 'POST' && new URL(r.url()).pathname === `/api/${kind}`).then(async (r) => {
      const body = await r.json()
      const id = (body.data ?? body).id as string
      made.push({ kind, id })
      return { status: r.status(), body: body.data ?? body }
    })
  }

  test('header A: one row, the grid starts well above where it did', async ({ authedPage: page }, testInfo) => {
    await withSettings(page, { phone_header_variant: 'row' })
    await page.goto('/calendar')
    const header = page.getByTestId('calendar-header')
    await expect(header).toBeVisible({ timeout: 15_000 })
    // «+ Задача» left the phone's row: «+» at the bottom makes tasks too.
    await expect(header.getByText(/\+ (Задача|Task)/)).toHaveCount(0)
    // The arrows, the date, Today and the switch share one line.
    const tops = await header.locator('button, h2').evaluateAll((els) =>
      [...new Set(els.filter((e) => (e as HTMLElement).offsetParent && !e.closest('[data-testid="calendar-hint"]')).map((e) => Math.round(e.getBoundingClientRect().top / 8)))],
    )
    expect(tops.length, `header controls on ${tops.length} rows`).toBe(1)
    const top = await gridTop(page)
    testInfo.annotations.push({ type: 'grid top', description: `${top}px (was ${OLD_GRID_TOP})` })
    expect(top).toBeLessThan(OLD_GRID_TOP - 40)
    // «Month» under header A opens a real month, never the strip (Month C).
    await withSettings(page, { phone_header_variant: 'row', phone_month_variant: 'strip' })
    await page.reload()
    await page.getByTestId('view-month').click()
    await expect(page.getByTestId('month-view')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('week-strip')).toHaveCount(0)
    await page.getByTestId('view-week').click()
    await page.screenshot({ path: testInfo.outputPath('header-row.png') })
  })

  test('header C: the strip navigates, the month opens from its title (Month B)', async ({ authedPage: page }, testInfo) => {
    await withSettings(page, { phone_header_variant: 'strip' })
    await page.goto('/calendar')
    const strip = page.getByTestId('week-strip')
    await expect(strip).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('view-month')).toHaveCount(0)
    await page.screenshot({ path: testInfo.outputPath('header-strip.png') })

    await page.getByTestId('calendar-month-title').click()
    await expect(page.getByTestId('month-view')).toBeVisible({ timeout: 15_000 })
    // A day chosen in the month, then «Open week», brings that day back with
    // the strip over it (the phone month's own way to a day).
    const day = page.getByTestId('month-day').nth(17)
    const picked = await day.getAttribute('data-day')
    await day.click()
    await page.getByRole('button', { name: /Open week|Открыть неделю/ }).click()
    await expect(strip).toBeVisible({ timeout: 15_000 })
    if (picked) await expect(strip.locator(`[data-day="${picked}"]`)).toHaveAttribute('aria-pressed', 'true')
  })

  test('«+» simple: a timed line is an event, a plain line a task', async ({ authedPage: page, session }) => {
    await withSettings(page, { add_mode: 'simple' })
    const tag = `E2E-ADD-${Date.now()}`
    await page.goto('/calendar')
    await page.getByTestId('tab-add').click()
    const line = page.getByTestId('add-line')
    await line.fill(`завтра 10:00 ${tag} врач`)
    await expect(page.getByTestId('add-preview')).toHaveAttribute('data-kind', 'event', { timeout: 10_000 })
    const event = track(page, 'events')
    await line.press('Enter')
    const ev = await event
    expect(ev.status).toBeLessThan(300)
    expect(ev.body.title).toContain(tag)
    // 10:00 in the account's zone: the hour is checked in that zone.
    const zone = await accountZone(session.token)
    const hour = new Intl.DateTimeFormat('en-GB', { timeZone: zone, hour: '2-digit', hourCycle: 'h23' }).format(new Date(ev.body.starts_at))
    expect(hour).toBe('10')
    const tomorrow = new Intl.DateTimeFormat('en-CA', { timeZone: zone }).format(new Date(Date.now() + 86_400_000))
    expect(new Intl.DateTimeFormat('en-CA', { timeZone: zone }).format(new Date(ev.body.starts_at))).toBe(tomorrow)
    await expect(page.getByTestId('add-sheet')).toHaveCount(0)

    await page.getByTestId('tab-add').click()
    await line.fill(`${tag} купить хлеб`)
    await expect(page.getByTestId('add-preview')).toHaveAttribute('data-kind', 'task', { timeout: 10_000 })
    const task = track(page, 'tasks')
    await line.press('Enter')
    const tk = await task
    expect(tk.status).toBeLessThan(300)
    expect(tk.body.title).toContain(tag)
  })

  test('«+» detailed: the day chip starts on the open day, and a line naming no day lands there', async ({ authedPage: page, session }) => {
    await withSettings(page, { add_mode: 'detailed', phone_header_variant: 'strip' })
    const tag = `E2E-ADD-${Date.now()}`
    await page.goto('/calendar')
    const strip = page.getByTestId('week-strip')
    await expect(strip).toBeVisible({ timeout: 15_000 })
    // Look at another day of the week; «+» must start there, not today.
    const other = strip.locator('[aria-pressed="false"]').first()
    const otherDay = await other.getAttribute('data-day')
    expect(otherDay).toBeTruthy()
    await other.click()
    await page.getByTestId('tab-add').click()
    await expect(page.getByTestId('add-detailed-form')).toBeVisible()
    await expect(page.getByTestId('add-chip-date')).toHaveAttribute('data-day', otherDay!)

    // No day in the line: POST /api/parse gets the open day and the event lands on it.
    await page.getByTestId('add-title').fill(`${tag} врач 10:00`)
    await expect(page.getByTestId('add-chip-time')).toContainText('10:00', { timeout: 10_000 })
    await expect(page.getByTestId('add-kind')).toHaveAttribute('data-kind', 'event')
    const event = track(page, 'events')
    await page.getByTestId('add-create').click()
    const ev = await event
    expect(ev.status).toBeLessThan(300)
    // The words that set the chips are not left in the title.
    expect(ev.body.title).toContain(tag)
    expect(ev.body.title).not.toContain('10:00')
    const zone = await accountZone(session.token)
    expect(new Intl.DateTimeFormat('en-CA', { timeZone: zone }).format(new Date(ev.body.starts_at))).toBe(otherDay)
  })
})
