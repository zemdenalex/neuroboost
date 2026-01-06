// Task types
export interface Task {
  id: string;
  userId?: string;
  title: string;
  description?: string;
  priority: number; // 0-5: Buffer, Emergency, ASAP, Must today, Deadline soon, If possible
  status: 'TODO' | 'IN_PROGRESS' | 'DONE' | 'CANCELLED' | 'SCHEDULED';
  tags: string[];
  dueDate?: string;
  estimatedMinutes?: number;
  contexts?: string[];
  energy?: number;
  parentId?: string;
  parent?: Task | null;
  subtasks?: Task[];
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

// Event types (matching backend API)
export interface ApiEvent {
  id: string;
  user_id?: string;
  title: string;
  description?: string;
  starts_at: string;
  ends_at: string;
  all_day?: boolean;
  rrule?: string | null;
  timezone?: string;
  location?: string;
  color?: string;
  tags?: string[];
  task_id?: string | null;
  is_work_event?: boolean;
  created_at?: string;
  updated_at?: string;
}

// Frontend event type (camelCase for convenience)
export interface NbEvent {
  id: string;
  userId?: string;
  title: string;
  description?: string;
  startsAt: string;
  endsAt: string;
  allDay?: boolean;
  rrule?: string | null;
  timezone?: string;
  location?: string;
  color?: string;
  tags?: string[];
  taskId?: string | null;
  isWorkEvent?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

// Convert API response to frontend format
export function toNbEvent(api: ApiEvent): NbEvent {
  return {
    id: api.id,
    userId: api.user_id,
    title: api.title,
    description: api.description,
    startsAt: api.starts_at,
    endsAt: api.ends_at,
    allDay: api.all_day,
    rrule: api.rrule,
    timezone: api.timezone,
    location: api.location,
    color: api.color,
    tags: api.tags || [],
    taskId: api.task_id,
    isWorkEvent: api.is_work_event,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
  };
}

// Convert frontend format to API request
export function toApiEvent(event: Partial<NbEvent>): Partial<ApiEvent> {
  const result: Partial<ApiEvent> = {};
  
  if (event.title !== undefined) result.title = event.title;
  if (event.description !== undefined) result.description = event.description;
  if (event.startsAt !== undefined) result.starts_at = event.startsAt;
  if (event.endsAt !== undefined) result.ends_at = event.endsAt;
  if (event.allDay !== undefined) result.all_day = event.allDay;
  if (event.rrule !== undefined) result.rrule = event.rrule;
  if (event.timezone !== undefined) result.timezone = event.timezone;
  if (event.location !== undefined) result.location = event.location;
  if (event.color !== undefined) result.color = event.color;
  if (event.tags !== undefined) result.tags = event.tags;
  if (event.taskId !== undefined) result.task_id = event.taskId;
  if (event.isWorkEvent !== undefined) result.is_work_event = event.isWorkEvent;
  
  return result;
}

// Reflection types
export interface Reflection {
  id: string;
  eventId: string;
  focusPct: number;
  goalPct: number;
  mood: number;
  energy?: number;
  notes?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
  createdAt: string;
}

// Priority system
export const TASK_PRIORITIES = {
  0: { name: 'Buffer', description: 'Low-priority fill tasks', color: 'zinc' },
  1: { name: 'Emergency', description: 'Override everything', color: 'red' },
  2: { name: 'ASAP', description: 'Urgent, high-impact', color: 'orange' },
  3: { name: 'Must Today', description: 'Default priority', color: 'yellow' },
  4: { name: 'Deadline Soon', description: 'Important but not urgent', color: 'green' },
  5: { name: 'If Possible', description: 'Nice to have', color: 'blue' },
} as const;

export type TaskPriority = keyof typeof TASK_PRIORITIES;

export function getPriorityInfo(priority: number) {
  return TASK_PRIORITIES[priority as TaskPriority] || TASK_PRIORITIES[3];
}
