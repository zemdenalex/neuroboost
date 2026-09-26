import type { Page } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * Gap list row 4 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md):
 * «Запланировать» asks when and how long, as the bot does, says when it went,
 * and the row shows the time.
 *
 * Nothing touches the account: the task is a fixture appended to the real
 * GET, the schedule POST is answered here, every other write is refused.
 */
const ZONE = 'Europe/Moscow'
const TASK = {
  id: 'e2e-plan-choice',
  title: 'e2e запланировать с выбором',
  status: 'TODO',
  priority: 1,
  estimated_minutes: 45,
  actual_minutes: 0,
  tags: [],
  contexts: [],
  reminder_offsets: [],
  user_id: 'e2e',
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
}
const SHOTS = 'C:/Users/zd/AppData/Local/Temp/claude/E--Projects-007---Ventures-V003---NeuroBoost/04e1a014-f855-4c44-926c-c4003cfaca56/scratchpad/row4'

async function stubAccount(page: Page) {
  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  // The «⋯» menu on a phone, and a known zone, whatever the account says.
  await page.route('**/api/auth/me', async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const user = body.data ?? body
    user.timezone = ZONE
    user.settings = { ...(user.settings ?? {}), task_row_actions: 'menu' }
    await route.fulfill({ response: res, json: body })
  })
  await page.route(/\/api\/tasks(\?.*)?$/, async (route) => {
    if (route.request().method() !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const list = Array.isArray(body.data) ? body.data : Array.isArray(body) ? body : []
    list.unshift(TASK)
    await route.fulfill({ response: res, json: Array.isArray(body) ? list : { ...body, data: list } })
  })
}

/** Tomorrow's date in the zone, YYYY-MM-DD, and the wall time of an instant there. */
function zoneParts(at: Date) {
  const f = new Intl.DateTimeFormat('en-CA', { timeZone: ZONE, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  const p = Object.fromEntries(f.formatToParts(at).map((x) => [x.type, x.value]))
  return { date: `${p.year}-${p.month}-${p.day}`, time: `${p.hour}:${p.minute}` }
}

test('a task is scheduled for tomorrow morning with its own estimate, and the row says when', async ({ authedPage: page }, testInfo) => {
  await stubAccount(page)
  let sent: { starts_at?: string; ends_at?: string; all_day?: boolean } | undefined
  await page.route(`**/api/tasks/${TASK.id}/schedule`, async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    sent = route.request().postDataJSON()
    await route.fulfill({
      status: 201, contentType: 'application/json',
      body: JSON.stringify({ data: { id: 'e2e-ev', task_id: TASK.id, title: TASK.title, starts_at: sent?.starts_at, ends_at: sent?.ends_at, all_day: false } }),
    })
  })

  await page.goto('/tasks')
  const row = page.locator('[id^="task-"]').filter({ hasText: TASK.title }).first()
  await expect(row).toBeVisible({ timeout: 15_000 })
  await expect(row.getByTestId('task-scheduled-at')).toHaveCount(0)

  const mobile = testInfo.project.name === 'mobile'
  if (mobile) {
    await row.getByTestId('task-row-more').click()
    await page.getByTestId('task-row-menu').getByRole('menuitem').first().click()
  } else {
    await row.hover()
    await row.locator('[data-hint="tasks.schedule"]').click()
  }

  const chooser = page.getByTestId('schedule-chooser')
  await expect(chooser).toBeVisible()
  await expect(chooser.getByTestId('schedule-when-now')).toBeVisible()
  await expect(chooser.getByRole('button', { name: /^(Сейчас|Now)$/ })).toBeVisible()
  if (mobile) await page.screenshot({ path: `${SHOTS}/chooser-when-375.png` })

  await chooser.getByTestId('schedule-when-tmr').click()
  await expect(chooser.getByTestId('schedule-chooser-question')).toHaveText(/^(завтра|tomorrow) 09:00 · /)
  // The bot's four plus the task's own 45, which is the suggested one.
  for (const m of [15, 30, 45, 60, 120]) await expect(chooser.getByTestId(`schedule-minutes-${m}`)).toBeVisible()
  await expect(chooser.getByTestId('schedule-minutes-45')).toContainText(/оценка|estimate/)
  await expect(chooser.getByTestId('schedule-minutes-45')).toBeFocused()
  if (mobile) await page.screenshot({ path: `${SHOTS}/chooser-howlong-375.png` })

  await chooser.getByTestId('schedule-minutes-45').click()

  await expect.poll(() => sent?.starts_at, { timeout: 10_000 }).toBeTruthy()
  const start = new Date(sent!.starts_at!)
  const tomorrow = zoneParts(new Date(Date.now() + 24 * 3600_000)).date
  expect(zoneParts(start)).toEqual({ date: tomorrow, time: '09:00' })
  expect(Date.parse(sent!.ends_at!) - start.getTime()).toBe(45 * 60_000)
  expect(sent!.all_day).toBe(false)

  await expect(chooser).toHaveCount(0)
  await expect(page.getByRole('status')).toHaveText(/^(Запланировано: завтра|Scheduled: tomorrow) 09:00$/)
  await expect(row.getByTestId('task-scheduled-at')).toHaveText(/^(завтра|tomorrow) 09:00$/)
  if (mobile) await page.screenshot({ path: `${SHOTS}/row-after-375.png` })

  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
