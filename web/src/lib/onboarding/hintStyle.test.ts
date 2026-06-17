import { describe, it, expect } from 'vitest'
import { parseHintStyle, getHintStyle, setHintStyle } from './hintStyle'

function fakeStorage(initial: Record<string, string> = {}): Storage {
  const map = new Map(Object.entries(initial))
  return {
    getItem: (k) => (map.has(k) ? (map.get(k) as string) : null),
    setItem: (k, v) => { map.set(k, String(v)) },
    removeItem: (k) => { map.delete(k) },
    clear: () => map.clear(),
    key: (i) => Array.from(map.keys())[i] ?? null,
    get length() { return map.size },
  } as Storage
}

describe('parseHintStyle', () => {
  it('accepts the three valid styles', () => {
    expect(parseHintStyle('bubbles')).toBe('bubbles')
    expect(parseHintStyle('walkthrough')).toBe('walkthrough')
    expect(parseHintStyle('markers')).toBe('markers')
  })
  it('defaults to bubbles for null/unknown/garbage', () => {
    expect(parseHintStyle(null)).toBe('bubbles')
    expect(parseHintStyle('')).toBe('bubbles')
    expect(parseHintStyle('BUBBLES')).toBe('bubbles')
    expect(parseHintStyle('tour')).toBe('bubbles')
  })
})

describe('getHintStyle / setHintStyle', () => {
  it('defaults to bubbles when storage is empty', () => {
    expect(getHintStyle(fakeStorage())).toBe('bubbles')
  })
  it('round-trips a written style', () => {
    const s = fakeStorage()
    setHintStyle('walkthrough', s)
    expect(getHintStyle(s)).toBe('walkthrough')
  })
  it('never throws when storage access throws', () => {
    const throwing = {
      getItem: () => { throw new Error('blocked') },
      setItem: () => { throw new Error('quota') },
      removeItem: () => {}, clear: () => {}, key: () => null, length: 0,
    } as unknown as Storage
    expect(getHintStyle(throwing)).toBe('bubbles')
    expect(() => setHintStyle('markers', throwing)).not.toThrow()
  })
})
