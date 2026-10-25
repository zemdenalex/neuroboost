import type { TgWebApp } from './webApp'

/**
 * Full screen inside Telegram (Bot API 8.0, 17.11.2024: requestFullscreen /
 * exitFullscreen, the fullscreenChanged and fullscreenFailed events). Not every
 * client has it (a failure answers UNSUPPORTED), so it is a per-device choice,
 * kept in localStorage and not in the account settings: PATCH /auth/me replaces
 * the whole settings blob (CLAUDE.md gotcha 21), and a phone and a computer
 * can reasonably differ.
 *
 * In full screen Telegram draws its buttons over the top of the page; the
 * insets it reports come out as CSS variables (see index.css, `.tg-fullscreen`).
 */

export const FULLSCREEN_KEY = 'nb.tg.fullscreen'
const FULLSCREEN_CLASS = 'tg-fullscreen'

type Store = Pick<Storage, 'getItem' | 'setItem'>

/** Whether this Telegram client can be asked for full screen at all. */
export function fullscreenSupported(wa: TgWebApp): boolean {
  return wa.isVersionAtLeast('8.0') && typeof wa.requestFullscreen === 'function'
}

export function readFullscreenPref(storage: Store): boolean {
  try {
    return storage.getItem(FULLSCREEN_KEY) === '1'
  } catch {
    return false
  }
}

export function storeFullscreenPref(storage: Store, on: boolean): void {
  try {
    storage.setItem(FULLSCREEN_KEY, on ? '1' : '0')
  } catch {
    // Storage blocked: the choice lasts until the Mini App closes.
  }
}

/** Asks Telegram for full screen or for the normal window; a refusal is not an error here. */
export function setFullscreen(wa: TgWebApp, on: boolean): void {
  if (!fullscreenSupported(wa)) return
  try {
    if (on && !wa.isFullscreen) wa.requestFullscreen?.()
    if (!on && wa.isFullscreen) wa.exitFullscreen?.()
  } catch {
    // The client threw (older build): the page stays as it is.
  }
}

/** At launch: the person's saved choice. Nothing is requested unless they made it. */
export function applyFullscreenAtLaunch(wa: TgWebApp, storage: Store): void {
  if (readFullscreenPref(storage)) setFullscreen(wa, true)
}

/**
 * Keeps `tg-fullscreen` on the root element equal to Telegram's own state, so
 * the page pads for Telegram's overlay buttons only while they are there.
 */
export function watchFullscreen(wa: TgWebApp, root: { classList: { toggle: (c: string, on: boolean) => unknown } }): void {
  const sync = () => root.classList.toggle(FULLSCREEN_CLASS, Boolean(wa.isFullscreen))
  sync()
  wa.onEvent?.('fullscreenChanged', sync)
}
