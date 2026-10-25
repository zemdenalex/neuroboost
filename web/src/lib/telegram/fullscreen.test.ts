import { describe, expect, it, vi } from 'vitest'
import {
  FULLSCREEN_KEY,
  applyFullscreenAtLaunch,
  fullscreenSupported,
  readFullscreenPref,
  setFullscreen,
  storeFullscreenPref,
  watchFullscreen,
} from './fullscreen'
import type { TgWebApp } from './webApp'

function app(over: Partial<TgWebApp> & { version?: string } = {}): TgWebApp {
  const version = over.version ?? '8.0'
  return {
    version,
    initData: 'x',
    ready: vi.fn(),
    expand: vi.fn(),
    isVersionAtLeast: (v: string) => parseFloat(version) >= parseFloat(v),
    BackButton: { show: vi.fn(), hide: vi.fn(), onClick: vi.fn(), offClick: vi.fn() },
    requestFullscreen: vi.fn(),
    exitFullscreen: vi.fn(),
    isFullscreen: false,
    ...over,
  }
}

const memory = (initial: Record<string, string> = {}) => {
  const data = { ...initial }
  return { getItem: (k: string) => data[k] ?? null, setItem: (k: string, v: string) => void (data[k] = v) }
}

describe('fullscreenSupported', () => {
  it('needs Bot API 8.0 and the method', () => {
    expect(fullscreenSupported(app())).toBe(true)
    expect(fullscreenSupported(app({ version: '7.10' }))).toBe(false)
    expect(fullscreenSupported(app({ requestFullscreen: undefined }))).toBe(false)
  })
})

describe('the saved choice', () => {
  it('is off until someone turns it on, and survives a storage that throws', () => {
    const store = memory()
    expect(readFullscreenPref(store)).toBe(false)
    storeFullscreenPref(store, true)
    expect(readFullscreenPref(store)).toBe(true)
    storeFullscreenPref(store, false)
    expect(readFullscreenPref(store)).toBe(false)
    const broken = {
      getItem: () => {
        throw new Error('blocked')
      },
      setItem: () => {
        throw new Error('blocked')
      },
    }
    expect(readFullscreenPref(broken)).toBe(false)
    expect(() => storeFullscreenPref(broken, true)).not.toThrow()
  })
})

describe('setFullscreen', () => {
  it('asks once, and only for what is not already so', () => {
    const wa = app()
    setFullscreen(wa, true)
    expect(wa.requestFullscreen).toHaveBeenCalledTimes(1)
    setFullscreen(wa, false)
    expect(wa.exitFullscreen).not.toHaveBeenCalled()
    const on = app({ isFullscreen: true })
    setFullscreen(on, true)
    expect(on.requestFullscreen).not.toHaveBeenCalled()
    setFullscreen(on, false)
    expect(on.exitFullscreen).toHaveBeenCalledTimes(1)
  })

  it('does nothing on a client without it, and swallows a throwing client', () => {
    const old = app({ version: '7.10' })
    setFullscreen(old, true)
    expect(old.requestFullscreen).not.toHaveBeenCalled()
    const bad = app({
      requestFullscreen: () => {
        throw new Error('nope')
      },
    })
    expect(() => setFullscreen(bad, true)).not.toThrow()
  })
})

describe('applyFullscreenAtLaunch', () => {
  it('requests full screen only when the person chose it', () => {
    const wa = app()
    applyFullscreenAtLaunch(wa, memory())
    expect(wa.requestFullscreen).not.toHaveBeenCalled()
    applyFullscreenAtLaunch(wa, memory({ 'nb.tg.fullscreen': '1' }))
    expect(wa.requestFullscreen).toHaveBeenCalledTimes(1)
  })
})

describe('watchFullscreen', () => {
  it('mirrors Telegram state onto the root class, now and on each change', () => {
    const calls: [string, boolean][] = []
    const root = { classList: { toggle: (c: string, on: boolean) => void calls.push([c, on]) } }
    let handler: () => void = () => {}
    const wa = app({ isFullscreen: false, onEvent: (_e, cb) => void (handler = cb) })
    watchFullscreen(wa, root)
    expect(calls.at(-1)).toEqual(['tg-fullscreen', false])
    wa.isFullscreen = true
    handler()
    expect(calls.at(-1)).toEqual(['tg-fullscreen', true])
  })

  it('forgets the saved choice when Telegram refuses, but not when it is already full screen', () => {
    const handlers: Record<string, () => void> = {}
    const store = memory({ [FULLSCREEN_KEY]: '1' })
    const wa = app({ isFullscreen: false, onEvent: (e, cb) => void (handlers[e] = cb) })
    watchFullscreen(wa, { classList: { toggle: () => {} } }, store)
    wa.isFullscreen = true
    handlers.fullscreenFailed() // e.g. ALREADY_FULLSCREEN
    expect(readFullscreenPref(store)).toBe(true)
    wa.isFullscreen = false
    handlers.fullscreenFailed() // UNSUPPORTED
    expect(readFullscreenPref(store)).toBe(false)
  })
})
