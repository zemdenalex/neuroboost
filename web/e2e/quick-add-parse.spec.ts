import type { Page, Route } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * Gap list row 1 (docs/team/research/V003-20260926-res-bot-vs-web-gaps.md,
 * Denis 26.09: «A: one parser, API endpoint»): a line typed into quick add is
 * read by the bot's parser through POST /api/parse.
 *
 * The parse answer is mocked here: staging gets the route only once it is
 * pushed, and the parser itself is tested in Go (api-go/internal/lineparse).
 * What this spec proves is the web half: what the row does with each answer.
 * Nothing is written to the account: every create is caught and answered
 * here, every other write is refused.
 */

interface Writes {
  tasks: Record<string, unknown>[]
  events: Record<string, unknown>[]
}

const parsedBase = {
  timezone: 'Europe/Moscow', tags: [], rrule: null, priority: null, due_date: null,
  estimated_minutes: null, starts_at: null, ends_at: null, all_day: false, color: null,
  calendar_id: null, calendar_name: null, reminder_offsets: null, is_task: false, uncertain: [], has_time: false,
}

async function catchWrites(
  page: Page,
  parse?: Record<string, unknown> | ((text: string) => Record<string, unknown>),
  opts: { failEventsOnce?: boolean } = {},
): Promise<Writes> {
  const writes: Writes = { tasks: [], events: [] }
  // Registered first: Playwright tries the newest route first, so this is the
  // fallback for anything the routes below do not answer.
  await page.route('**/api/**', (route) =>
    ['GET', 'HEAD', 'OPTIONS'].includes(route.request().method()) ? route.fallback() : route.abort(),
  )
  const answer = (route: Route, data: Record<string, unknown>) =>
    route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ data }) })
  const stamp = new Date().toISOString()
  await page.route('**/api/tasks', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const sent = route.request().postDataJSON() as Record<string, unknown>
    writes.tasks.push(sent)
    await answer(route, { id: `e2e-task-${writes.tasks.length}`, status: 'TODO', priority: 3, tags: [], contexts: [], created_at: stamp, updated_at: stamp, ...sent })
  })
  await page.route('**/api/events', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    const sent = route.request().postDataJSON() as Record<string, unknown>
    writes.events.push(sent)
    if (opts.failEventsOnce && writes.events.length === 1) {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { code: 'E2E', message: 'e2e failure' } }) })
      return
    }
    await answer(route, { id: `e2e-event-${writes.events.length}`, ...sent })
  })
  if (parse) {
    await page.route('**/api/parse', (route) => {
      const text = String((route.request().postDataJSON() as { text?: string }).text ?? '')
      const data = typeof parse === 'function' ? parse(text) : parse
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { ...parsedBase, ...data } }) })
    })
  }
  return writes
}

async function typeLine(page: Page, line: string) {
  await page.goto('/tasks')
  const input = page.getByRole('textbox', { name: /Новая задача: введи|New task: type/ })
  await input.fill(line)
  await input.press('Enter')
  return input
}

