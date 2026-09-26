/**
 * Click and double click on a month day (spec V003-20260924-arc-web-month-view, R6).
 *
 * A click opens the day's week, a double click creates an event there. The
 * browser reports the first click of a double click as a plain click, so the
 * single action waits CLICK_WAIT_MS: without the wait, a double click would
 * leave the month on its first click and create nothing.
 */

/** Denis, 24.09: 300 ms, and adjustable in settings (month_click_wait_ms). */
export const CLICK_WAIT_MS = 300
export const CLICK_WAIT_MIN = 150
export const CLICK_WAIT_MAX = 800

/** The wait from settings, clamped; 300 when unset or not a number. */
export function readClickWait(settings: { month_click_wait_ms?: unknown } | undefined): number {
  const v = settings?.month_click_wait_ms
  if (typeof v !== 'number' || !Number.isFinite(v)) return CLICK_WAIT_MS
  return Math.min(CLICK_WAIT_MAX, Math.max(CLICK_WAIT_MIN, Math.round(v)))
}

export interface DayClick {
  /** `detail` is MouseEvent.detail: 1 for a click, 2 for the second click of a double click. */
  (day: string, detail: number): void
  cancel(): void
}

export function createDayClick(
  onOpen: (day: string) => void,
  onCreate: (day: string) => void,
  waitMs: number = CLICK_WAIT_MS,
): DayClick {
  let timer: ReturnType<typeof setTimeout> | null = null

  const cancel = () => {
    if (timer !== null) clearTimeout(timer)
    timer = null
  }

  const click = ((day: string, detail: number) => {
    cancel()
    if (detail >= 2) {
      onCreate(day)
      return
    }
    timer = setTimeout(() => {
      timer = null
      onOpen(day)
    }, waitMs)
  }) as DayClick
  click.cancel = cancel
  return click
}

export interface CellClickActions {
  /** The shared click / double click (open week / create). */
  click: DayClick
  /** Variant D: its own click / double click (choose / create). */
  splitClick: DayClick
  /** Variant D: choose the day at once, before the double-click wait. */
  choose: (day: string) => void
}

/**
 * A click on a month cell. The click the browser fires right after a drop is
 * not a click. In variant D a click chooses the day at once (spec R10); a
 * second click still creates.
 */
export function routeCellClick(split: boolean, day: string, detail: number, wasDrag: boolean, a: CellClickActions): void {
  if (wasDrag) return
  if (split) {
    if (detail < 2) a.choose(day)
    a.splitClick(day, detail)
    return
  }
  a.click(day, detail)
}
