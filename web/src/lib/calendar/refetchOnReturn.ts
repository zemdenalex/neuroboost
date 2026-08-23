/**
 * Should coming back to the tab trigger a reload?
 *
 * Denis, 23.08: changes another person makes in a shared calendar are only
 * visible after a page reload. His decision, same day: re-read when the tab
 * becomes visible again — no realtime.
 *
 * ⚠ The threshold is not decoration. Alt-tabbing between two windows fires
 * `visibilitychange` constantly, and an unconditional refetch would put the
 * calendar's whole week on every window switch — busier than polling, and
 * invisible until someone reads the access log.
 */

/** How stale the data has to be before a return is worth a request. */
export const STALE_AFTER_MS = 30_000

export function shouldRefetchOnReturn(
  visibility: DocumentVisibilityState,
  lastLoadedAt: number,
  now: number,
): boolean {
  if (visibility !== 'visible') return false
  // A clock that jumped backwards (a resumed laptop, an NTP correction) would
  // otherwise produce a negative age and suppress the reload for as long as the
  // skew lasts — exactly when the data is most likely to be stale.
  const age = now - lastLoadedAt
  if (age < 0) return true
  return age >= STALE_AFTER_MS
}
