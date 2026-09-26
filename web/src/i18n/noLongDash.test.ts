import { describe, it, expect } from 'vitest'

// Denis: no long dashes in anything people read (bot D2; web 24.09). Every
// locale file, so a new one cannot slip past a per-namespace test.
const files = import.meta.glob('./locales/*/*.json', { eager: true, import: 'default' }) as Record<string, unknown>

function strings(o: unknown): string[] {
  if (typeof o === 'string') return [o]
  if (o && typeof o === 'object') return Object.values(o).flatMap(strings)
  return []
}

describe('web texts', () => {
  it('finds the locale files', () => {
    expect(Object.keys(files).length).toBeGreaterThan(20)
  })

  it('have no long dashes', () => {
    const bad = Object.entries(files).flatMap(([f, o]) => strings(o).filter((s) => s.includes('—')).map((s) => `${f}: ${s}`))
    expect(bad).toEqual([])
  })
})
