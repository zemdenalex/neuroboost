import type { Page } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * Gap list row 5 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md;
 * Denis 26.09 «link A + B»): a task becomes an event and an event a task,
 * linked or moved, one question per step as the bot asks it.
 *
 * Nothing touches the account: the task and the event are fixtures added to the
 * real GETs, convert and to-task are answered here, every other write is refused.
 */
const ZONE = 'Europe/Moscow'
const SHOTS = 'C:/Users/zd/AppData/Local/Temp/claude/E--Projects-007---Ventures-V003---NeuroBoost/04e1a014-f855-4c44-926c-c4003cfaca56/scratchpad/row5'
const now = new Date().toISOString()
const base = { status: 'TODO', actual_minutes: 0, tags: [], contexts: [], reminder_offsets: [10], user_id: 'e2e', created_at: now, updated_at: now }
const PLAIN = { ...base, id: 'e2e-link-plain', title: 'e2e связать с событием', priority: 2 }
const DAILY = {
  ...base, id: 'e2e-link-daily', title: 'e2e ежедневная в календарь', priority: 3, estimated_minutes: 45,
  rrule: 'FREQ=DAILY', repeat_anchor: '2026-09-01T00:00:00+03:00', due_date: '2026-09-01T00:00:00+03:00',
}

function zoneParts(at: Date) {
  const f = new Intl.DateTimeFormat('en-CA', { timeZone: ZONE, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' })
  const p = Object.fromEntries(f.formatToParts(at).map((x) => [x.type, x.value]))
  return { date: `${p.year}-${p.month}-${p.day}`, time: `${p.hour}:${p.minute}` }
}

async function stubAccount(page: Page, extraEvents: object[] = []) {
  // Registered first: Playwright tries the newest route first.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
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
    list.unshift(PLAIN, DAILY)
    await route.fulfill({ response: res, json: Array.isArray(body) ? list : { ...body, data: list } })
  })
  if (extraEvents.length > 0) {
    await page.route(/\/api\/events\?/, async (route) => {
      if (route.request().method() !== 'GET') return route.fallback()
      const res = await route.fetch()
      const body = await res.json()
      const list = Array.isArray(body) ? body : Array.isArray(body?.data) ? body.data : null
      if (!list) throw new Error(`GET /api/events answered in a shape the page does not read: ${JSON.stringify(body).slice(0, 200)}`)
      list.push(...extraEvents)
      await route.fulfill({ response: res, json: body })
    })
  }
}

async function openToEvent(page: Page, title: string, mobile: boolean) {
  const row = page.locator('[id^="task-"]').filter({ hasText: title }).first()
  await expect(row).toBeVisible({ timeout: 15_000 })
  if (mobile) {
    await row.getByTestId('task-row-more').click()
    await page.getByTestId('task-row-menu').getByTestId('task-to-event').click()
  } else {
    await row.hover()
    await row.getByTestId('task-to-event').click()
  }
  const sheet = page.getByTestId('link-sheet')
  await expect(sheet).toBeVisible()
  return sheet
}

