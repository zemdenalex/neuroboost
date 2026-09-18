import { request as playwrightRequest } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * «Что дальше» on a phone.
 *
 * The view exists because neither the web nor the bot answered "what is next"
 * (docs/razbor-mobilnyy-kalendar-2026-08-19.md §3, option B). So the spec checks
 * the two things that would make it useless: that it cannot be reached, and
 * that it does not fit the screen.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

test.describe('agenda', () => {
  test('is reachable from the phone and lists what is coming', async ({ authedPage, session }) => {
    const api = await playwrightRequest.newContext({
      extraHTTPHeaders: { Authorization: `Bearer ${session.token}` },
    })

    const soon = new Date()
    soon.setDate(soon.getDate() + 1)
    soon.setHours(11, 0, 0, 0)
    const ends = new Date(soon)
    ends.setHours(12, 0, 0, 0)

    const created: string[] = []
    try {
      const res = await api.post(`${API_BASE}/api/events`, {
        data: {
          // 🔴 A long unbreakable title on purpose: at 375px this is what tows
          // a row wider than the screen, which is the defect the week grid had.
          title: 'повесткаповесткаповесткаповесткаповестка',
          starts_at: soon.toISOString(),
          ends_at: ends.toISOString(),
          all_day: false,
        },
      })
      expect(res.ok(), `seeding: ${res.status()} ${await res.text()}`).toBeTruthy()
      created.push((await res.json()).data.id)

      await authedPage.goto('/agenda')
      await authedPage.waitForSelector('[data-testid="agenda"]', { timeout: 20_000 })
      await expect(authedPage.locator('[data-testid="agenda-error"]')).toHaveCount(0)
      await authedPage.waitForSelector('[data-testid="agenda-item"]', { timeout: 15_000 })

      const spill = await authedPage.evaluate(() => ({
        doc: document.documentElement.scrollWidth,
        win: window.innerWidth,
      }))
      expect(
        spill.doc,
        `the agenda scrolls sideways: content ${spill.doc}px in a ${spill.win}px window`,
      ).toBeLessThanOrEqual(spill.win + 1)

      const rows = await authedPage.locator('[data-testid="agenda-item"]').count()
      expect(rows, 'the seeded event is not listed').toBeGreaterThan(0)
    } finally {
      for (const id of created) {
        await api.delete(`${API_BASE}/api/events/${id}`)
      }
      await api.dispose()
    }
  })
})
