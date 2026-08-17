import { describe, it, expect } from 'vitest'
import { toTask, type RawTask } from './toTask'

const raw: RawTask = {
  id: 't1',
  user_id: 'u1',
  title: 'Дописать отчёт',
  status: 'TODO',
  priority: 2,
  estimated_minutes: 15,
  due_date: '2026-07-28T09:00:00Z',
  tags: ['work'],
  contexts: ['@computer'],
  parent_id: 'p1',
  created_at: '2026-07-27T20:00:00Z',
  updated_at: '2026-07-27T20:00:00Z',
}

describe('toTask', () => {
  it('maps snake_case fields onto the camelCase Task type', () => {
    const task = toTask(raw)
    expect(task.estimatedMinutes).toBe(15)
    expect(task.dueDate).toBe('2026-07-28T09:00:00Z')
    expect(task.parentId).toBe('p1')
    expect(task.userId).toBe('u1')
    expect(task.createdAt).toBe('2026-07-27T20:00:00Z')
  })

  it('keeps fields that need no renaming', () => {
    const task = toTask(raw)
    expect(task.id).toBe('t1')
    expect(task.title).toBe('Дописать отчёт')
    expect(task.priority).toBe(2)
    expect(task.status).toBe('TODO')
  })

  it('defaults arrays to [] and leaves absent optionals undefined', () => {
    const task = toTask({
      id: 't2',
      user_id: 'u1',
      title: 'Купить хлеб',
      status: 'TODO',
      priority: 3,
      created_at: '2026-07-27T20:00:00Z',
      updated_at: '2026-07-27T20:00:00Z',
    })
    expect(task.tags).toEqual([])
    expect(task.contexts).toEqual([])
    expect(task.estimatedMinutes).toBeUndefined()
    expect(task.dueDate).toBeUndefined()
    expect(task.parentId).toBeUndefined()
  })
})
