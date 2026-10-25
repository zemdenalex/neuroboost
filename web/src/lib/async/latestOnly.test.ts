import { describe, it, expect } from 'vitest'
import { createLatestOnly } from './latestOnly'

function deferred<T>() {
  let resolve!: (v: T) => void
  const promise = new Promise<T>((r) => (resolve = r))
  return { promise, resolve }
}

describe('createLatestOnly', () => {
  it('drops an older answer that arrives after a newer one', async () => {
    const latest = createLatestOnly()
    const seen: string[] = []
    const a = deferred<string>()
    const b = deferred<string>()
    const pa = latest(a.promise).then((v) => v !== undefined && seen.push(v))
    const pb = latest(b.promise).then((v) => v !== undefined && seen.push(v))
    b.resolve('november')
    a.resolve('october')
    await Promise.all([pa, pb])
    expect(seen).toEqual(['november'])
  })

  it('passes the answer through when nothing newer was asked', async () => {
    const latest = createLatestOnly()
    expect(await latest(Promise.resolve(42))).toBe(42)
  })
})
