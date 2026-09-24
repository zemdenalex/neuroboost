/**
 * Escape in the event editor (Denis, 24.09: close, but if something was typed
 * «press escape again to …»).
 *
 * - Another modal on top (the "only this / all" dialog) owns Escape: ignore.
 * - Nothing changed: close.
 * - Changed: the first Escape arms (the editor shows the hint), the second closes.
 *   Typing again disarms, so an old press cannot count as the first of two.
 */
export type EscapeAction = 'ignore' | 'close' | 'arm'

export function escapeStep(opts: { dirty: boolean; armed: boolean; otherModalOpen: boolean }): EscapeAction {
  if (opts.otherModalOpen) return 'ignore'
  if (!opts.dirty || opts.armed) return 'close'
  return 'arm'
}
