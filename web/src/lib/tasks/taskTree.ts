/**
 * Subtasks in the web task list (N2 of pass 3). Rows keep their priority
 * groups; a subtask is drawn right under its parent, indented, wherever its own
 * priority would have put it. Only when the parent is not in the list
 * (filtered, closed) does it stay in its own group, unindented.
 */
export interface TreeRow<T> {
  task: T
  depth: number
}

type Node = { id: string; parent_id?: string }

const MAX_DEPTH = 5

export function nestGroups<T extends Node>(groups: Map<number, T[]>): Map<number, TreeRow<T>[]> {
  const all = Array.from(groups.values()).flat()
  const visible = new Set(all.map((t) => t.id))
  const children = new Map<string, T[]>()
  for (const task of all) {
    if (task.parent_id && task.parent_id !== task.id && visible.has(task.parent_id)) {
      const list = children.get(task.parent_id) ?? []
      list.push(task)
      children.set(task.parent_id, list)
    }
  }
  const placed = new Set<string>()
  const walk = (task: T, depth: number, out: TreeRow<T>[]) => {
    if (placed.has(task.id)) return
    placed.add(task.id)
    out.push({ task, depth })
    if (depth >= MAX_DEPTH) return
    for (const child of children.get(task.id) ?? []) walk(child, depth + 1, out)
  }
  const isNested = (task: T) => Boolean(task.parent_id && task.parent_id !== task.id && visible.has(task.parent_id))
  const result = new Map<number, TreeRow<T>[]>()
  for (const [priority, list] of groups) {
    const out: TreeRow<T>[] = []
    for (const task of list) if (!isNested(task)) walk(task, 0, out)
    result.set(priority, out)
  }
  // A cycle (a is b's parent and b is a's) has no root: its tasks would vanish.
  // They are drawn flat in their own group instead.
  for (const [priority, list] of groups) {
    for (const task of list) if (!placed.has(task.id)) walk(task, 0, result.get(priority)!)
  }
  return result
}

/** Done and total subtasks per parent, cancelled ones not counted. */
export function subtaskProgress(tasks: { id: string; parent_id?: string; status: string }[]): Map<string, { done: number; total: number }> {
  const out = new Map<string, { done: number; total: number }>()
  for (const task of tasks) {
    if (!task.parent_id || task.status === 'CANCELLED') continue
    const p = out.get(task.parent_id) ?? { done: 0, total: 0 }
    p.total++
    if (task.status === 'DONE') p.done++
    out.set(task.parent_id, p)
  }
  return out
}
