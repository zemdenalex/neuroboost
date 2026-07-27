import { describe, expect, it } from 'vitest'
import { sortWithinPriority, type SortableTask } from './sortTasks'

function task(over: Partial<SortableTask>): SortableTask {
  return { id: 'x', status: 'TODO', ...over }
}

describe('sortWithinPriority', () => {
  it('puts dated tasks before undated ones', () => {
    // A task with a deadline is actionable in a way an undated one is not;
    // burying it under a wall of someday-tasks is what makes a long list
    // useless.
    const out = sortWithinPriority([
      task({ id: 'no-date' }),
      task({ id: 'dated', dueDate: '2026-08-01T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['dated', 'no-date'])
  })

  it('orders dated tasks soonest first', () => {
    const out = sortWithinPriority([
      task({ id: 'later', dueDate: '2026-08-03T09:00:00Z' }),
      task({ id: 'sooner', dueDate: '2026-08-01T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['sooner', 'later'])
  })

  it('breaks ties on newest-created, so a task just typed is visible', () => {
    // The quick-add flow adds to the top of its group; if new tasks sank to
    // the bottom, Enter would look like it did nothing.
    const out = sortWithinPriority([
      task({ id: 'old', createdAt: '2026-07-01T09:00:00Z' }),
      task({ id: 'new', createdAt: '2026-07-27T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['new', 'old'])
  })

  it('falls back to id so the order is never arbitrary', () => {
    // Two tasks created in the same batch share a timestamp to the second.
    // Without a final tiebreak they would swap places between renders.
    const out = sortWithinPriority([
      task({ id: 'b', createdAt: '2026-07-27T09:00:00Z' }),
      task({ id: 'a', createdAt: '2026-07-27T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['a', 'b'])
  })

  it('sinks completed tasks below open ones regardless of date', () => {
    const out = sortWithinPriority([
      task({ id: 'done', status: 'DONE', dueDate: '2026-08-01T09:00:00Z' }),
      task({ id: 'open', status: 'TODO', dueDate: '2026-08-09T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['open', 'done'])
  })

  it('treats an unparseable due date as undated rather than throwing', () => {
    const out = sortWithinPriority([
      task({ id: 'broken', dueDate: 'not-a-date' }),
      task({ id: 'fine', dueDate: '2026-08-01T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['fine', 'broken'])
  })

  it('does not mutate its input', () => {
    const input = [task({ id: 'b' }), task({ id: 'a' })]
    const out = sortWithinPriority(input)
    expect(input.map(t => t.id)).toEqual(['b', 'a'])
    expect(out).not.toBe(input)
  })

  it('handles an empty list', () => {
    expect(sortWithinPriority([])).toEqual([])
  })

  it('reads snake_case fields too — the Tasks page uses the raw API shape', () => {
    // This repo carries two task representations. A comparator that only
    // understood camelCase would silently treat every Tasks-page row as
    // undated and uncreated, i.e. sort by id alone.
    const out = sortWithinPriority([
      task({ id: 'later', due_date: '2026-08-03T09:00:00Z' }),
      task({ id: 'sooner', due_date: '2026-08-01T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['sooner', 'later'])
  })

  it('does not mix casings up when both stacks appear in one list', () => {
    const out = sortWithinPriority([
      task({ id: 'snake', due_date: '2026-08-05T09:00:00Z' }),
      task({ id: 'camel', dueDate: '2026-08-02T09:00:00Z' }),
    ])
    expect(out.map(t => t.id)).toEqual(['camel', 'snake'])
  })
})
