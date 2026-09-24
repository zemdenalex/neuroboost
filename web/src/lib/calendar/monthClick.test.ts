import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createDayClick, CLICK_WAIT_MS } from './monthClick'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createDayClick', () => {
  it('a single click opens the day after the wait', () => {
    const open = vi.fn()
    const create = vi.fn()
    const click = createDayClick(open, create)
    click('2026-09-24', 1)
    expect(open).not.toHaveBeenCalled()
    vi.advanceTimersByTime(CLICK_WAIT_MS)
    expect(open).toHaveBeenCalledWith('2026-09-24')
    expect(create).not.toHaveBeenCalled()
  })

  it('a double click creates and never opens the week', () => {
    const open = vi.fn()
    const create = vi.fn()
    const click = createDayClick(open, create)
    // The browser sends click (detail 1), click (detail 2), then dblclick.
    click('2026-09-24', 1)
    click('2026-09-24', 2)
    vi.advanceTimersByTime(CLICK_WAIT_MS * 3)
    expect(create).toHaveBeenCalledTimes(1)
    expect(create).toHaveBeenCalledWith('2026-09-24')
    expect(open).not.toHaveBeenCalled()
  })

  it('two single clicks on different days far apart open each', () => {
    const open = vi.fn()
    const click = createDayClick(open, vi.fn())
    click('2026-09-24', 1)
    vi.advanceTimersByTime(CLICK_WAIT_MS)
    click('2026-09-25', 1)
    vi.advanceTimersByTime(CLICK_WAIT_MS)
    expect(open.mock.calls).toEqual([['2026-09-24'], ['2026-09-25']])
  })

  it('cancel drops a pending single click (the view unmounted)', () => {
    const open = vi.fn()
    const click = createDayClick(open, vi.fn())
    click('2026-09-24', 1)
    click.cancel()
    vi.advanceTimersByTime(CLICK_WAIT_MS)
    expect(open).not.toHaveBeenCalled()
  })
})
