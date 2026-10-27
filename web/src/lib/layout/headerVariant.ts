export type HeaderVariant = 'horizontal' | 'vertical'

/** The phone breakpoint the layout uses: the sidebar is `hidden md:flex`. */
export const PHONE_QUERY = '(max-width: 767px)'

/**
 * The header actually drawn. The vertical sidebar is hidden below md, so on a
 * phone "vertical" meant no top bar at all: no avatar menu, no help. The
 * setting syncs from the account, so choosing the sidebar on a desktop took
 * the phone's top bar away (mobile tour 25.09, MW5). A phone always gets the
 * top bar; the saved choice stays for the desktop.
 */
/**
 * 'none' inside Telegram on a phone: Telegram draws its own header, and ours
 * under it made two (friction audit 26.10 #3, Denis 27.10 «hide, profile in
 * More»). Header and Layout's padding both ask here, so they cannot disagree.
 */
export function effectiveHeaderVariant(
  saved: string | null | undefined,
  isPhone: boolean,
  inTelegram = false,
): HeaderVariant | 'none' {
  if (isPhone && inTelegram) return 'none'
  if (isPhone) return 'horizontal'
  return saved === 'vertical' ? 'vertical' : 'horizontal'
}
