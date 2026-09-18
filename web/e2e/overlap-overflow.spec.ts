import { request as playwrightRequest } from '@playwright/test'
import { test, expect } from './fixtures/auth'

/**
 * Denis, 18.09: «пофикси выход за поля в вебе в событиях, чем больше событий
 * тем больше они за поля выходят».
 *
 * 🔴 This spec MEASURES rather than looks. The thing this project keeps
 * relearning is that a red screen gets explained before it gets measured, and
 * the explanation is wrong about half the time — twice in one night on 17.09.
 * So: seed a known number of overlapping events, read the real bounding boxes
 * out of the browser, and compare them against the column meant to hold them.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

test.describe('overlapping events stay inside their day column', () => {
  test('eight overlapping events do not leave the column', async ({ authedPage, session }) => {
    const api = await playwrightRequest.newContext({
      extraHTTPHeaders: { Authorization: `Bearer ${session.token}` },
    })

    // Today, so no week navigation is needed to see it.
    const iso = (h: number, m = 0) => {
      const d = new Date()
      d.setHours(h, m, 0, 0)
      return d.toISOString()
    }

    // Everything overlaps everything: one cluster, eight columns.
    const created: string[] = []
    try {
      for (let i = 0; i < 8; i++) {
        const res = await api.post(`${API_BASE}/api/events`, {
          data: {
            title: `перекрытие ${i + 1}`,
            starts_at: iso(10, i * 5),
            ends_at: iso(12),
            all_day: false,
          },
        })
        expect(res.ok(), `seeding event ${i + 1}: ${res.status()} ${await res.text()}`).toBeTruthy()
        created.push((await res.json()).data.id)
      }

      await authedPage.goto('/calendar')
      await authedPage.waitForSelector('[data-testid="event-block"]', { timeout: 20_000 })
      // Let the layout settle: the blocks are positioned from measured heights.
      await authedPage.waitForTimeout(1000)

      const geometry = await authedPage.evaluate(() => {
        const blocks = Array.from(
          document.querySelectorAll<HTMLElement>('[data-testid="event-block"]'),
        ).filter((b) => (b.textContent ?? '').includes('перекрытие'))
        if (blocks.length === 0) return null
        const column = blocks[0].parentElement as HTMLElement
        const col = column.getBoundingClientRect()
        return {
          count: blocks.length,
          column: { left: col.left, right: col.right, width: col.width },
          blocks: blocks.map((b) => {
            const r = b.getBoundingClientRect()
            // scrollWidth > clientWidth means the CONTENT does not fit, which is
            // the other reading of «выходят за поля»: the box is inside the
            // column but the text inside it is not inside the box.
            return {
              left: r.left,
              right: r.right,
              width: r.width,
              scrollWidth: b.scrollWidth,
              clientWidth: b.clientWidth,
              scrollHeight: b.scrollHeight,
              clientHeight: b.clientHeight,
            }
          }),
        }
      })

      expect(geometry, 'no seeded blocks rendered at all').not.toBeNull()
      const g = geometry!
      console.log('column', JSON.stringify(g.column))
      console.log('blocks', JSON.stringify(g.blocks, null, 1))

      const outside = g.blocks.filter((b) => b.right > g.column.right + 1 || b.left < g.column.left - 1)
      expect(
        outside,
        `${outside.length} of ${g.count} blocks leave the column ` +
          `(column ${g.column.left.toFixed(1)}–${g.column.right.toFixed(1)}): ` +
          JSON.stringify(outside.map((b) => `${b.left.toFixed(1)}–${b.right.toFixed(1)}`)),
      ).toHaveLength(0)

      const spillingWide = g.blocks.filter((b) => b.scrollWidth > b.clientWidth + 1)
      expect(
        spillingWide,
        `${spillingWide.length} of ${g.count} blocks have content wider than themselves: ` +
          JSON.stringify(spillingWide.map((b) => `${b.clientWidth}<${b.scrollWidth}`)),
      ).toHaveLength(0)

      // 🔴 The one Denis actually saw: eight narrow columns wrap a title to one
      // character per line, and the tower runs down over the hours below.
      const spillingTall = g.blocks.filter((b) => b.scrollHeight > b.clientHeight + 1)
      expect(
        spillingTall,
        `${spillingTall.length} of ${g.count} blocks have content taller than themselves: ` +
          JSON.stringify(spillingTall.map((b) => `${b.clientHeight}<${b.scrollHeight}`)),
      ).toHaveLength(0)
    } finally {
      for (const id of created) {
        await api.delete(`${API_BASE}/api/events/${id}`)
      }
      await api.dispose()
    }
  })
})
