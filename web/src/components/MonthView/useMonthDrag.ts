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

  useEffect(() => {
    const move = (e: PointerEvent) => machine.move(e.clientX, e.clientY)
    const up = (e: PointerEvent) => machine.up(e.clientX, e.clientY)
    const cancel = () => machine.cancel()
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
    window.addEventListener('pointercancel', cancel)
    return () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      window.removeEventListener('pointercancel', cancel)
    }
  }, [machine])

  return { onItemPointerDown, over, draggingId, wasDrag: machine.wasDrag }
}
