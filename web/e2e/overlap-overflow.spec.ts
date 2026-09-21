import { request as playwrightRequest } from '@playwright/test'
import { test, expect } from './fixtures/auth'
import { localMidnightUtc } from './fixtures/localTime'

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

    // Today IN THE ACCOUNT'S ZONE, so no week navigation is needed to see it.
    //
    // 🔴 This was `new Date(); d.setHours(h)` — the runner's clock, which is
    // UTC. From 21:00 UTC the account (Moscow) is already on tomorrow, the
    // 375px grid shows only that day, and the eight events were drawn on a day
    // nobody was looking at: every run in that window failed with «no seeded
    // blocks rendered at all» (21.09, 00:03 MSK). shared-badge-mobile fixed the
    // same line the same way.
    //
    // 🔴 And each VIEWPORT gets its own hours and its own tag. The desktop and
    // mobile runs of this spec go in parallel as the same user: with one band
    // both seeded 10:00–12:00, sixteen events shared one lane, every block came
    // out 7px wide, and the desktop run failed «16 of 16 blocks have content
    // wider than themselves» (22.09) — measuring the other run's events as its
    // own. Bands: desktop 10:00–12:00, mobile 18:00–20:00 (fixtures/localTime.ts).
    const project = test.info().project.name
    const startHour = project === 'mobile' ? 18 : 10
    const tag = `перекрытие ${project}`
    const me = (await (await api.get(`${API_BASE}/api/auth/me`)).json()).data
    const dayStart = localMidnightUtc(me.timezone || 'Europe/Moscow', 0)
    const iso = (h: number, m = 0) => new Date(dayStart + (h * 60 + m) * 60_000).toISOString()

    // Everything overlaps everything: one cluster, eight columns.
    const created: string[] = []
    try {
      for (let i = 0; i < 8; i++) {
        const res = await api.post(`${API_BASE}/api/events`, {
          data: {
            title: `${tag} ${i + 1}`,
            starts_at: iso(startHour, i * 5),
            ends_at: iso(startHour + 2),
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

      const geometry = await authedPage.evaluate((tag) => {
        const blocks = Array.from(
          document.querySelectorAll<HTMLElement>('[data-testid="event-block"]'),
        ).filter((b) => (b.textContent ?? '').includes(tag))
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
      }, tag)

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

      // 🔴 The one Denis actually saw: eight narrow columns wrapped a title to
      // one character per line and the tower ran down over the hours below.
      //
      // ⚠ NOT asserted with scrollHeight. `overflow: hidden` clips what is
      // PAINTED and leaves scrollHeight reporting the full content height, so a
      // scrollHeight assertion stays red after the bug is fixed and red after it
      // regresses — it cannot tell the two apart. What the user sees is what is
      // painted, so the test asks what is painted: sample a point just below
      // each block and check the block is not what answers there.
      const painted = await authedPage.evaluate((tag) => {
        const blocks = Array.from(
          document.querySelectorAll<HTMLElement>('[data-testid="event-block"]'),
        ).filter((b) => (b.textContent ?? '').includes(tag))
        return blocks.map((b) => {
          const r = b.getBoundingClientRect()
          const below = document.elementFromPoint(r.left + r.width / 2, r.bottom + 12)
          return { title: (b.textContent ?? '').slice(0, 14), leaks: b.contains(below) }
        })
      }, tag)
      const leaking = painted.filter((p) => p.leaks)
      expect(
        leaking,
        `${leaking.length} of ${painted.length} blocks still paint 12px below themselves: ` +
          JSON.stringify(leaking.map((p) => p.title)),
      ).toHaveLength(0)
    } finally {
      for (const id of created) {
        await api.delete(`${API_BASE}/api/events/${id}`)
      }
      await api.dispose()
    }
  })
})
