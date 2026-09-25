/**
 * Light or dark (Denis 25.09, variants page choice 3: «follow Telegram's
 * theme»). Inside the Mini App the app takes Telegram's scheme; outside it
 * stays dark, as the web always was, until there is a choice of its own
 * (docs/tasks-light-theme.md, LT7).
 *
 * The colours themselves are CSS variables behind Tailwind's zinc / white /
 * black (tailwind.config.js, index.css): the dark values are the old hex
 * codes, the light ones the same scale turned over.
 */

export type Theme = 'light' | 'dark'

/** Light or dark from a #rgb / #rrggbb background, by relative luminance. */
export function schemeFromBg(bg: string | undefined): Theme | null {
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(bg ?? '')
  if (!m) return null
  const hex = m[1].length === 3 ? m[1].replace(/./g, (c) => c + c) : m[1]
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
  const luminance = 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
  return luminance > 0.4 ? 'light' : 'dark'
}

/**
 * Telegram passes its theme in the launch hash (tgWebAppThemeParams), so the
 * scheme is known before the first frame, without waiting for its script.
 */
export function schemeFromHash(hash: string): Theme | null {
  if (!hash.startsWith('#')) return null
  const raw = new URLSearchParams(hash.slice(1)).get('tgWebAppThemeParams')
  if (!raw) return null
  try {
    const params = JSON.parse(raw) as { bg_color?: unknown }
    return typeof params.bg_color === 'string' ? schemeFromBg(params.bg_color) : null
  } catch {
    return null
  }
}

export function resolveTheme(s: { telegram: Theme | null }): Theme {
  return s.telegram ?? 'dark'
}

/** Puts the theme on <html>: data-theme for the CSS variables, color-scheme for native controls. */
export function applyTheme(theme: Theme, root: HTMLElement = document.documentElement): void {
  root.dataset.theme = theme
  root.style.colorScheme = theme
}
