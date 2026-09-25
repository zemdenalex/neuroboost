/**
 * The hour the time grid opens on. Until 25.09 it always opened at 00:00, so
 * on a phone the first screen was the empty small hours and "00:00" sat half
 * under the sticky day header (mobile tour, MW11).
 *
 * Today on screen: one hour before now, so the current hour and a little
 * context are visible. Another period: the start of the working day.
 */
export function initialScrollHour(opts: { nowHour: number; todayVisible: boolean; workStart?: string }): number {
  if (opts.todayVisible) return Math.max(0, Math.min(23, opts.nowHour) - 1)
  const m = /^(\d{1,2}):\d{2}$/.exec(opts.workStart ?? '')
  const h = m ? Number(m[1]) : 8
  return h >= 0 && h <= 23 ? h : 8
}
