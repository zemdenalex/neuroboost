import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createDayClick, routeCellClick, readClickWait, CLICK_WAIT_MS } from './monthClick'

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

describe('routeCellClick', () => {
  function actions() {
    const click = Object.assign(vi.fn(), { cancel: vi.fn() })
    const splitClick = Object.assign(vi.fn(), { cancel: vi.fn() })
    return { click, splitClick, choose: vi.fn() }
  }

  it('ignores the click that follows a drop', () => {
    const a = actions()
    routeCellClick(false, '2026-09-24', 1, true, a)
    routeCellClick(true, '2026-09-24', 1, true, a)
    expect(a.click).not.toHaveBeenCalled()
    expect(a.splitClick).not.toHaveBeenCalled()
    expect(a.choose).not.toHaveBeenCalled()
  })

  it('sends an ordinary click to the open-week / create handler', () => {
    const a = actions()
    routeCellClick(false, '2026-09-24', 2, false, a)
    expect(a.click).toHaveBeenCalledWith('2026-09-24', 2)
    expect(a.choose).not.toHaveBeenCalled()
  })

  it('in variant D chooses at once on a click, and not again on the second click', () => {
    const a = actions()
    routeCellClick(true, '2026-09-24', 1, false, a)
    routeCellClick(true, '2026-09-24', 2, false, a)
    expect(a.choose).toHaveBeenCalledTimes(1)
    expect(a.splitClick.mock.calls).toEqual([
      ['2026-09-24', 1],
      ['2026-09-24', 2],
    ])
    expect(a.click).not.toHaveBeenCalled()
  })
})

describe('the click wait', () => {
  it('is 300 ms by default (Denis, 24.09)', () => {
    expect(CLICK_WAIT_MS).toBe(300)
  })

  it('uses the wait it is given', () => {
    const open = vi.fn()
    const click = createDayClick(open, vi.fn(), 500)
    click('2026-09-24', 1)
    vi.advanceTimersByTime(499)
    expect(open).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(open).toHaveBeenCalledWith('2026-09-24')
  })
})

describe('readClickWait', () => {
  it('reads the setting, clamps it to 150–800 and falls back to 300', () => {
    expect(readClickWait(undefined)).toBe(300)
    expect(readClickWait({ month_click_wait_ms: 450 })).toBe(450)
    expect(readClickWait({ month_click_wait_ms: 20 })).toBe(150)
    expect(readClickWait({ month_click_wait_ms: 5000 })).toBe(800)
    expect(readClickWait({ month_click_wait_ms: 'x' })).toBe(300)
  })
})
