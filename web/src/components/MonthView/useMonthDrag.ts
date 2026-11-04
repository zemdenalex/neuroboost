import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import type { NbEvent } from '../../types'
import { DRAG_THRESHOLD_PX } from '../Calendar/WeekGrid/weekgrid.constants'
import { createMonthDrag } from '../../lib/calendar/monthDrag'

/** The day cell under a screen point, read from its data-day attribute. */
function dayAt(x: number, y: number): string | null {
  const el = document.elementFromPoint(x, y)
  const cell = el instanceof Element ? el.closest('[data-day]') : null
  return cell?.getAttribute('data-day') ?? null
}

/**
 * Window pointer events wired to the month's drag machine (lib/calendar/monthDrag.ts).
 * `onPress` runs when a drag may start: the month cancels a pending
 * single click there, or its timer would switch to the week mid-drag.
 */
export function useMonthDrag(onMoveToDay: (event: NbEvent, fromDay: string, toDay: string) => void, onPress: () => void) {
  const [over, setOver] = useState<string | null>(null)
  const [draggingId, setDraggingId] = useState<string | null>(null)
  const moveRef = useRef(onMoveToDay)
  moveRef.current = onMoveToDay

  const [machine] = useState(() =>
    createMonthDrag<NbEvent>({
      threshold: DRAG_THRESHOLD_PX,
      dayAt,
      onMove: (event, from, to) => moveRef.current(event, from, to),
      onOver: setOver,
      onDragging: (event) => setDraggingId(event?.id ?? null),
      defer: (fn) => setTimeout(fn, 0),
    }),
  )

  const onItemPointerDown = useCallback(
    (event: NbEvent, e: ReactPointerEvent) => {
      if (e.button !== 0 || e.pointerType === 'touch') return
      const fromDay = (e.currentTarget as Element).closest('[data-day]')?.getAttribute('data-day')
      if (!fromDay) return
      onPress()
      machine.down(event, fromDay, e.clientX, e.clientY)
    },
    [machine, onPress],
  )

  // The finger (step 8 of docs/tasks-phone-drag.md). The mouse path above
  // returns on pointerType touch: a finger on a list row scrolls. A held finger
  // (TouchHoldBinder on DayList's row) picks the event up instead, and from then
  // on these document touch listeners steer the same machine the mouse does.
  // They are always attached, so a lift can never outrun them.
  const touchOwned = useRef(false)
  const lastTouch = useRef({ x: 0, y: 0 })
  const onItemTouchLift = useCallback(
    (event: NbEvent, fromDay: string, x: number, y: number) => {
      onPress()
      touchOwned.current = true
      lastTouch.current = { x, y }
      machine.down(event, fromDay, x, y)
    },
    [machine, onPress],
  )

  useEffect(() => {
    const tmove = (e: TouchEvent) => {
      if (!touchOwned.current) return
      const t = e.touches[0]
      if (!t || e.touches.length !== 1) return
      if (e.cancelable) e.preventDefault()
      lastTouch.current = { x: t.clientX, y: t.clientY }
      machine.move(t.clientX, t.clientY)
    }
    const tend = (e: TouchEvent) => {
      // The last finger up is the drop; a second finger lifting is not.
      if (!touchOwned.current || e.touches.length > 0) return
      touchOwned.current = false
      if (e.cancelable) e.preventDefault() // no click on the cell under the finger
      machine.up(lastTouch.current.x, lastTouch.current.y)
    }
    const tstart = (e: TouchEvent) => {
      // A second finger down while carrying: a pinch, not a drop.
      if (touchOwned.current && e.touches.length > 1) {
        touchOwned.current = false
        machine.cancel()
      }
    }
    const tcancel = () => {
      if (!touchOwned.current) return
      touchOwned.current = false
      machine.cancel()
    }
    document.addEventListener('touchstart', tstart)
    document.addEventListener('touchmove', tmove, { passive: false })
    document.addEventListener('touchend', tend, { passive: false })
    document.addEventListener('touchcancel', tcancel)
    return () => {
      document.removeEventListener('touchstart', tstart)
      document.removeEventListener('touchmove', tmove)
      document.removeEventListener('touchend', tend)
      document.removeEventListener('touchcancel', tcancel)
    }
  }, [machine])

  useEffect(() => {
    // A finger is steered by the touch listeners above. Its pointer events are
    // ignored here: a phone fires pointercancel when it takes a gesture for a
    // scroll, and that must not drop an event the finger is carrying.
    const move = (e: PointerEvent) => { if (e.pointerType !== 'touch') machine.move(e.clientX, e.clientY) }
    const up = (e: PointerEvent) => { if (e.pointerType !== 'touch') machine.up(e.clientX, e.clientY) }
    const cancel = (e: PointerEvent) => { if (e.pointerType !== 'touch') machine.cancel() }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
    window.addEventListener('pointercancel', cancel)
    return () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      window.removeEventListener('pointercancel', cancel)
    }
  }, [machine])

  return { onItemPointerDown, onItemTouchLift, over, draggingId, wasDrag: machine.wasDrag }
}
