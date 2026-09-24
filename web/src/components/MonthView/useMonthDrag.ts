import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import type { NbEvent } from '../../types'
import { DRAG_THRESHOLD_PX } from '../Calendar/WeekGrid/weekgrid.constants'

interface Pending {
  event: NbEvent
  fromDay: string
  x: number
  y: number
  moved: boolean
}

/** The day cell under a screen point, read from its data-day attribute. */
function dayAt(x: number, y: number): string | null {
  const el = document.elementFromPoint(x, y)
  const cell = el instanceof Element ? el.closest('[data-day]') : null
  return cell?.getAttribute('data-day') ?? null
}

/**
 * Dragging an event to another day of the month (spec R8).
 *
 * Pointer events with the week grid's threshold, not HTML5 drag-and-drop: a
 * press that never moves DRAG_THRESHOLD_PX stays a click. `wasDrag()` lets the
 * cell's click handler ignore the click the browser fires after a drop.
 */
export function useMonthDrag(onMoveToDay: (event: NbEvent, fromDay: string, toDay: string) => void) {
  const pending = useRef<Pending | null>(null)
  const justDropped = useRef(false)
  const [over, setOver] = useState<string | null>(null)
  const [draggingId, setDraggingId] = useState<string | null>(null)

  const onItemPointerDown = useCallback((event: NbEvent, e: ReactPointerEvent) => {
    if (e.button !== 0 || e.pointerType === 'touch') return
    const fromDay = (e.currentTarget as Element).closest('[data-day]')?.getAttribute('data-day')
    if (!fromDay) return
    pending.current = { event, fromDay, x: e.clientX, y: e.clientY, moved: false }
  }, [])

  useEffect(() => {
    const move = (e: PointerEvent) => {
      const p = pending.current
      if (!p) return
      if (!p.moved && Math.hypot(e.clientX - p.x, e.clientY - p.y) < DRAG_THRESHOLD_PX) return
      if (!p.moved) {
        p.moved = true
        setDraggingId(p.event.id)
      }
      setOver(dayAt(e.clientX, e.clientY))
    }
    const up = (e: PointerEvent) => {
      const p = pending.current
      pending.current = null
      if (!p) return
      setOver(null)
      setDraggingId(null)
      if (!p.moved) return
      // The browser fires its click right after pointerup, in the same task;
      // clear the flag after that, or a drop that lands on no cell would eat
      // the next real click.
      justDropped.current = true
      setTimeout(() => {
        justDropped.current = false
      }, 0)
      const to = dayAt(e.clientX, e.clientY)
      if (to && to !== p.fromDay) onMoveToDay(p.event, p.fromDay, to)
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
    return () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
    }
  }, [onMoveToDay])

  /** True once, right after a drop: the click that follows it is not a click. */
  const wasDrag = useCallback(() => {
    const was = justDropped.current
    justDropped.current = false
    return was
  }, [])

  return { onItemPointerDown, over, draggingId, wasDrag }
}
