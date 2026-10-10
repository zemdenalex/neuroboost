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

export const THEME_CHOICES = ['dark', 'light', 'system'] as const
export type ThemeChoice = (typeof THEME_CHOICES)[number]

/** The web's own choice (⚙️, account setting `theme`); dark when unset or unknown. */
export function readThemeChoice(settings: { theme?: unknown } | undefined): ThemeChoice {
  const v = settings?.theme
  return typeof v === 'string' && (THEME_CHOICES as readonly string[]).includes(v) ? (v as ThemeChoice) : 'dark'
}

/**
 * Inside Telegram its scheme wins (Denis 25.09, choice B). Outside it, the
 * choice in settings: dark, light, or the device's (Denis 25.09: «dark light
 * system is fine»). No choice means dark, as the web has always been.
 */
export function resolveTheme(s: { telegram: Theme | null; choice?: ThemeChoice; systemDark?: boolean }): Theme {
  if (s.telegram) return s.telegram
  if (s.choice === 'light') return 'light'
  if (s.choice === 'system') return s.systemDark === false ? 'light' : 'dark'
  return 'dark'
}

const CHOICE_KEY = 'nb-theme'

/** The choice kept on this device too, so it applies before the first frame. */
export function storedThemeChoice(): ThemeChoice {
  try {
    return readThemeChoice({ theme: localStorage.getItem(CHOICE_KEY) })
  } catch {
    return 'dark'
  }
}

export function storeThemeChoice(choice: ThemeChoice): void {
  try {
    localStorage.setItem(CHOICE_KEY, choice)
  } catch {
    // Only the pre-first-frame hint is lost; the account keeps the choice.
  }
}

/** Resolves and applies the theme for this page right now. */
export function applyCurrentTheme(choice: ThemeChoice = storedThemeChoice()): Theme {
  const theme = resolveTheme({
    telegram: schemeFromHash(window.location.hash),
    choice,
    systemDark: window.matchMedia?.('(prefers-color-scheme: dark)').matches,
  })
  applyTheme(theme)
  return theme
}

/** Puts the theme on <html>: data-theme for the CSS variables, color-scheme for native controls. */
export function applyTheme(theme: Theme, root: HTMLElement = document.documentElement): void {
  root.dataset.theme = theme
  root.style.colorScheme = theme
}

/**
 * Telegram's own frame (header bar, the background behind the page) in the
 * app's colours, so the Mini App reads as one surface (LT6): the page is
 * zinc-950 and the app's header zinc-900 of the current theme.
 */
export function telegramChrome(theme: Theme): { header: string; background: string } {
  return theme === 'light' ? { header: '#f4f4f5', background: '#fafafa' } : { header: '#18181b', background: '#09090b' }
}
