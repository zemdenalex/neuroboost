import type { Page } from '@playwright/test'

/**
 * Show the page the account's own settings with `patch` laid over them, and
 * never let the page write settings back.
 *
 * 🔴 The e2e account is Denis's real staging account: a spec must not change
 * his settings to test a variant. The GET answer is rewritten in flight and
 * the PATCH is refused, so the variant exists only in this browser.
 */
export async function withSettings(page: Page, patch: Record<string, unknown>): Promise<void> {
  await page.route('**/api/auth/me', async (route) => {
    const method = route.request().method()
    if (method === 'PATCH' || method === 'PUT') return route.abort()
    if (method !== 'GET') return route.fallback()
    const res = await route.fetch()
    const body = await res.json()
    const user = body.data ?? body
    user.settings = { ...(user.settings ?? {}), ...patch }
    await route.fulfill({ response: res, json: body })
  })
}
