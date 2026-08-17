import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createDebouncedSaver } from './debouncedSaver'

type Patch = { a?: number; b?: number }

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createDebouncedSaver', () => {
  it('coalesces rapid changes into one save', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    s.schedule({ a: 1 })
    s.schedule({ a: 2 })
    s.schedule({ a: 3 })
    expect(save).not.toHaveBeenCalled()

    vi.advanceTimersByTime(300)
    expect(save).toHaveBeenCalledTimes(1)
    expect(save).toHaveBeenCalledWith({ a: 3 })
  })

  it('merges different fields instead of letting the last one win', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    s.schedule({ a: 1 })
    s.schedule({ b: 2 })
    vi.advanceTimersByTime(300)

    expect(save).toHaveBeenCalledWith({ a: 1, b: 2 })
  })

  // 🔴 The defect this file exists for. Settings.tsx cancelled the timer on
  // unmount under a comment promising it flushed, so a change followed by
  // leaving the page inside 300ms was lost — and lost silently, because the
  // "saved" toast only prints after a save that never happened.
  it('sends what is owed when flushed before the timer fires', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    s.schedule({ a: 42 })
    s.flush()

    expect(save, 'a pending change must survive an early flush').toHaveBeenCalledTimes(1)
    expect(save).toHaveBeenCalledWith({ a: 42 })
  })

  it('does not save twice when the timer would have fired after a flush', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    s.schedule({ a: 1 })
    s.flush()
    vi.advanceTimersByTime(1000)

    expect(save).toHaveBeenCalledTimes(1)
  })

  // The negative control. Without it, a flush that always fired — or one that
  // sent `{}` — would satisfy the test above while writing empty patches to the
  // API on every navigation.
  it('sends nothing when there is nothing owed', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    s.flush()
    expect(save).not.toHaveBeenCalled()

    s.schedule({ a: 1 })
    vi.advanceTimersByTime(300)
    expect(save).toHaveBeenCalledTimes(1)

    s.flush() // already saved by the timer — nothing left
    expect(save).toHaveBeenCalledTimes(1)
  })

  it('reports success only for saves that completed while mounted', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const onSaved = vi.fn()
    const s = createDebouncedSaver<Patch>({ save, onSaved })

    s.schedule({ a: 1 })
    await vi.advanceTimersByTimeAsync(300)
    expect(onSaved).toHaveBeenCalledTimes(1)

    // A flush has no onSaved by design: it runs when the owner is going away,
    // and calling back into it would be a setState on something unmounted.
    s.schedule({ b: 2 })
    s.flush()
    await Promise.resolve()
    expect(onSaved, 'a flush must not call back into a component that is leaving').toHaveBeenCalledTimes(1)
  })

  it('routes a failed timer save to onError and a failed flush to onFlushError', async () => {
    const boom = new Error('network')
    const save = vi.fn().mockRejectedValue(boom)
    const onError = vi.fn()
    const onFlushError = vi.fn()
    const s = createDebouncedSaver<Patch>({ save, onError, onFlushError })

    s.schedule({ a: 1 })
    await vi.advanceTimersByTimeAsync(300)
    expect(onError).toHaveBeenCalledWith(boom)
    expect(onFlushError).not.toHaveBeenCalled()

    s.schedule({ b: 2 })
    s.flush()
    await Promise.resolve()
    await Promise.resolve()
    expect(onFlushError).toHaveBeenCalledWith(boom)
  })

  it('hasPending reflects what would be lost right now', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save })

    expect(s.hasPending()).toBe(false)
    s.schedule({ a: 1 })
    expect(s.hasPending()).toBe(true)
    vi.advanceTimersByTime(300)
    expect(s.hasPending()).toBe(false)
  })

  it('honours a custom delay', () => {
    const save = vi.fn().mockResolvedValue(undefined)
    const s = createDebouncedSaver<Patch>({ save, delay: 1000 })

    s.schedule({ a: 1 })
    vi.advanceTimersByTime(300)
    expect(save).not.toHaveBeenCalled()
    vi.advanceTimersByTime(700)
    expect(save).toHaveBeenCalledTimes(1)
  })
})
