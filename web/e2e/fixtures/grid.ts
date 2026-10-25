import type { Page } from '@playwright/test'

/**
 * Puts the week grid back at 00:00, instantly.
 *
 * Since 25.09 (MW11) the grid opens near the current hour. The drag specs were
 * written for a grid at the top: their events sit in time bands chosen to be
 * on the first screen (fixtures/localTime.ts). Rather than chase each event
 * with scrollIntoView, which goes through `html { scroll-behavior: smooth }`
 * and can hand back coordinates mid-animation, the grid is reset to the state
 * those bands were chosen for.
 */
export async function scrollGridToTop(page: Page): Promise<void> {
  await page.evaluate(() => {
    const header = document.querySelector('[data-testid="week-day-header"]')
    let el = header?.parentElement ?? null
    while (el && getComputedStyle(el).overflowY !== 'auto') el = el.parentElement
    if (el) {
      el.style.scrollBehavior = 'auto'
      el.scrollTop = 0
    }
  })
  // One frame so layout and the sticky headers settle before anyone measures.
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => r(null))))
}
