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
export function effectiveHeaderVariant(saved: string | null | undefined, isPhone: boolean): HeaderVariant {
  if (isPhone) return 'horizontal'
  return saved === 'vertical' ? 'vertical' : 'horizontal'
}
