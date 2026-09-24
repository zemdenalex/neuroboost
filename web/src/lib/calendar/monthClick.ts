/**
 * Click and double click on a month day (spec V003-20260924-arc-web-month-view, R6).
 *
 * A click opens the day's week, a double click creates an event there. The
 * browser reports the first click of a double click as a plain click, so the
 * single action waits CLICK_WAIT_MS: without the wait, a double click would
 * leave the month on its first click and create nothing.
 */

export const CLICK_WAIT_MS = 250

export interface DayClick {
  /** `detail` is MouseEvent.detail: 1 for a click, 2 for the second click of a double click. */
  (day: string, detail: number): void
  cancel(): void
}

export function createDayClick(onOpen: (day: string) => void, onCreate: (day: string) => void): DayClick {
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
    }, CLICK_WAIT_MS)
  }) as DayClick
  click.cancel = cancel
  return click
}
