import { describe, it, expect, vi } from 'vitest'
import { backButtonVisible, initDataFromHash, pickStartupAuth, prepareWebApp, type TgWebApp } from './webApp'

// Telegram opens a Mini App with its launch data in the hash:
// #tgWebAppData=<url-encoded initData>&tgWebAppVersion=8.0&tgWebAppPlatform=ios
const initData = 'auth_date=1790000000&hash=abc&user=%7B%22id%22%3A1%7D'
const hash = `#tgWebAppData=${encodeURIComponent(initData)}&tgWebAppVersion=8.0&tgWebAppPlatform=ios`

describe('initDataFromHash', () => {
  it('returns the initData string exactly as Telegram signed it', () => {
    expect(initDataFromHash(hash)).toBe(initData)
  })
  it('is null outside Telegram', () => {
    expect(initDataFromHash('')).toBeNull()
    expect(initDataFromHash('#section-2')).toBeNull()
    expect(initDataFromHash('#tgWebAppData=')).toBeNull()
  })
})

describe('pickStartupAuth', () => {
  it('inside Telegram signs in with initData even over a stored session', () => {
    // The WebView may hold a session of someone else who used this device's
    // browser storage; the Telegram identity is the one that is sure.
    expect(pickStartupAuth({ initData, storedTokenValid: true })).toBe('webapp')
  })
  it('outside Telegram keeps the stored session, or none', () => {
    expect(pickStartupAuth({ initData: null, storedTokenValid: true })).toBe('stored')
    expect(pickStartupAuth({ initData: null, storedTokenValid: false })).toBe('none')
  })
})

function fakeWebApp(version: string): TgWebApp & { calls: string[] } {
  const calls: string[] = []
  return {
    calls,
    version,
    initData,
    ready: () => calls.push('ready'),
    expand: () => calls.push('expand'),
    disableVerticalSwipes: () => calls.push('disableVerticalSwipes'),
    isVersionAtLeast: (v: string) => Number(version) >= Number(v),
    BackButton: { show: vi.fn(), hide: vi.fn(), onClick: vi.fn(), offClick: vi.fn() },
  }
}

describe('prepareWebApp', () => {
  it('turns off vertical swipes so dragging an event does not close the app', () => {
    const wa = fakeWebApp('8.0')
    prepareWebApp(wa)
    expect(wa.calls).toEqual(['ready', 'expand', 'disableVerticalSwipes'])
  })
  it('skips what an old client does not have instead of throwing', () => {
    const wa = fakeWebApp('7.0')
    expect(() => prepareWebApp(wa)).not.toThrow()
    expect(wa.calls).toEqual(['ready', 'expand'])
  })
})

describe('backButtonVisible', () => {
  it('is hidden on the bottom-bar tabs, where back would leave the app', () => {
    for (const path of ['/', '/home', '/calendar', '/agenda', '/tasks', '/calendar/']) {
      expect(backButtonVisible(path), path).toBe(false)
    }
  })
  it('shows on every inner page', () => {
    for (const path of ['/settings', '/tools/kanban', '/day-tasks', '/profile']) {
      expect(backButtonVisible(path), path).toBe(true)
    }
  })
})
