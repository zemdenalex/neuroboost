import { describe, it, expect } from 'vitest'
import { nestGroups, subtaskProgress } from './taskTree'

// Subtasks on the web (N2 of pass 3: «перенести всё новое из бота в веб»).
// The bot shows a task's subtasks on its card since 24.09; the web listed every
// task flat, so a subtask made in the bot was a stranger in the list.
type T = { id: string; parent_id?: string; status: string }
const t = (id: string, parent_id?: string, status = 'TODO'): T => ({ id, parent_id, status })
const ids = (rows: { task: T; depth: number }[]) => rows.map((r) => `${'-'.repeat(r.depth)}${r.task.id}`)

describe('nestGroups', () => {
  it('puts subtasks right under their parent, indented', () => {
    const groups = new Map([[3, [t('a'), t('a1', 'a'), t('b'), t('a2', 'a')]]])
    expect(ids(nestGroups(groups).get(3)!)).toEqual(['a', '-a1', '-a2', 'b'])
  })
  it('a subtask of another priority still follows its parent', () => {
    const groups = new Map([
      [1, [t('urgent-child', 'p')]],
      [3, [t('p')]],
    ])
    const nested = nestGroups(groups)
    expect(ids(nested.get(1)!)).toEqual([])
    expect(ids(nested.get(3)!)).toEqual(['p', '-urgent-child'])
  })
  it('a subtask whose parent is not in the list stays where it is, unindented', () => {
    const groups = new Map([[3, [t('orphan', 'gone')]]])
    expect(ids(nestGroups(groups).get(3)!)).toEqual(['orphan'])
  })
  it('nests deeper levels and survives a cycle', () => {
    const groups = new Map([[3, [t('x'), t('y', 'x'), t('z', 'y'), t('c1', 'c2'), t('c2', 'c1')]]])
    const rows = ids(nestGroups(groups).get(3)!)
    expect(rows.slice(0, 3)).toEqual(['x', '-y', '--z'])
    expect(rows).toHaveLength(5)
  })
})

describe('subtaskProgress', () => {
  it('counts done and total per parent from every task, filtered or not', () => {
    const p = subtaskProgress([t('p'), t('s1', 'p', 'DONE'), t('s2', 'p'), t('s3', 'p', 'CANCELLED')])
    expect(p.get('p')).toEqual({ done: 1, total: 2 })
    expect(p.has('s1')).toBe(false)
  })
})
