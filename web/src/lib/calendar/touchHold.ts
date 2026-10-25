/**
 * Hold or scroll — the one decision a finger on an event has to make.
 *
 * A finger on a phone calendar means two different things: "scroll the day" and
 * "pick this event up". Apple and Google settle it the same way: holding still
 * picks the event up, moving first scrolls (docs/team/architecture/
 * V003-20261025-arc-perenos-na-telefone.md). The browser settles it too — as
 * soon as the finger travels it starts scrolling and stops sending us anything
 * we can cancel — so the hold has to be measured here, before that happens.
 *
 * Pure of the DOM: the caller feeds it touchstart / touchmove / touchend and
 * gets one callback when the hold is complete. That keeps the timing rules
 * testable with fake timers; the part that cannot be tested here (does the
 * browser really leave the scroll alone once we lift) is `e2e/phone-drag.spec.ts`.
 */

export interface HoldTracker {
  /** A finger went down at (x, y). */
  start(x: number, y: number): void
  /** The finger moved. Before the lift, travelling past the slop cancels the hold. */
  move(x: number, y: number): void
  /** The finger came up or the browser took the gesture (touchcancel). */
  end(): void
  /** The hold completed and the event is "in the hand". */
  lifted(): boolean
  /** A hold is running or done: true from touchstart until end(). */
  active(): boolean
}

interface Options {
  holdMs: number
  /** How far the finger may drift and still count as holding still. */
  slopPx: number
  /** Called once, at the finger's current position, when the hold completes. */
  onLift: (x: number, y: number) => void
}

export function createHoldTracker({ holdMs, slopPx, onLift }: Options): HoldTracker {
  let timer: ReturnType<typeof setTimeout> | null = null
  let originX = 0
  let originY = 0
  let curX = 0
  let curY = 0
  let isActive = false
  let isLifted = false

  const clear = () => {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  return {
    start(x, y) {
      clear()
      originX = curX = x
      originY = curY = y
      isActive = true
      isLifted = false
      timer = setTimeout(() => {
        timer = null
        isLifted = true
        onLift(curX, curY)
      }, holdMs)
    },
    move(x, y) {
      curX = x
      curY = y
      if (isLifted || timer === null) return
      if (Math.hypot(x - originX, y - originY) > slopPx) {
        // A scroll. Stop timing; the page keeps the gesture.
        clear()
      }
    },
    end() {
      clear()
      isActive = false
      isLifted = false
    },
    lifted: () => isLifted,
    active: () => isActive,
  }
}
