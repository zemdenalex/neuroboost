/**
 * Telegram Mini App glue: the same web app, opened by a bot button.
 *
 * Sign-in does not depend on Telegram's script. Telegram puts the signed
 * launch data in the URL hash (#tgWebAppData=…), so the app reads it there
 * and posts it to /api/auth/telegram-webapp. The script (for ready(), swipes,
 * BackButton) is loaded only inside Telegram: telegram.org can be blocked for
 * ordinary visitors, and a script tag in <head> for everyone would stall the
 * whole site on it.
 *
 * Research: E:/Projects/100 - Research/prochee/telegram-mini-app-initdata-2026-09-24.md
 */

export interface TgBackButton {
  show: () => void
  hide: () => void
  onClick: (cb: () => void) => void
  offClick: (cb: () => void) => void
}

/** The part of window.Telegram.WebApp this app uses. */
export interface TgWebApp {
  version: string
  initData: string
  ready: () => void
  expand: () => void
  disableVerticalSwipes?: () => void
  isVersionAtLeast: (version: string) => boolean
  BackButton: TgBackButton
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TgWebApp }
  }
}

const SDK_URL = 'https://telegram.org/js/telegram-web-app.js'

/** The initData string from a Mini App launch hash, or null outside Telegram. */
export function initDataFromHash(hash: string): string | null {
  if (!hash.startsWith('#')) return null
  const value = new URLSearchParams(hash.slice(1)).get('tgWebAppData')
  return value ? value : null
}

/** The startapp parameter of a t.me/<bot>/<app>?startapp=… launch, or null. */
export function startParamFromHash(hash: string): string | null {
  if (!hash.startsWith('#')) return null
  return new URLSearchParams(hash.slice(1)).get('tgWebAppStartParam') || null
}

/**
 * Where a start link lands. Telegram allows only [A-Za-z0-9_-] (its length
 * limit is not confirmed, see the research file), so the forms are short:
 * `t-<task uuid>` opens the task, `dt` the day tasks, `d-2026-09-25` the
 * calendar on that day.
 * Anything unknown is ignored (the app opens as usual) rather than guessed.
 */
export function startAppRoute(param: string | null): string | null {
  if (!param) return null
  if (param === 'dt') return '/day-tasks'
  const day = /^d-(\d{4}-\d{2}-\d{2})$/.exec(param)
  if (day) return `/calendar?date=${day[1]}`
  const task = /^t-([0-9a-f-]{36})$/i.exec(param)
  if (task) return `/tasks?task=${task[1]}`
  return null
}

export type StartupAuth = 'webapp' | 'stored' | 'none'

/** Inside Telegram the signed identity wins over whatever session storage holds. */
export function pickStartupAuth(s: { initData: string | null; storedTokenValid: boolean }): StartupAuth {
  if (s.initData) return 'webapp'
  return s.storedTokenValid ? 'stored' : 'none'
}

/**
 * ready() + expand(), and disableVerticalSwipes() where the client has it
 * (Bot API 7.7+): without it a vertical drag of a calendar event is read as
 * "swipe down to close" and the Mini App folds away mid-drag.
 */
export function prepareWebApp(wa: TgWebApp): void {
  wa.ready()
  wa.expand()
  if (wa.isVersionAtLeast('7.7') && wa.disableVerticalSwipes) wa.disableVerticalSwipes()
}

// The bottom-bar tabs: there Telegram's own close button is the way out, and
// a back button would step through tab history instead.
const ROOT_PATHS = new Set(['/', '/home', '/calendar', '/agenda', '/tasks'])

/** Telegram's BackButton shows on inner pages only. */
export function backButtonVisible(pathname: string): boolean {
  const path = pathname.length > 1 ? pathname.replace(/\/+$/, '') : pathname
  return !ROOT_PATHS.has(path)
}

// Read once at startup: client-side navigation drops the hash, and Telegram's
// script reads the same hash when it loads, so nothing here rewrites it.
const launchInitData = typeof window === 'undefined' ? null : initDataFromHash(window.location.hash)

export function launchedInTelegram(): string | null {
  return launchInitData
}

const launchStartParam = typeof window === 'undefined' ? null : startParamFromHash(window.location.hash)

/** The page a start link asked for, once per launch; null afterwards. */
let startRouteTaken = false
export function takeStartRoute(): string | null {
  if (startRouteTaken || !launchInitData) return null
  startRouteTaken = true
  return startAppRoute(launchStartParam)
}

let sdk: Promise<TgWebApp | null> | null = null

/** Loads Telegram's script once, inside Telegram only; null elsewhere or on failure. */
export function loadWebApp(): Promise<TgWebApp | null> {
  if (!launchInitData) return Promise.resolve(null)
  if (sdk) return sdk
  sdk = new Promise((resolve) => {
    if (window.Telegram?.WebApp) return resolve(window.Telegram.WebApp)
    const s = document.createElement('script')
    s.src = SDK_URL
    s.async = true
    s.onload = () => resolve(window.Telegram?.WebApp ?? null)
    s.onerror = () => resolve(null)
    document.head.appendChild(s)
  })
  return sdk
}
