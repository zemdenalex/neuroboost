import { describe, it, expect } from 'vitest'
import { statusToColumn, COLUMN_TO_STATUS, type KanbanColumnId } from './kanban'
import type { TaskStatus } from '../../api/tasks'

describe('statusToColumn', () => {
  const cases: Array<[TaskStatus, KanbanColumnId]> = [
    ['TODO', 'TODO'],
    ['IN_PROGRESS', 'IN_PROGRESS'],
    ['SCHEDULED', 'SCHEDULED'],
    ['DONE', 'DONE'],
    ['CANCELLED', 'DONE'],
  ]

  it.each(cases)('%s renders in %s', (status, column) => {
    expect(statusToColumn(status)).toBe(column)
  })

  // The rule that is easy to "fix" into a bug: a cancelled task is not done,
  // but the board has no column for it, and dropping it out of view would read
  // as data loss.
  it('shows cancelled tasks under Done rather than hiding them', () => {
    expect(statusToColumn('CANCELLED')).toBe('DONE')
  })

  // Every status the type allows must land somewhere. A new status added to
  // TaskStatus without a case here would silently pile up in TODO — this test
  // does not catch that (the default is the point), but it does catch a case
  // being deleted.
  it('never returns a column outside the board', () => {
    const columns: KanbanColumnId[] = ['INBOX', 'TODO', 'IN_PROGRESS', 'SCHEDULED', 'DONE']
    for (const [status] of cases) {
      expect(columns).toContain(statusToColumn(status))
    }
  })

  it('falls back to TODO for an unknown status instead of dropping the task', () => {
    // Cast deliberately: the point is what happens when the API returns a
    // status this build does not know about, which types cannot prevent.
    expect(statusToColumn('SOMETHING_NEW' as TaskStatus)).toBe('TODO')
  })
})

describe('COLUMN_TO_STATUS', () => {
  it('writes TODO for the INBOX column, which the backend has no status for', () => {
    expect(COLUMN_TO_STATUS.INBOX).toBe('TODO')
  })

  it('covers every column', () => {
    // A missing entry would be sent to the API as undefined.
    const columns: KanbanColumnId[] = ['INBOX', 'TODO', 'IN_PROGRESS', 'SCHEDULED', 'DONE']
    for (const c of columns) {
      expect(typeof COLUMN_TO_STATUS[c], `${c} has no status`).toBe('string')
    }
  })

  it('keeps a task in the column it was dropped into, except for INBOX', () => {
    // Otherwise the card jumps elsewhere on drop and the move looks broken.
    const columns: KanbanColumnId[] = ['TODO', 'IN_PROGRESS', 'SCHEDULED', 'DONE']
    for (const c of columns) {
      expect(statusToColumn(COLUMN_TO_STATUS[c]), `dropping into ${c}`).toBe(c)
    }
    // INBOX is the documented exception: it is a staging view, and a task
    // dropped there becomes a plain TODO and renders in the TODO column.
    expect(statusToColumn(COLUMN_TO_STATUS.INBOX)).toBe('TODO')
  })
})
