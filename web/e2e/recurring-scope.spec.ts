import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * The R1 dialog — "this event" vs "all events" before mutating a repeating
 * event — shipped verified by reading JSON. It had never been opened in a
 * browser. This spec opens it.
 *
 * The fixture data is created through the API rather than by clicking: fewer
 * moving parts between "a repeating event exists" and the thing under test,
 * and a failure here then means the dialog is broken, not that event creation
 * is.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

// Both locales, because the app follows the account's language and the point
// of the assertion is "the right dialog appeared", not "the app is in English".
const DELETE_TITLE = /Delete repeating event|Удалить повторяющееся событие/
const THIS_EVENT = /^(This event|Только это событие)$/
const ALL_EVENTS = /^(All events|Все повторы)$/
const DELETE_ACTION = /^(Delete|Удалить)$/

/**
 * The editor guards deletion with a native window.confirm before the scope
 * dialog is ever reached (useEditorForm.ts handleDelete). Playwright dismisses
 * native dialogs by default, so without this the delete silently no-ops and the
 * failure reads as "the scope dialog is broken" — which is exactly the wrong
 * conclusion.
 */
function acceptNativeConfirms(page: import('@playwright/test').Page) {
  page.on('dialog', (d) => {
    void d.accept()
  })
}

async function apiContext(token: string): Promise<APIRequestContext> {
  return playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  })
}

/** A daily-repeating event at a fixed hour today, titled uniquely per run. */
async function createRecurringEvent(token: string, title: string) {
  const ctx = await apiContext(token)
  try {
    const start = new Date()
    start.setUTCHours(9, 0, 0, 0)
    const end = new Date(start.getTime() + 30 * 60 * 1000)

    const res = await ctx.post('/api/events', {
      data: {
        title,
        starts_at: start.toISOString(),
        ends_at: end.toISOString(),
        rrule: 'FREQ=DAILY;COUNT=5',
        all_day: false,
      },
    })
    if (!res.ok()) throw new Error(`create failed: ${res.status()} ${await res.text()}`)
    return (await res.json()).data
  } finally {
    await ctx.dispose()
  }
}

async function deleteEvent(token: string, id: string) {
  const ctx = await apiContext(token)
  try {
    await ctx.delete(`/api/events/${id}?scope=series`)
  } finally {
    await ctx.dispose()
  }
}

test.describe('recurring scope dialog', () => {
  let createdId: string | undefined
  let token: string | undefined

  test.afterEach(async () => {
    // Clean up even when the test failed, so a red run does not leave repeating
    // events on the account this project is meant to be usable from.
    if (createdId && token) {
      await deleteEvent(token, createdId)
      createdId = undefined
    }
  })

  test('deleting a repeating event asks which occurrences it applies to', async ({ authedPage, session }) => {
    token = session.token
    const title = `E2E recurring ${Date.now()}`
    const created = await createRecurringEvent(session.token, title)
    createdId = created.id

    acceptNativeConfirms(authedPage)
    await authedPage.goto('/calendar')
    await authedPage.waitForLoadState('networkidle')

    // Event blocks expose their title as the tooltip; double-click opens the
    // editor (single click only selects).
    const block = authedPage.locator(`[title^="${title}"]`).first()
    await expect(block, 'the created repeating event should render on the calendar').toBeVisible({ timeout: 15_000 })
    await block.dblclick()

    const deleteButton = authedPage.getByRole('button', { name: DELETE_ACTION })
    await expect(deleteButton, 'the editor should open with a delete action').toBeVisible({ timeout: 10_000 })
    await deleteButton.click()

    const dialog = authedPage.getByRole('dialog')
    await expect(dialog, 'R1: the scope dialog must appear before a repeating event is deleted').toBeVisible()
    await expect(dialog).toContainText(DELETE_TITLE)
    await expect(dialog.getByRole('button', { name: THIS_EVENT })).toBeVisible()
    await expect(dialog.getByRole('button', { name: ALL_EVENTS })).toBeVisible()
  })

  test('Escape abandons the deletion instead of applying a default', async ({ authedPage, session }) => {
    token = session.token
    const title = `E2E escape ${Date.now()}`
    const created = await createRecurringEvent(session.token, title)
    createdId = created.id

    acceptNativeConfirms(authedPage)
    await authedPage.goto('/calendar')
    await authedPage.waitForLoadState('networkidle')

    const block = authedPage.locator(`[title^="${title}"]`).first()
    await expect(block).toBeVisible({ timeout: 15_000 })
    await block.dblclick()
    await authedPage.getByRole('button', { name: DELETE_ACTION }).click()

    const dialog = authedPage.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await authedPage.keyboard.press('Escape')

    await expect(dialog, 'Escape must close the dialog').not.toBeVisible()
    // The event must survive: a cancelled dialog that still deleted would be
    // the worst possible outcome of this feature.
    await authedPage.reload()
    await authedPage.waitForLoadState('networkidle')
    await expect(authedPage.locator(`[title^="${title}"]`).first()).toBeVisible({ timeout: 15_000 })
  })
})
