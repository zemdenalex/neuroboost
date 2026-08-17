import { describe, it, expect } from 'vitest'
import { PRIORITY_META, priorityMeta, PRIORITY_DOT_COLORS } from './priority'

const LEVELS = [0, 1, 2, 3, 4, 5]

describe('PRIORITY_META', () => {
  it('defines all six levels with non-empty labels', () => {
    for (const lvl of LEVELS) {
      expect(PRIORITY_META[lvl]).toBeDefined()
      expect(PRIORITY_META[lvl].label.length).toBeGreaterThan(0)
    }
  })

  it('follows the red→orange→yellow→green→blue heat gradient (urgent→calm)', () => {
    expect(PRIORITY_META[1].hue).toBe('red')
    expect(PRIORITY_META[2].hue).toBe('orange')
    expect(PRIORITY_META[3].hue).toBe('yellow')
    // Regression guard: api/tasks.ts used to have 4=blue / 5=green (swapped vs the
    // calendar + sidebar). The whole app now agrees on green→blue here.
    expect(PRIORITY_META[4].hue).toBe('green')
    expect(PRIORITY_META[5].hue).toBe('blue')
    expect(PRIORITY_META[0].hue).toBe('zinc')
  })

  it('keeps every class variant on its level hue (guards future divergence)', () => {
    for (const lvl of LEVELS) {
      const m = PRIORITY_META[lvl]
      expect(m.dotClass).toContain(m.hue)
      expect(m.blockClass).toContain(m.hue)
      expect(m.sidebarClass).toContain(m.hue)
    }
  })
})

describe('PRIORITY_DOT_COLORS', () => {
  // Pinned contract: the task list, Kanban, and Eisenhower all render dots from this map.
  it('maps every level to its exact dot class', () => {
    expect(PRIORITY_DOT_COLORS).toEqual({
      0: 'bg-zinc-600',
      1: 'bg-red-600',
      2: 'bg-orange-500',
      3: 'bg-yellow-500',
      4: 'bg-green-500',
      5: 'bg-blue-500',
    })
  })
})

describe('priorityMeta', () => {
  it('returns the entry for a known level', () => {
    expect(priorityMeta(1)).toBe(PRIORITY_META[1])
  })

  it('falls back to "Must Today" (level 3) for unknown levels', () => {
    expect(priorityMeta(99)).toBe(PRIORITY_META[3])
    expect(priorityMeta(-1)).toBe(PRIORITY_META[3])
  })
})
