import { describe, it, expect, vi } from 'vitest'
import { createMonthDrag } from './monthDrag'

function setup(dayAt: (x: number, y: number) => string | null = (x) => (x < 100 ? '2026-09-24' : x < 200 ? '2026-09-25' : null)) {
  const onMove = vi.fn()
  const onOver = vi.fn()
  const onDragging = vi.fn()
  const deferred: Array<() => void> = []
  const drag = createMonthDrag<string>({
    threshold: 5,
    dayAt,
    onMove,
    onOver,
    onDragging,
    defer: (fn) => deferred.push(fn),
  })
  return { drag, onMove, onOver, onDragging, flush: () => deferred.splice(0).forEach((f) => f()) }
}

describe('createMonthDrag', () => {
  it('moves the event to the day it is dropped on', () => {
    const { drag, onMove } = setup()
    drag.down('ev', '2026-09-24', 50, 10)
    drag.move(150, 10)
    drag.up(150, 10)
    expect(onMove).toHaveBeenCalledWith('ev', '2026-09-24', '2026-09-25')
  })

  it('treats a press that stays under the threshold as a click', () => {
    const { drag, onMove, onDragging } = setup()
    drag.down('ev', '2026-09-24', 50, 10)
    drag.move(53, 12)
    drag.up(53, 12)
    expect(onMove).not.toHaveBeenCalled()
    expect(onDragging).not.toHaveBeenCalled()
    expect(drag.wasDrag()).toBe(false)
  })

  it('does not move when dropped on its own day or on no day', () => {
    const own = setup()
    own.drag.down('ev', '2026-09-24', 50, 10)
    own.drag.move(80, 10)
    own.drag.up(80, 10)
    expect(own.onMove).not.toHaveBeenCalled()

    const nowhere = setup()
    nowhere.drag.down('ev', '2026-09-24', 50, 10)
    nowhere.drag.move(300, 10)
    nowhere.drag.up(300, 10)
    expect(nowhere.onMove).not.toHaveBeenCalled()
  })

  it('reports the click after a drop as a drag once, until the deferred reset', () => {
    const { drag, flush } = setup()
    drag.down('ev', '2026-09-24', 50, 10)
    drag.move(80, 10)
    drag.up(80, 10)
    expect(drag.wasDrag()).toBe(true)
    expect(drag.wasDrag()).toBe(false)

    // A drop that fires no click: the flag must not eat the next real click.
    drag.down('ev', '2026-09-24', 50, 10)
    drag.move(300, 10)
    drag.up(300, 10)
    flush()
    expect(drag.wasDrag()).toBe(false)
  })

  it('pointercancel ends the drag: the next click does not finish it', () => {
    const { drag, onMove, onOver, onDragging } = setup()
    drag.down('ev', '2026-09-24', 50, 10)
    drag.move(150, 10)
    drag.cancel()
    expect(onOver).toHaveBeenLastCalledWith(null)
    expect(onDragging).toHaveBeenLastCalledWith(null)
    drag.up(150, 10)
    expect(onMove).not.toHaveBeenCalled()
  })
})
