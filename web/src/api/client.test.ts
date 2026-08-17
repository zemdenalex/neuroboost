import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  api,
  getStoredToken,
  setStoredToken,
  clearStoredToken,
  setAuthToken,
  getAuthToken,
  isTokenExpired,
  getTokenDaysRemaining,
} from './client'

function mockResponse(status: number, body: unknown): Response {
  return {
    status,
    ok: status >= 200 && status < 300,
    json: async () => body,
  } as unknown as Response
}

const fetchMock = vi.fn()

beforeEach(() => {
  localStorage.clear()
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('token storage', () => {
  it('sets, reads, and clears the token', () => {
    setStoredToken('abc')
    expect(getStoredToken()).toBe('abc')
    clearStoredToken()
    expect(getStoredToken()).toBeNull()
  })

  it('setAuthToken stores a token and clears on null', () => {
    setAuthToken('t1')
    expect(getAuthToken()).toBe('t1')
    setAuthToken(null)
    expect(getAuthToken()).toBeNull()
  })

  it('isTokenExpired: false without expiry, true when past, false when future', () => {
    expect(isTokenExpired()).toBe(false)
    const nowSec = Math.floor(Date.now() / 1000)
    setStoredToken('t', nowSec - 100)
    expect(isTokenExpired()).toBe(true)
    setStoredToken('t', nowSec + 10000)
    expect(isTokenExpired()).toBe(false)
  })

  it('getTokenDaysRemaining: 0 without expiry or when past, whole days otherwise', () => {
    expect(getTokenDaysRemaining()).toBe(0)
    const nowSec = Math.floor(Date.now() / 1000)
    setStoredToken('t', nowSec + 10 * 86400 + 100) // ~10 days out
    const d = getTokenDaysRemaining()
    expect(d).toBeGreaterThanOrEqual(9)
    expect(d).toBeLessThanOrEqual(10)
    setStoredToken('t', nowSec - 100)
    expect(getTokenDaysRemaining()).toBe(0)
  })
})

describe('api request', () => {
  it('unwraps a { data } envelope', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: [1, 2, 3] }))
    expect(await api.get<number[]>('/things')).toEqual([1, 2, 3])
  })

  it('returns the raw payload when there is no data key', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { ok: true, db: 'up' }))
    expect(await api.get('/health', false)).toEqual({ ok: true, db: 'up' })
  })

  it('attaches a Bearer token when authenticated', async () => {
    setStoredToken('tok123')
    fetchMock.mockResolvedValue(mockResponse(200, { data: {} }))
    await api.get('/secure')
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>)['Authorization']).toBe('Bearer tok123')
  })

  it('omits the token when requireAuth is false', async () => {
    setStoredToken('tok123')
    fetchMock.mockResolvedValue(mockResponse(200, { data: {} }))
    await api.get('/public', false)
    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect((init.headers as Record<string, string>)['Authorization']).toBeUndefined()
  })

  it('posts a JSON body with the right method and content type', async () => {
    fetchMock.mockResolvedValue(mockResponse(200, { data: { id: 1 } }))
    await api.post('/things', { name: 'x' })
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toContain('/things')
    expect(init.method).toBe('POST')
    expect(init.body).toBe(JSON.stringify({ name: 'x' }))
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json')
  })

  it('returns {} for 204 No Content', async () => {
    fetchMock.mockResolvedValue(mockResponse(204, null))
    expect(await api.delete('/things/1')).toEqual({})
  })

  it('extracts error messages from several payload shapes', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse(400, { error: { message: 'nested' } }))
    await expect(api.get('/a')).rejects.toThrow('nested')

    fetchMock.mockResolvedValueOnce(mockResponse(400, { message: 'flat' }))
    await expect(api.get('/b')).rejects.toThrow('flat')

    fetchMock.mockResolvedValueOnce(mockResponse(400, { error: 'stringy' }))
    await expect(api.get('/c')).rejects.toThrow('stringy')

    fetchMock.mockResolvedValueOnce(mockResponse(500, {}))
    await expect(api.get('/d')).rejects.toThrow('Request failed')
  })

  it('on 401 clears the token and throws Unauthorized', async () => {
    setStoredToken('tok')
    vi.stubGlobal('location', { href: '' }) // avoid jsdom navigation
    fetchMock.mockResolvedValue(mockResponse(401, { message: 'nope' }))
    await expect(api.get('/secure')).rejects.toThrow('Unauthorized')
    expect(getStoredToken()).toBeNull()
  })
})
