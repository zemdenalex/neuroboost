import { test } from './fixtures/auth'

/**
 * Not a test: a screenshot tour of every screen at 375px, for the eye pass the
 * visual build loop asks for (E:/Projects/.claude/rules/visual-build-loop.md).
 * mobile-overflow.spec.ts proves nothing scrolls sideways; it cannot see a
 * cramped header, a tap target 20px wide or a half-empty screen. This can.
 *
 * Off unless NB_TOUR=1, so CI never runs it. Screens land in NB_TOUR_DIR
 * (default test-results/mobile-tour).
 *
 *   NB_TOUR=1 web/scripts/e2e-local.sh --project mobile e2e/mobile-tour.spec.ts
 */

const ROUTES = [
  '/home',
  '/calendar',
  '/agenda',
  '/tasks',
  '/day-tasks',
  '/planning',
  '/reflections',
  '/tools',
  '/tools/pomodoro',
  '/tools/kanban',
  '/tools/eisenhower',
  '/tools/time-blocking',
  '/settings',
  '/profile',
]

test.describe('mobile tour', () => {
  test.beforeEach(({}, testInfo) => {
    test.skip(process.env.NB_TOUR !== '1', 'screenshot tour, run by hand')
    test.skip(testInfo.project.name !== 'mobile', 'mobile viewport only')
  })

  for (const route of ROUTES) {
    test(`tour ${route}`, async ({ authedPage }) => {
      const dir = process.env.NB_TOUR_DIR ?? 'test-results/mobile-tour'
      const name = route.replace(/\//g, '_').replace(/^_/, '')
      test.setTimeout(90_000)
      // Not networkidle: the vite dev server's HMR socket keeps the network
      // busy forever, and a first-compile page can take 20 s.
      await authedPage.goto(route, { waitUntil: 'load', timeout: 60_000 })
      await authedPage.waitForTimeout(3000)
      await authedPage.screenshot({ path: `${dir}/${name}.png` })
      await authedPage.screenshot({ path: `${dir}/${name}-full.png`, fullPage: true })
    })
  }
})
