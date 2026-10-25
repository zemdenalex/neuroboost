import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createHoldTracker } from './touchHold'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createHoldTracker — hold or scroll, decided before the browser decides for us', () => {
  it('lifts after the hold time when the finger stayed put', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    vi.advanceTimersByTime(399)
    expect(onLift).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(onLift).toHaveBeenCalledWith(100, 200)
    expect(t.lifted()).toBe(true)
  })

  it('a finger that travels past the slop before the hold is a scroll, and never lifts', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    vi.advanceTimersByTime(200)
    t.move(100, 215)
    vi.advanceTimersByTime(1000)
    expect(onLift).not.toHaveBeenCalled()
    expect(t.lifted()).toBe(false)
  })

  it('a tremor inside the slop does not cancel the hold', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    t.move(103, 204)
    vi.advanceTimersByTime(400)
    expect(onLift).toHaveBeenCalledTimes(1)
  })

  it('lifts at the point the finger is on when the time is up, not where it first touched', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    t.move(104, 203)
    vi.advanceTimersByTime(400)
    expect(onLift).toHaveBeenCalledWith(104, 203)
  })

  it('lifting early (finger up before the hold) is a tap: nothing happens', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    vi.advanceTimersByTime(300)
    t.end()
    vi.advanceTimersByTime(1000)
    expect(onLift).not.toHaveBeenCalled()
  })

  it('after the lift, moving the finger does not cancel it, and end() clears the lifted state', () => {
    const onLift = vi.fn()
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift })
    t.start(100, 200)
    vi.advanceTimersByTime(400)
    t.move(100, 400)
    expect(t.lifted()).toBe(true)
    t.end()
    expect(t.lifted()).toBe(false)
  })

  it('a pending hold is reported as active so the context menu can be suppressed', () => {
    const t = createHoldTracker({ holdMs: 400, slopPx: 8, onLift: () => {} })
    expect(t.active()).toBe(false)
    t.start(1, 1)
    expect(t.active()).toBe(true)
    t.end()
    expect(t.active()).toBe(false)
  })
})
