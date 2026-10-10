import { describe, it, expect } from 'vitest'
import { dropPriority, priorityToQuadrant, QUADRANT_TO_PRIORITY, taskQuadrant, type QuadrantId } from './eisenhower'

describe('priorityToQuadrant', () => {
  // The whole scale, spelled out. A table is the point here: the mapping is
  // small enough that a partial test would look complete while leaving the
  // inverted end — where the mistakes happen — unexamined.
  const cases: Array<[number, QuadrantId, string]> = [
    [1, 'q1', 'Emergency'],
    [2, 'q1', 'ASAP'],
    [3, 'q2', 'Normal'],
    [4, 'q3', 'Low'],
    [5, 'q4', 'If Possible'],
    [0, 'q4', 'Buffer'],
  ]

  it.each(cases)('priority %i (%s) → %s', (priority, quadrant) => {
    expect(priorityToQuadrant(priority)).toBe(quadrant)
  })

  // 🔴 The one that matters. The scale is inverted, so the intuitive reading —
  // "5 is the highest priority" — puts emergencies in Eliminate and buffer work
  // in Do First. This asserts the direction itself, not one value.
  it('puts the most urgent priority in Do First and the least in Eliminate', () => {
    expect(priorityToQuadrant(1)).toBe('q1')
    expect(priorityToQuadrant(5)).toBe('q4')
    expect(priorityToQuadrant(1)).not.toBe(priorityToQuadrant(5))
  })

  it('sends unrecognised priorities to Eliminate rather than throwing', () => {
    // A value out of range must not crash the board; it lands in the quadrant
    // that costs the user least if it is wrong.
    expect(priorityToQuadrant(99)).toBe('q4')
    expect(priorityToQuadrant(-1)).toBe('q4')
    expect(priorityToQuadrant(2.5)).toBe('q4')
  })
})

describe('QUADRANT_TO_PRIORITY', () => {
  it('assigns a priority that lands back in the same quadrant', () => {
    // Otherwise a task would visibly jump out of the quadrant it was just
    // dropped into — the drop would appear to fail.
    for (const q of ['q1', 'q2', 'q3', 'q4'] as QuadrantId[]) {
      expect(priorityToQuadrant(QUADRANT_TO_PRIORITY[q]), `dropping into ${q}`).toBe(q)
    }
  })

  it('covers every quadrant', () => {
    // A missing entry would read as undefined and be sent to the API as the
    // task's new priority.
    for (const q of ['q1', 'q2', 'q3', 'q4'] as QuadrantId[]) {
      expect(typeof QUADRANT_TO_PRIORITY[q], `${q} has no priority`).toBe('number')
    }
  })

  it('is not a lossless inverse, and that is deliberate', () => {
    // Documented, not accidental: q1 holds priorities 1 and 2, so a task at 2
    // dragged out of Do First and back returns as 1.
    expect(priorityToQuadrant(2)).toBe('q1')
    expect(QUADRANT_TO_PRIORITY.q1).toBe(1)
  })
})

// Denis 25.09 (docs/tasks-web-cleanup.md 4.9): urgency from the due date
// (within two days, or overdue), importance from priority 1-2; a drop changes
// only the priority. Until then both axes came from `priority` alone.

describe('taskQuadrant', () => {
  const today = '2026-09-25'
  it.each([
    [{ priority: 1, due_date: '2026-09-26' }, 'q1', 'important, due tomorrow'],
    [{ priority: 2, due_date: '2026-09-20' }, 'q1', 'important, overdue'],
    [{ priority: 2, due_date: '2026-10-10' }, 'q2', 'important, due later'],
    [{ priority: 1, due_date: null }, 'q2', 'important, no date'],
    [{ priority: 4, due_date: '2026-09-27' }, 'q3', 'not important, due in two days'],
    [{ priority: 3, due_date: '2026-09-28' }, 'q4', 'not important, due in three days'],
    [{ priority: 0, due_date: undefined }, 'q4', 'buffer, no date'],
  ] as const)('%o → %s (%s)', (task, quadrant, _label) => {
    void _label
    expect(taskQuadrant(task, today)).toBe(quadrant)
  })
  it('reads a full timestamp by its date', () => {
    expect(taskQuadrant({ priority: 1, due_date: '2026-09-26T21:00:00Z' }, today)).toBe('q1')
  })
})

describe('dropPriority', () => {
  it('into an important quadrant makes an unimportant task priority 2, keeps 1 and 2', () => {
    expect(dropPriority(4, 'q1')).toBe(2)
    expect(dropPriority(1, 'q2')).toBe(1)
    expect(dropPriority(2, 'q1')).toBe(2)
  })
  it('into an unimportant quadrant makes an important task priority 3, keeps the rest', () => {
    expect(dropPriority(1, 'q3')).toBe(3)
    expect(dropPriority(5, 'q4')).toBe(5)
    expect(dropPriority(0, 'q3')).toBe(0)
  })
})
