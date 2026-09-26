/**
 * Dragging an event to another day of the month, as a plain state machine
 * (spec V003-20260924-arc-web-month-view, R8). useMonthDrag wires it to
 * window pointer events; the rules live here so they can be tested.
 *
 * - A press that never moves `threshold` px stays a click.
 * - A drop on another day moves the event; on its own day or on no day, nothing.
 * - pointercancel ends the drag without moving: otherwise the next ordinary
 *   click anywhere would finish it and move the event there.
 * - After a drop, `wasDrag()` is true once, for the click the browser fires
 *   right after pointerup; `defer` clears it after that.
 */

export interface MonthDragOptions<E> {
  threshold: number
  dayAt: (x: number, y: number) => string | null
  onMove: (event: E, fromDay: string, toDay: string) => void
  onOver: (day: string | null) => void
  onDragging: (event: E | null) => void
  defer: (fn: () => void) => void
}

export function createMonthDrag<E>(o: MonthDragOptions<E>) {
  let pending: { event: E; fromDay: string; x: number; y: number; moved: boolean } | null = null
  let justDropped = false

  const reset = () => {
    const wasMoving = pending?.moved
    pending = null
    if (wasMoving) {
      o.onOver(null)
      o.onDragging(null)
    }
  }

  return {
    down(event: E, fromDay: string, x: number, y: number) {
      pending = { event, fromDay, x, y, moved: false }
    },
    move(x: number, y: number) {
      const p = pending
      if (!p) return
      if (!p.moved && Math.hypot(x - p.x, y - p.y) < o.threshold) return
      if (!p.moved) {
        p.moved = true
        o.onDragging(p.event)
      }
      o.onOver(o.dayAt(x, y))
    },
    up(x: number, y: number) {
      const p = pending
      if (!p) return
      reset()
      if (!p.moved) return
      justDropped = true
      o.defer(() => {
        justDropped = false
      })
      const to = o.dayAt(x, y)
      if (to && to !== p.fromDay) o.onMove(p.event, p.fromDay, to)
    },
    cancel() {
      reset()
    },
    /** True once, right after a drop. */
    wasDrag() {
      const was = justDropped
      justDropped = false
      return was
    },
  }
}
