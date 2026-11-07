import { describe, it, expect } from 'vitest'
import { ApiError } from '../../api/client'
import { endsSession, loadWithRetry } from './startupSession'

// What these protect: a person is logged out on app start only when the server
// rejects the credential — never because the network dropped or the server was
// redeploying (07.11: any getMe failure cleared the token).
describe('restoring a session on start', () => {
  it('ends the session only on 401/403, not on a dropped network or a 5xx', () => {
    expect(endsSession(new ApiError('Unauthorized', undefined, undefined, 401))).toBe(true)
    expect(endsSession(new ApiError('Forbidden', undefined, undefined, 403))).toBe(true)
    expect(endsSession(new TypeError('Failed to fetch'))).toBe(false)
    expect(endsSession(new ApiError('Request failed', undefined, undefined, 502))).toBe(false)
    expect(endsSession(new ApiError('Request failed', undefined, undefined))).toBe(false)
  })

  it('asks again after a dropped network or a 502, and gives up after two retries', async () => {
    const waits: number[] = []
    const wait = async (ms: number) => { waits.push(ms) }
    let calls = 0
    const flaky = async () => {
      calls++
      if (calls === 1) throw new TypeError('Failed to fetch')
      if (calls === 2) throw new ApiError('Request failed', undefined, undefined, 502)
      return 'me'
    }
    expect(await loadWithRetry(flaky, wait)).toBe('me')
    expect(waits).toEqual([1000, 3000])

    calls = 0
    const down = async () => { calls++; throw new TypeError('Failed to fetch') }
    await expect(loadWithRetry(down, wait)).rejects.toThrow('Failed to fetch')
    expect(calls).toBe(3)
  })

  it('does not retry a 4xx: the answer will not change', async () => {
    let calls = 0
    const bad = async () => { calls++; throw new ApiError('Bad', 'X', undefined, 400) }
    await expect(loadWithRetry(bad, async () => {})).rejects.toThrow('Bad')
    expect(calls).toBe(1)
  })
})