test('a task is linked to an event: how, when, how long, the card, one confirm', async ({ authedPage: page }, testInfo) => {
  const mobile = testInfo.project.name === 'mobile'
  await stubAccount(page)
  let sent: Record<string, unknown> | undefined
  await page.route(`**/api/tasks/${PLAIN.id}/convert`, async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    sent = route.request().postDataJSON()
    await route.fulfill({
      status: 201, contentType: 'application/json',
      body: JSON.stringify({ data: { id: 'e2e-ev', calendar_id: 'c', title: PLAIN.title, starts_at: sent?.starts_at, ends_at: sent?.ends_at, all_day: false, task_id: PLAIN.id } }),
    })
  })

  await page.goto('/tasks')
  const sheet = await openToEvent(page, PLAIN.title, mobile)
  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 1 (из|of) 4/)
  if (mobile) await page.screenshot({ path: `${SHOTS}/how-375.png` })
  await sheet.getByTestId('link-how-link').click()

  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 2 (из|of) 4/)
  await sheet.getByTestId('link-when-tmr').click()
  // No estimate on this task: the length is asked (the bot's t2d_ step).
  await sheet.getByTestId('link-minutes-30').click()

  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 4 (из|of) 4/)
  await expect(sheet.getByTestId('link-card')).toContainText(/(когда|when): (завтра|tomorrow) 09:00/)
  await expect(sheet.getByTestId('link-card')).toContainText(/(приоритет|priority)/)
  // A link loses nothing: the task stays.
  await expect(sheet.getByTestId('link-lost')).toHaveCount(0)
  if (mobile) {
    await page.screenshot({ path: `${SHOTS}/card-375.png` })
    const tabBar = await page.getByTestId('tab-calendar').boundingBox()
    expect(tabBar, 'the phone tab bar is on screen under the sheet').not.toBeNull()
    // The sheet takes the taps, not the tab bar under it.
    if (tabBar) expect(await page.evaluate(([x, y]) => document.elementFromPoint(x, y)?.closest('[data-testid="link-sheet"]') !== null, [tabBar.x + tabBar.width / 2, tabBar.y + tabBar.height / 2])).toBe(true)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  }
  await sheet.getByTestId('link-confirm').click()

  await expect.poll(() => sent?.mode, { timeout: 10_000 }).toBe('link')
  expect(sent).not.toHaveProperty('repeat')
  expect(sent!.all_day).toBe(false)
  const start = new Date(String(sent!.starts_at))
  expect(zoneParts(start)).toEqual({ date: zoneParts(new Date(Date.now() + 24 * 3600_000)).date, time: '09:00' })
  expect(Date.parse(String(sent!.ends_at)) - start.getTime()).toBe(30 * 60_000)
  await expect(sheet).toHaveCount(0)
  await expect(page.getByRole('status')).toContainText(/(В календаре|On the calendar): (завтра|tomorrow) 09:00/)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('a repeating task moves one day: the series question, back, and a day the series lacks is asked again', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'the steps are the same sheet on both; the phone is covered above')
  await stubAccount(page)
  const bodies: Record<string, unknown>[] = []
  await page.route(`**/api/tasks/${DAILY.id}/convert`, async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const body = route.request().postDataJSON()
    bodies.push(body)
    // The first answer refuses as convert.go does for a day off the series.
    if (bodies.length === 1) {
      return route.fulfill({ status: 400, contentType: 'application/json', body: JSON.stringify({ error: { code: 'NOT_AN_OCCURRENCE', message: 'That day is not in the series' } }) })
    }
    await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ data: { id: 'e2e-ev2', calendar_id: 'c', title: DAILY.title, starts_at: body.starts_at, ends_at: body.ends_at, all_day: false } }) })
  })

  await page.goto('/tasks')
  const sheet = await openToEvent(page, DAILY.title, false)
  // how, repeat, when, card: the estimate (45) spares the length question.
  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 1 (из|of) 4/)
  await sheet.getByTestId('link-how-move').click()
  await expect(sheet.getByTestId('link-repeat-once')).toBeVisible()
  await sheet.getByTestId('link-back').click()
  await expect(sheet.getByTestId('link-how-move')).toBeVisible()
  await sheet.getByTestId('link-how-move').click()
  await sheet.getByTestId('link-repeat-once').click()
  await sheet.getByTestId('link-when-tmr').click()

  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 4 (из|of) 4/)
  await expect(sheet.getByTestId('link-outcome')).toHaveText(/(серия останется|the series stays)/)
  await sheet.getByTestId('link-confirm').click()

  // Refused: back to «when», with the bot's words.
  await expect(sheet.getByTestId('link-notice')).toHaveText(/(не входит в серию|not in the series)/)
  await expect(sheet.getByTestId('link-when-tmr')).toBeVisible()
  await sheet.getByTestId('link-when-eve').click()
  await sheet.getByTestId('link-confirm').click()

  await expect.poll(() => bodies.length, { timeout: 10_000 }).toBe(2)
  for (const b of bodies) {
    expect(b.mode).toBe('move')
    expect(b.repeat).toBe('once')
    expect(Date.parse(String(b.ends_at)) - Date.parse(String(b.starts_at))).toBe(45 * 60_000)
  }
  expect(zoneParts(new Date(String(bodies[1].starts_at))).time).toBe('19:00')
  await expect(sheet).toHaveCount(0)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('one day of a repeating event becomes a linked task from the editor: dry run card, then the same call for real', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport: the week grid opens the editor by double-click')
  const today = zoneParts(new Date()).date
  const parent = 'e2e0link-0000-0000-0000-000000000005'
  const title = 'e2e событие в задачу'
  const occurrence = {
    id: `${parent}:${today}`, title, starts_at: `${today}T10:00:00+03:00`, ends_at: `${today}T11:00:00+03:00`,
    all_day: false, rrule: 'FREQ=DAILY', timezone: ZONE, tags: [], reminder_offsets: [15], color: '#3b82f6',
    recurring_event_id: parent,
  }
  await stubAccount(page, [occurrence])
  const calls: { url: string; body: Record<string, unknown> }[] = []
  await page.route('**/api/events/*/to-task', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const body = route.request().postDataJSON()
    calls.push({ url: route.request().url(), body })
    await route.fulfill({
      status: body.dry_run ? 200 : 201, contentType: 'application/json',
      body: JSON.stringify({ data: {
        task: { id: body.dry_run ? undefined : 'e2e-new-task', title, calendar_id: 'c', tags: [], due_date: today, estimated_minutes: 60 },
        lost: ['start_time', 'color', 'reminders'], event_id: body.dry_run ? null : 'e2e-detached', dry_run: !!body.dry_run,
      } }),
    })
  })

  await page.goto('/calendar')
  const block = page.locator(`[title^="${title}"]`).first()
  await expect(block).toBeVisible({ timeout: 15_000 })
  await block.dblclick()
  await page.getByTestId('event-to-task').click()

  const sheet = page.getByTestId('link-sheet')
  await expect(sheet.getByTestId('link-step')).toHaveText(/(Шаг|Step) 1 (из|of) 3/)
  await sheet.getByTestId('link-how-link').click()
  // Opened from one day, so «just this once» is offered.
  await sheet.getByTestId('link-repeat-once').click()
  await expect(sheet.getByTestId('link-card')).toContainText(/(час начала|start hour)/)
  await expect(sheet.getByTestId('link-lost')).toHaveCount(3)
  await expect(sheet.getByTestId('link-outcome')).toHaveText(/(отдельным событием|its own event)/)
  await sheet.getByTestId('link-confirm').click()

  await expect.poll(() => calls.length, { timeout: 10_000 }).toBe(2)
  // The occurrence rides in the id, unencoded, as the bot sends it.
  expect(calls[0].url).toContain(`/api/events/${parent}:${today}/to-task`)
  expect(calls[0].body).toEqual({ mode: 'link', repeat: 'once', dry_run: true })
  expect(calls[1].body).toEqual({ mode: 'link', repeat: 'once' })
  // Link of one day detaches it: the editor closes.
  await expect(sheet).toHaveCount(0)
  await expect(page.getByTestId('event-editor')).toHaveCount(0)
  await expect(page.getByRole('status')).toContainText(title)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
