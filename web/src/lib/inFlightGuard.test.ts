import { describe, it, expect, vi } from 'vitest'
import { createInFlightGuard } from './inFlightGuard'

describe('createInFlightGuard', () => {
  it('runs the task only once when called re-entrantly while in flight', async () => {
    let resolve!: () => void
    const pending = new Promise<void>((r) => {
      resolve = r
    })
    const task = vi.fn(() => pending)
    const run = createInFlightGuard()

    // First call starts the task and flips the in-flight flag synchronously.
    const firstCall = run(task)
    // A second call in the same tick (e.g. an impatient double-click) must be ignored.
    const secondCall = run(task)

    expect(task).toHaveBeenCalledTimes(1)

    resolve()
    await Promise.all([firstCall, secondCall])
  })

  it('allows a new run after the previous one completes', async () => {
    const task = vi.fn(() => Promise.resolve())
    const run = createInFlightGuard()

    await run(task)
    await run(task)

    expect(task).toHaveBeenCalledTimes(2)
  })

  it('resets the in-flight flag even if the task rejects (does not get stuck)', async () => {
    const run = createInFlightGuard()
    const failing = vi.fn(() => Promise.reject(new Error('boom')))

    await run(failing).catch(() => {})
    expect(failing).toHaveBeenCalledTimes(1)

    // After a failure the guard must accept the next submit.
    const ok = vi.fn(() => Promise.resolve())
    await run(ok)
    expect(ok).toHaveBeenCalledTimes(1)
  })
})
