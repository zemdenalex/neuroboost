import { api } from './client'

export type TaskStatus = 'TODO' | 'IN_PROGRESS' | 'SCHEDULED' | 'DONE' | 'CANCELLED'
export type TaskCategory = 'EMERGENCY' | 'ASAP' | 'MUST_TODAY' | 'DEADLINE_SOON' | 'IF_POSSIBLE' | 'BUFFER'

export interface Task {
  id: string
  user_id: string
  title: string
  description?: string
  status: TaskStatus
  category?: TaskCategory
  priority: number
  estimated_minutes?: number
  actual_minutes: number
  due_date?: string
  tags: string[]
  contexts: string[]
  energy?: number
  parent_id?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export interface CreateTaskRequest {
  title: string
  description?: string
  status?: TaskStatus
  category?: TaskCategory
  priority?: number
  estimated_minutes?: number
  due_date?: string
  tags?: string[]
  contexts?: string[]
  energy?: number
  parent_id?: string
}

export interface UpdateTaskRequest {
  title?: string
  description?: string
  status?: TaskStatus
  category?: TaskCategory
  priority?: number
  estimated_minutes?: number
  due_date?: string
  tags?: string[]
  contexts?: string[]
  energy?: number
  parent_id?: string
}

export interface ScheduleTaskRequest {
  starts_at: string
  ends_at: string
  all_day?: boolean
  color?: string
}

export interface ScheduledEvent {
  id: string
  task_id: string
  title: string
  starts_at: string
  ends_at: string
  all_day: boolean
  color?: string
}

export interface ListTasksQuery {
  status?: TaskStatus
  category?: TaskCategory
  context?: string
}

export async function listTasks(query?: ListTasksQuery): Promise<Task[]> {
  const params = new URLSearchParams()
  if (query?.status) params.set('status', query.status)
  if (query?.category) params.set('category', query.category)
  if (query?.context) params.set('context', query.context)
  
  const queryString = params.toString()
  const url = queryString ? `/tasks?${queryString}` : '/tasks'
  return api.get<Task[]>(url)
}

export async function createTask(data: CreateTaskRequest): Promise<Task> {
  return api.post<Task>('/tasks', data)
}

export interface BatchRowError {
  index: number
  code: string
  message: string
}

export interface BatchCreateResponse {
  tasks: Task[]
  errors: BatchRowError[]
}

/**
 * Create many tasks in one round-trip. Rows fail independently: valid rows are
 * created and invalid ones come back in `errors` with their index.
 */
export async function createTasksBatch(tasks: CreateTaskRequest[]): Promise<BatchCreateResponse> {
  return api.post<BatchCreateResponse>('/tasks/batch', { tasks })
}

export async function getTask(id: string): Promise<Task> {
  return api.get<Task>(`/tasks/${id}`)
}

export async function updateTask(id: string, data: UpdateTaskRequest): Promise<Task> {
  return api.patch<Task>(`/tasks/${id}`, data)
}

export async function deleteTask(id: string): Promise<void> {
  return api.delete(`/tasks/${id}`)
}

export async function scheduleTask(id: string, data: ScheduleTaskRequest): Promise<ScheduledEvent> {
  return api.post<ScheduledEvent>(`/tasks/${id}/schedule`, data)
}

export async function logTaskTime(id: string, minutes: number): Promise<Task> {
  return api.post<Task>(`/tasks/${id}/log-time`, { minutes })
}

// Priority display metadata is centralized in lib/priority (single source of truth).
// PRIORITY_COLORS maps to the solid dot/badge variant used by the task list.
export { PRIORITY_LABELS, PRIORITY_DOT_COLORS as PRIORITY_COLORS } from '../lib/priority'

// Context icons (using emoji for now, can be replaced with Lucide icons)
export const CONTEXT_ICONS: Record<string, string> = {
  '@home': '🏠',
  '@work': '💼',
  '@computer': '💻',
  '@errands': '🛒',
  '@university': '🎓',
  '@personal': '👤',
  '@routine': '🔄',
}