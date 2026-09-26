/**
 * The person's Telegram colours inside the Mini App (MA3b, Denis 26.09:
 * variant 2 of https://claude.ai/artifact/GYZJSUPtrGaR6T5XNoVcyu, «everything
 * from Telegram»): page, panels, text, hint, buttons and links take
 * themeParams. Calendar colours are the user's data and are not touched.
 *
 * The app already paints through CSS variables (index.css, tailwind.config.js),
 * so the palette is a set of those variables put inline on <html>, over the
 * light or dark defaults. A custom Telegram theme can be unreadable, so two
 * floors: text on the page needs 4.5:1 or the whole palette is skipped, and
 * white on the button needs 2:1 or the buttons keep our blue. Not 3:1: Telegram's
 * own night theme puts white on #50a8eb (2.6:1), and a floor that refuses
 * Telegram's stock theme would refuse the case this exists for.
 */

export interface TelegramThemeParams {
  bg_color?: string
  secondary_bg_color?: string
  text_color?: string
  hint_color?: string
  link_color?: string
  button_color?: string
  button_text_color?: string
}

type Rgb = [number, number, number]

function parseHex(hex: unknown): Rgb | null {
  if (typeof hex !== 'string') return null
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex)
  if (!m) return null
  const h = m[1].length === 3 ? m[1].replace(/./g, (c) => c + c) : m[1]
  return [0, 2, 4].map((i) => parseInt(h.slice(i, i + 2), 16)) as Rgb
}

const triple = (c: Rgb) => c.map((n) => Math.round(n)).join(' ')
const toHex = (c: Rgb) => '#' + c.map((n) => Math.round(n).toString(16).padStart(2, '0')).join('')
/** `from` moved toward `to` by t (0 = from, 1 = to). */
const mix = (from: Rgb, to: Rgb, t: number): Rgb => from.map((v, i) => v + (to[i] - v) * t) as Rgb

function luminance([r, g, b]: Rgb): number {
  const lin = (c: number) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
}

/** WCAG contrast ratio of two hex colours (1 to 21); 1 when either is not hex. */
export function contrastRatio(a: string, b: string): number {
  const ca = parseHex(a)
  const cb = parseHex(b)
  if (!ca || !cb) return 1
  const [hi, lo] = [luminance(ca), luminance(cb)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

// Where each step of the grey scale sits between the page (0) and the text (1),
// read off Tailwind's zinc on a zinc-950 page. 900 and 500 come from Telegram
// itself when it sends them (panels, hint).
const SCALE: Array<[string, number]> = [
  ['800', 0.1], ['700', 0.18], ['600', 0.32], ['400', 0.58], ['300', 0.72], ['200', 0.84], ['100', 0.92], ['50', 0.96],
]

/** CSS variables for the palette, or null when it is missing or unreadable. */
export function telegramPaletteVars(p: TelegramThemeParams | null | undefined): Record<string, string> | null {
  const bg = parseHex(p?.bg_color)
  const text = parseHex(p?.text_color)
  if (!bg || !text || contrastRatio(p!.bg_color!, p!.text_color!) < 4.5) return null

  const vars: Record<string, string> = {
    '--nb-zinc-950': triple(bg),
    '--nb-zinc-900': triple(parseHex(p!.secondary_bg_color) ?? mix(bg, text, 0.05)),
    '--nb-zinc-500': triple(parseHex(p!.hint_color) ?? mix(bg, text, 0.45)),
    '--nb-white': triple(text),
    '--nb-black': triple(bg),
  }
  for (const [step, t] of SCALE) vars[`--nb-zinc-${step}`] = triple(mix(bg, text, t))

  const button = parseHex(p!.button_color)
  if (button && contrastRatio(p!.button_color!, '#ffffff') >= 2) {
    vars['--nb-blue-500'] = triple(button)
    vars['--nb-blue-600'] = triple(button)
    vars['--nb-blue-700'] = triple(mix(button, [0, 0, 0], 0.15))
  }
  const link = parseHex(p!.link_color) ?? (vars['--nb-blue-600'] ? button : null)
  if (link) {
    vars['--nb-blue-400'] = triple(link)
    vars['--nb-blue-300'] = triple(mix(link, text, 0.35))
  }
  return vars
}

/** Telegram's own header and background in the page's colours, or null. */
export function telegramChromeFrom(p: TelegramThemeParams | null | undefined): { header: string; background: string } | null {
  const bg = parseHex(p?.bg_color)
  if (!bg || !telegramPaletteVars(p)) return null
  return { header: toHex(parseHex(p!.secondary_bg_color) ?? bg), background: toHex(bg) }
}

/** themeParams from the launch hash, so the palette is on before Telegram's script loads. */
export function paletteFromHash(hash: string): TelegramThemeParams | null {
  if (!hash.startsWith('#')) return null
  const raw = new URLSearchParams(hash.slice(1)).get('tgWebAppThemeParams')
  if (!raw) return null
  try {
    const v: unknown = JSON.parse(raw)
    return v && typeof v === 'object' ? (v as TelegramThemeParams) : null
  } catch {
    return null
  }
}

const APPLIED = new Set<string>()

/** Puts the palette on <html>, or takes a previous one off when there is none. */
export function applyTelegramPalette(p: TelegramThemeParams | null | undefined, root: HTMLElement = document.documentElement): boolean {
  const vars = telegramPaletteVars(p)
  for (const name of APPLIED) if (!vars || !(name in vars)) root.style.removeProperty(name)
  APPLIED.clear()
  if (!vars) return false
  for (const [name, value] of Object.entries(vars)) {
    root.style.setProperty(name, value)
    APPLIED.add(name)
  }
  return true
}
