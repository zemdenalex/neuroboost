import { test, expect } from './fixtures/auth'
import { request as playwrightRequest } from '@playwright/test'

/**
 * A new event gets the user's default reminder preset.
 *
 * 🔴 It did not, and nothing noticed for weeks. The editor always sent
 * `reminder_offsets` — `[]` when untouched — and createEvent applies
 * DefaultEventOffsets only when the field is ABSENT (that fallback exists for
 * the bot and the importer, which genuinely omit it). So every event created in
 * the web was stored with an empty array, and an empty array is not "use the
 * default": the scanner skips those rows outright
 * (`cardinality(reminder_offsets) > 0` in reminders/scan.go). The event never
 * reminded and nothing reported it.
 *
 * Found by creating an event by hand on staging and reading the row back — the
 * checklist item "change the default preset, a new event gets its offsets" had
 * been on the list since v0.4.10 and had never been performed.
 *
 * TASKS were always fine, which is why this is easy to miss by reading: the
 * task path omits the field, so the server's fallback applies. Only events sent
 * the empty array. The assertion below therefore covers events specifically.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

test.describe('default reminder preset', () => {
  // One viewport: this creates an event on the shared test account and deletes
  // it again, and two copies racing would delete each other's row.
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'writes to the shared account; one viewport only')
  })

  test('a new event is created with the default preset already applied', async ({
    authedPage,
    session,
  }) => {
    const ctx = await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: { Authorization: `Bearer ${session.token}`, 'Content-Type': 'application/json' },
    })

    // What the account's default preset actually resolves to, read rather than
    // assumed: this test must not hardcode "обычное", which a user may have
    // renamed — renaming is a supported operation and repoints the default.
    const me = (await (await ctx.get('/api/auth/me')).json()).data
    const reminders = me.settings?.reminders ?? {}
    const presets: Record<string, number[]> = reminders.presets ?? {
      'важное': [43200, 10080, 4320, 1440, 60],
      'обычное': [1440, 60],
      'без': [],
    }
    const defaultName: string = reminders.default_event_preset ?? 'обычное'
    const expected = presets[defaultName] ?? []

    // A default of "no reminders" would make every assertion below vacuous:
    // the empty array is exactly the broken state. Skip loudly instead.
    test.skip(
      expected.length === 0,
      `the account's default event preset (${defaultName}) has no offsets, so this test could not fail`,
    )

    const title = `E2E default reminders ${Date.now()}`
    let createdId: string | undefined

    try {
      await authedPage.goto('/calendar')
      await authedPage.waitForLoadState('networkidle')

      await authedPage.getByRole('button', { name: /quick/i }).click()
      await authedPage.getByPlaceholder(/event title/i).fill(title)
      await authedPage.getByRole('button', { name: /^create$/i }).click()

      // Read the row back rather than trusting the screen: the defect was
      // invisible in the UI — the form said "No reminders" and saved happily.
      await expect
        .poll(
          async () => {
            const from = new Date(Date.now() - 3 * 86400000).toISOString()
            const to = new Date(Date.now() + 3 * 86400000).toISOString()
            const res = await ctx.get(`/api/events?start=${from}&end=${to}`)
            const events = (await res.json()).data ?? []
            const mine = events.find((e: { title: string }) => e.title === title)
            createdId = mine?.id
            return mine?.reminder_offsets ?? null
          },
          { message: `the event created in the web did not get the ${defaultName} preset` },
        )
        .toEqual(expected)
    } finally {
      if (createdId) await ctx.delete(`/api/events/${createdId}`)
      await ctx.dispose()
    }
  })
})
