import type { TelegramThemeParams } from '../theme/telegramPalette'

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
  /** Telegram's current scheme; follows the person's theme switch. */
  colorScheme?: 'light' | 'dark'
  /** The person's Telegram colours (MA3b); follows the theme switch too. */
  themeParams?: TelegramThemeParams
  onEvent?: (event: 'themeChanged', cb: () => void) => void
  setHeaderColor?: (color: string) => void
  setBackgroundColor?: (color: string) => void
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

/**
 * Inside the Mini App a stored session is kept only if it belongs to the person
 * who launched it (review I2, 25.09). A failed exchange used to fall back to
 * whatever token the WebView held, possibly someone else's, and the Mini App
 * hides Sign out, so there was no way to leave it. The user id is read from
 * the launch string; the server already checked, or will check, its signature.
 */
export function sessionFitsLaunch(initData: string | null, sessionTgId: number | undefined): boolean {
  if (!initData) return true
  try {
    const user = JSON.parse(new URLSearchParams(initData).get('user') ?? '') as { id?: unknown }
    return typeof user.id === 'number' && user.id === sessionTgId
  } catch {
    return false
  }
}

/**
 * Telegram's Back on a page opened first (a start link, or a redirect chain of
 * replaces) has no history to go back to: it goes to the calendar instead of
 * doing nothing (review I3). `idx` is React Router's index in history.state.
 */
export function backAction(historyIdx: number | undefined): 'back' | 'calendar' {
  return typeof historyIdx === 'number' && historyIdx > 0 ? 'back' : 'calendar'
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

/**
 * Telegram's script reads its launch parameters (version, platform, data) from
 * the hash when it runs. It is loaded async, and by then a redirect (/ to
 * /home, a start link) may have replaced the URL without the hash: the script
 * then believes it is version 6.0 and disableVerticalSwipes and BackButton
 * quietly do nothing (review I1, 25.09). The script itself falls back to
 * sessionStorage["__telegram__initParams"], the way it survives its own
 * navigation, so the parameters are seeded there while the hash is intact.
 * ⚠ That key is the script's internal detail, not a documented API.
 */
function readJson(raw: string | null): Record<string, string> {
  try {
    return JSON.parse(raw ?? '{}') as Record<string, string>
  } catch {
    return {}
  }
}

export function seedSdkParams(
  storage: { getItem: (k: string) => string | null; setItem: (k: string, v: string) => void },
  hash: string,
): void {
  if (!initDataFromHash(hash)) return
  const stored = readJson(storage.getItem('__telegram__initParams'))
  const fresh = Object.fromEntries(new URLSearchParams(hash.slice(1)))
  storage.setItem('__telegram__initParams', JSON.stringify({ ...stored, ...fresh }))
}

// Read once at startup: client-side navigation drops the hash, and Telegram's
// script reads the same hash when it loads, so nothing here rewrites it.
const launchInitData = typeof window === 'undefined' ? null : initDataFromHash(window.location.hash)
if (launchInitData) {
  try {
    seedSdkParams(window.sessionStorage, window.location.hash)
  } catch {
    // Storage blocked: the script falls back to 6.0 behaviour, sign-in is unaffected.
  }
}

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