test('a line with a time is confirmed before anything is written, then becomes an event', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough: the same row on both')
  const writes = await catchWrites(page, {
    kind: 'event', title: 'стоматолог',
    starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z', reminder_offsets: [60],
  })

  const input = await typeLine(page, 'стоматолог завтра 15:00 напомни за час')
  const confirm = page.getByTestId('quick-add-confirm')
  await expect(confirm).toBeVisible()
  await expect(confirm).toContainText('стоматолог')
  // Wall time in the zone the line was read in: 12:00Z is 15:00 in Moscow.
  await expect(page.getByTestId('quick-add-confirm-when')).toContainText('15:00')
  expect(writes.tasks.length + writes.events.length, 'nothing is written before the confirmation').toBe(0)

  // The second Enter on the unchanged line creates what was shown.
  await input.press('Enter')
  await expect.poll(() => writes.events.length, { timeout: 10_000 }).toBe(1)
  expect(writes.events[0]).toMatchObject({
    title: 'стоматолог', starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z',
    all_day: false, reminder_offsets: [60],
  })
  expect(writes.events[0]).not.toHaveProperty('task_id')
  expect(writes.tasks).toHaveLength(0)
  await expect(confirm).toBeHidden()
  await expect(input).toHaveValue('')
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('Esc drops the confirmation and writes nothing', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page, {
    kind: 'event', title: 'созвон', starts_at: '2026-09-27T08:00:00Z', ends_at: '2026-09-27T09:00:00Z',
  })
  const input = await typeLine(page, 'созвон завтра 11:00')
  await expect(page.getByTestId('quick-add-confirm')).toBeVisible()
  await input.press('Escape')
  await expect(page.getByTestId('quick-add-confirm')).toBeHidden()
  await expect(input).toHaveValue('созвон завтра 11:00')
  expect(writes.tasks.length + writes.events.length).toBe(0)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('a line without a time is saved as a task at once, with what the line said', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page, {
    kind: 'task', title: 'купить молоко', priority: 1, estimated_minutes: 30,
    due_date: '2026-09-26T21:00:00Z', tags: ['дом'],
  })
  await typeLine(page, 'купить молоко завтра !1 30м #дом')

  await expect.poll(() => writes.tasks.length, { timeout: 10_000 }).toBe(1)
  expect(writes.tasks[0]).toMatchObject({
    title: 'купить молоко', priority: 1, estimated_minutes: 30, due_date: '2026-09-26T21:00:00Z',
  })
  expect(writes.tasks[0].tags).toContain('дом')
  expect(writes.events).toHaveLength(0)
  await expect(page.getByTestId('quick-add-confirm')).toHaveCount(0)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('with the parser unreachable the line is saved as typed, as before', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  // No parse route: the catch-all aborts POST /api/parse.
  const writes = await catchWrites(page)
  // A line without a clock time; a timed one asks first (the test below).
  await typeLine(page, 'купить молоко завтра')

  await expect.poll(() => writes.tasks.length, { timeout: 10_000 }).toBe(1)
  expect(writes.tasks[0].title).toBe('купить молоко завтра')
  expect(writes.events).toHaveLength(0)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('a timed line the bot would ask about is not saved as typed: the day is asked, then it is an event', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page, (text) =>
    text.endsWith('завтра')
      ? { kind: 'event', title: 'встреча', starts_at: '2026-09-27T12:00:00Z', ends_at: '2026-09-27T13:00:00Z', has_time: true }
      : { kind: 'ask', missing: 'date', title: 'встреча', has_time: true },
  )
  const input = await typeLine(page, 'встреча 15:00')
  const confirm = page.getByTestId('quick-add-confirm')
  await expect(confirm).toBeVisible()
  await expect(confirm).toContainText(/Нужен день|Which day/)
  // A second Enter does not save the raw line either.
  await input.press('Enter')
  await page.waitForTimeout(300)
  expect(writes.tasks.length + writes.events.length, 'nothing is written while the day is unknown').toBe(0)

  await page.getByTestId('quick-add-day-tomorrow').click()
  await expect(page.getByTestId('quick-add-confirm-when')).toContainText('15:00')
  await input.press('Enter')
  await expect.poll(() => writes.events.length, { timeout: 10_000 }).toBe(1)
  expect(writes.events[0]).toMatchObject({ title: 'встреча', starts_at: '2026-09-27T12:00:00Z' })
  expect(writes.tasks).toHaveLength(0)
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

test('as a task: one task and an event bound to it, and a failed event does not make a second task', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page, {
    kind: 'event', title: 'отчёт', starts_at: '2026-09-27T07:00:00Z', ends_at: '2026-09-27T08:00:00Z', tags: ['работа'],
  }, { failEventsOnce: true })
  const input = await typeLine(page, 'отчёт завтра 10:00')
  await page.getByTestId('quick-add-confirm-other').click()
  // The event failed: the task is there, and the panel says so.
  await expect(page.getByTestId('quick-add-confirm')).toContainText(/e2e failure/)
  expect(writes.tasks).toHaveLength(1)
  expect(writes.events).toHaveLength(1)

  // Retry with Enter: the same task, a second event attempt, no second task.
  await input.press('Enter')
  await expect.poll(() => writes.events.length, { timeout: 10_000 }).toBe(2)
  expect(writes.tasks).toHaveLength(1)
  expect(writes.tasks[0]).toMatchObject({ title: 'отчёт', status: 'TODO', tags: ['работа'] })
  expect(writes.events[1]).toMatchObject({ title: 'отчёт', task_id: 'e2e-task-1' })
  await expect(page.getByTestId('quick-add-confirm')).toBeHidden()
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

// Review of b49bcc8: with the parser down (here: /api/parse refused like every
// other unanswered write), a line with a clock time is not saved unread. The
// first Enter says so; the second saves it as typed.
test('a timed line is not saved unread when the parser is down', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page)
  const input = await typeLine(page, 'стоматолог завтра 15:00')
  await expect(page.getByTestId('quick-add-unread-time')).toBeVisible()
  expect(writes.tasks, 'nothing written on the first Enter').toHaveLength(0)
  await input.press('Enter')
  await expect.poll(() => writes.tasks.length, { timeout: 10_000 }).toBe(1)
  expect(writes.tasks[0]).toMatchObject({ title: 'стоматолог завтра 15:00' })
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

// Review of b49bcc8: text typed while the line is being read was wiped by the
// save. The input is read-only for that moment instead.
test('the input is read-only while the line is being read', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page)
  await page.route('**/api/parse', async (route) => {
    await new Promise((r) => setTimeout(r, 1200))
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { ...parsedBase, kind: 'task', title: 'купить хлеб' } }) })
  })
  const input = await typeLine(page, 'купить хлеб')
  await expect(input).toHaveAttribute('readonly', '')
  await expect.poll(() => writes.tasks.length, { timeout: 10_000 }).toBe(1)
  await expect(input).not.toHaveAttribute('readonly', '')
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})

// Gap list row 18: after a quick save the bot offers «↩️ Отменить»; the toast
// offers Undo, which deletes the task just made.
test('Undo in the toast takes back the task just made', async ({ authedPage: page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', 'one viewport is enough')
  const writes = await catchWrites(page, { kind: 'task', title: 'купить хлеб' })
  let deleted = ''
  await page.route('**/api/tasks/e2e-task-*', async (route) => {
    if (route.request().method() !== 'DELETE') return route.fallback()
    deleted = new URL(route.request().url()).pathname
    await route.fulfill({ status: 204, body: '' })
  })
  await typeLine(page, 'купить хлеб')
  await expect.poll(() => writes.tasks.length, { timeout: 10_000 }).toBe(1)
  await page.getByRole('button', { name: /^(Undo|Отменить)$/ }).click()
  await expect.poll(() => deleted, { timeout: 10_000 }).toBe('/api/tasks/e2e-task-1')
  await page.unrouteAll({ behavior: 'ignoreErrors' })
})
