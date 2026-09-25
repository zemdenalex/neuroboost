import { test } from './fixtures/auth'
import * as fs from 'node:fs'

/**
 * A measurement, not a test (docs/agents/queue.md «Производительность»): LCP and
 * CLS of the logged-in pages, which DevTools cannot reach without a token.
 * Runs only with NB_PERF=1; writes NB_PERF_OUT (JSON) if set. Nothing asserted.
 *
 *   NB_PERF=1 NB_PERF_OUT=perf.json web/scripts/e2e-local.sh --staging --project mobile e2e/perf.spec.ts
 *
 * Throttling like Lighthouse's mobile preset: CPU ×4, ~Fast 4G. Cold: a new
 * page per route, cache disabled.
 */

const ROUTES = ['/calendar', '/day-tasks', '/tasks']

test.skip(!process.env.NB_PERF, 'measurement, run with NB_PERF=1')

test('LCP and CLS of logged-in pages', async ({ authedPage }) => {
  test.setTimeout(180_000)
  const results: Record<string, { lcp: number; cls: number; shifts: string[] }> = {}
  const cdp = await authedPage.context().newCDPSession(authedPage)
  await cdp.send('Network.enable')
  await cdp.send('Network.setCacheDisabled', { cacheDisabled: true })
  await cdp.send('Network.emulateNetworkConditions', {
    offline: false,
    latency: 150,
    downloadThroughput: (9 * 1024 * 1024) / 8,
    uploadThroughput: (1.5 * 1024 * 1024) / 8,
  })
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 })

  for (const route of ROUTES) {
    await authedPage.goto(route, { waitUntil: 'load' })
    await authedPage.waitForTimeout(4000)
    results[route] = await authedPage.evaluate(
      () =>
        new Promise<{ lcp: number; cls: number; shifts: string[] }>((resolve) => {
          let lcp = 0
          let cls = 0
          const shifts: string[] = []
          new PerformanceObserver((l) => {
            for (const e of l.getEntries()) lcp = Math.max(lcp, e.startTime)
          }).observe({ type: 'largest-contentful-paint', buffered: true })
          new PerformanceObserver((l) => {
            type Shift = PerformanceEntry & { value: number; hadRecentInput: boolean; sources?: Array<{ node?: Node | null }> }
            for (const e of l.getEntries() as Shift[]) {
              if (e.hadRecentInput) continue
              cls += e.value
              // What moved, so a bad number points at an element.
              const who = (e.sources ?? []).map((src) => {
                const el = src.node as HTMLElement | null | undefined
                return el ? `${el.nodeName.toLowerCase()}${el.dataset?.testid ? '[' + el.dataset.testid + ']' : ''}.${String(el.className).slice(0, 60)}` : '?'
              })
              shifts.push(`${Math.round(e.value * 1000) / 1000} @${Math.round(e.startTime)}ms ${who.join(' | ')}`)
            }
          }).observe({ type: 'layout-shift', buffered: true })
          setTimeout(() => resolve({ lcp: Math.round(lcp), cls: Math.round(cls * 1000) / 1000, shifts }), 500)
        }),
    )
  }
  console.log(JSON.stringify(results))
  if (process.env.NB_PERF_OUT) fs.writeFileSync(process.env.NB_PERF_OUT, JSON.stringify(results, null, 2))
})
