import type { Task } from '../types'

/**
 * Shape the Go API actually sends for a task (snake_case).
 * Mirrors api-go/internal/tasks/types.go:30-48.
 */
export interface RawTask {
  id: string
  user_id: string
  title: string
  description?: string
  status: Task['status']
  priority: number
  estimated_minutes?: number
  actual_minutes?: number
  due_date?: string
  tags?: string[]
  contexts?: string[]
  energy?: number
  parent_id?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

/**
 * Convert an API task into the camelCase Task used by TaskSidebar and Calendar.
 *
 * Without this, getTasks cast raw rows straight to Task and every renamed
 * field read back undefined — which is why dragging a task onto the calendar
 * always scheduled 60 minutes instead of its real estimate.
 */
export function toTask(raw: RawTask): Task {
  return {
    id: raw.id,
    userId: raw.user_id,
    title: raw.title,
    description: raw.description,
    status: raw.status,
    priority: raw.priority,
    estimatedMinutes: raw.estimated_minutes,
    dueDate: raw.due_date,
    tags: raw.tags ?? [],
    contexts: raw.contexts ?? [],
    energy: raw.energy,
    parentId: raw.parent_id,
    completedAt: raw.completed_at,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  }
}
