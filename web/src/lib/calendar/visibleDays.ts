/**
 * How many day columns the week grid draws: 1 on a phone, 3 on a small
 * tablet, 7 otherwise.
 *
 * 🔴 Asked of a media query, not of window.innerWidth. On a phone innerWidth
 * is the visual viewport, and it widens whenever content overflows the page
 * and the browser zooms out to fit it. A 375px phone read 457 on this machine
 * and above 480 on CI (26.09, run 36210478748), so the grid drew three 83px
 * columns and the header ran off the screen. Media queries answer the layout
 * width, which overflow does not change, and they are what Tailwind's
 * breakpoints use, so the grid and the rest of the page agree.
 */
export const MOBILE_BREAKPOINT = 480
export const TABLET_BREAKPOINT = 768

export const DAY_COUNT_QUERIES = [
  `(max-width: ${MOBILE_BREAKPOINT - 1}px)`,
  `(max-width: ${TABLET_BREAKPOINT - 1}px)`,
] as const

export interface ViewportLike {
  innerWidth: number
  matchMedia?: (query: string) => { matches: boolean }
}

export function visibleDaysFor(win: ViewportLike): 1 | 3 | 7 {
  if (typeof win.matchMedia !== 'function') {
    const w = win.innerWidth
    return w < MOBILE_BREAKPOINT ? 1 : w < TABLET_BREAKPOINT ? 3 : 7
  }
  const [phone, tablet] = DAY_COUNT_QUERIES
  if (win.matchMedia(phone).matches) return 1
  if (win.matchMedia(tablet).matches) return 3
  return 7
}
