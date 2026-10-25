import { test, expect } from './fixtures/auth'

/**
 * Tools switched off in Settings: gone from the menu and closed by URL
 * (audit 24.09, T0c). The flag is patched into /auth/me in the page, and a
 * settings write is blocked: the e2e account is a real person's.
 */
test('switched-off tools are hidden and closed by URL', async ({ authedPage }, info) => {
  test.skip(info.project.name !== 'desktop', 'the header menu is the desktop one')
  await authedPage.route('**/api/auth/me', (route) =>
    route.request().method() === 'PATCH' ? route.abort() : route.fallback(),
  )
  await authedPage.addInitScript(() => {
    const orig = window.fetch.bind(window)
    window.fetch = async (input, init) => {
      const res = await orig(input, init)
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (!url.includes('/api/auth/me') || (init?.method && init.method !== 'GET')) return res
      const body = await res.clone().json()
      const user = body.data ?? body
      user.settings = { ...(user.settings ?? {}), features: { ...(user.settings?.features ?? {}), tools: false } }
      return new Response(JSON.stringify(body.data ? { ...body, data: user } : user), {
        status: res.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }
  })
  await authedPage.goto('/tools/kanban')
  await expect(authedPage).toHaveURL(/\/home$/, { timeout: 15_000 })
  await expect(authedPage.getByRole('link', { name: /^(Tools|Инструменты)$/ })).toHaveCount(0)
})
