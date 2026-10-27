import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'
import { localMidnightUtc } from './fixtures/localTime'

/**
 * A 15-minute event showed only the top half of its title (friction audit
 * 26.10, #4): the block is ~11px, the title line was ~15px, and overflow-hidden
 * cut it. jsdom cannot measure layout, so this asks the rendered page whether
 * the title row fits inside the block.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.space'
const HOUR = 3600 * 1000

test('the title of a 15-minute event fits inside its block', async ({ authedPage, session }) => {
  const ctx = await playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${session.token}`, 'Content-Type': 'application/json' },
  })
  const me = (await (await ctx.get('/api/auth/me')).json()).data
  const dayStart = localMidnightUtc(me.timezone || 'Europe/Moscow', 0)
  const startMs = dayStart + 5 * HOUR
  const title = `E2E short ${Date.now()}`
  const created = await ctx.post('/api/events', {
    data: {
      title,
      starts_at: new Date(startMs).toISOString(),
      ends_at: new Date(startMs + 15 * 60 * 1000).toISOString(),
      all_day: false,
    },
  })
  expect(created.ok(), `create failed: ${created.status()}`).toBe(true)
  const id = (await created.json()).data.id

  try {
    await authedPage.goto('/calendar')
    const block = authedPage.locator(`[title^="${title}"]`).first()
    await expect(block).toBeVisible({ timeout: 15_000 })
    await block.evaluate((el) => el.scrollIntoView({ block: 'center', behavior: 'instant' }))
    const b = (await block.boundingBox())!
    const row = (await block.getByTestId('event-title').boundingBox())!
    // Positive control: the block really is a short one, or fitting proves nothing.
    expect(b.height, 'a 15-minute block should be under 16px').toBeLessThan(16)
    expect(row.y, 'title row starts above the block').toBeGreaterThanOrEqual(b.y - 0.5)
    expect(row.y + row.height, 'title row runs past the bottom of the block').toBeLessThanOrEqual(b.y + b.height + 0.5)
  } finally {
    await ctx.delete(`/api/events/${id}`)
    await ctx.dispose()
  }
})
