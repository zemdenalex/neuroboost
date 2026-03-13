// HTTP client & token management
export {
  api,
  getStoredToken,
  setStoredToken,
  clearStoredToken,
  isTokenExpired,
  getTokenDaysRemaining,
  setAuthToken,
  getAuthToken,
  checkHealth,
} from './client';

// Raw event/task modules (snake_case types & low-level functions)
// Used by hooks and pages that work with raw API types
export type {
  Event,
  CreateEventRequest,
  UpdateEventRequest,
  MoveEventRequest,
  ResizeEventRequest,
} from './events';
export {
  listEvents,
  getEvent,
  resizeEvent,
} from './events';

export type {
  Task as RawTask,
  TaskStatus,
  TaskCategory,
  CreateTaskRequest,
  UpdateTaskRequest,
  ScheduleTaskRequest,
  ScheduledEvent,
  ListTasksQuery,
} from './tasks';
export {
  listTasks,
  getTask,
  PRIORITY_LABELS,
  PRIORITY_COLORS,
  CONTEXT_ICONS,
} from './tasks';

// Auth & feedback
export * from './auth';
export * from './feedback';

// ============================================================
// camelCase wrappers for Calendar & EventEditor consumers
// Accept camelCase params, return NbEvent/Task (camelCase)
// ============================================================

import type { NbEvent, ApiEvent } from '../types';
import { toNbEvent } from '../types';
import { api } from './client';

/** Fetch events for a date range, returned as NbEvent[] */
export async function getEvents(startISO: string, endISO: string): Promise<NbEvent[]> {
  const params = new URLSearchParams({ start: startISO, end: endISO });
  const response = await api.get<{ events: ApiEvent[] } | ApiEvent[]>(`/events?${params}`);
  const events = Array.isArray(response) ? response : response.events || [];
  return events.map(toNbEvent);
}

export interface CreateEventBody {
  title: string;
  startsAt: string;
  endsAt: string;
  allDay?: boolean;
  description?: string;
  location?: string;
  tags?: string[];
  color?: string;
  reminders?: Array<{ minutesBefore: number; channel: 'TELEGRAM' | 'WEB' }>;
}

/** Create event from camelCase body, return NbEvent */
export async function createEvent(body: CreateEventBody): Promise<NbEvent> {
  const apiBody = {
    title: body.title,
    starts_at: body.startsAt,
    ends_at: body.endsAt,
    all_day: body.allDay,
    description: body.description,
    location: body.location,
    tags: body.tags,
    color: body.color,
    reminders: body.reminders,
  };
  const response = await api.post<{ event: ApiEvent } | ApiEvent>('/events', apiBody);
  const event = 'event' in (response as Record<string, unknown>)
    ? (response as { event: ApiEvent }).event
    : response as ApiEvent;
  return toNbEvent(event);
}

/** Update event from camelCase body, return NbEvent */
export async function updateEvent(id: string, updates: Partial<CreateEventBody>): Promise<NbEvent> {
  const apiBody: Record<string, unknown> = {};
  if (updates.title !== undefined) apiBody.title = updates.title;
  if (updates.startsAt !== undefined) apiBody.starts_at = updates.startsAt;
  if (updates.endsAt !== undefined) apiBody.ends_at = updates.endsAt;
  if (updates.allDay !== undefined) apiBody.all_day = updates.allDay;
  if (updates.description !== undefined) apiBody.description = updates.description;
  if (updates.location !== undefined) apiBody.location = updates.location;
  if (updates.tags !== undefined) apiBody.tags = updates.tags;
  if (updates.color !== undefined) apiBody.color = updates.color;

  const response = await api.patch<{ event: ApiEvent } | ApiEvent>(`/events/${id}`, apiBody);
  const event = 'event' in (response as Record<string, unknown>)
    ? (response as { event: ApiEvent }).event
    : response as ApiEvent;
  return toNbEvent(event);
}

/** Delete an event */
export async function deleteEvent(id: string): Promise<void> {
  await api.delete(`/events/${id}`);
}

/** Move event by camelCase times */
export async function moveEvent(id: string, startsAt: string, endsAt: string): Promise<NbEvent> {
  return updateEvent(id, { startsAt, endsAt });
}

export interface ReflectionBody {
  focusPct: number;
  goalPct: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

/** Save reflection with camelCase body */
export async function saveReflection(eventId: string, reflection: ReflectionBody): Promise<void> {
  await api.post(`/events/${eventId}/reflection`, {
    focus_pct: reflection.focusPct,
    goal_pct: reflection.goalPct,
    mood: reflection.mood,
    note: reflection.note,
    was_completed: reflection.wasCompleted,
    was_on_time: reflection.wasOnTime,
  });
}

/** Fetch tasks with optional filters */
export async function getTasks(status?: string, priority?: number): Promise<import('../types').Task[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (priority !== undefined) params.append('priority', String(priority));

  const response = await api.get<{ tasks: import('../types').Task[] } | import('../types').Task[]>(`/tasks?${params}`);
  return Array.isArray(response) ? response : response.tasks || [];
}

/** Create task with camelCase body */
export async function createTask(body: {
  title: string;
  description?: string;
  priority?: number;
  tags?: string[];
  dueDate?: string;
  estimatedMinutes?: number;
}): Promise<import('../types').Task> {
  const response = await api.post<{ task: import('../types').Task }>('/tasks', {
    title: body.title,
    description: body.description,
    priority: body.priority,
    tags: body.tags,
    due_date: body.dueDate,
    estimated_minutes: body.estimatedMinutes,
  });
  return response.task;
}

/** Update task */
export async function updateTask(id: string, updates: {
  title?: string;
  status?: string;
  priority?: number;
}): Promise<import('../types').Task> {
  const response = await api.patch<{ task: import('../types').Task }>(`/tasks/${id}`, updates);
  return response.task;
}

/** Delete a task */
export async function deleteTask(id: string): Promise<void> {
  await api.delete(`/tasks/${id}`);
}

/** Schedule task to calendar, creating an event */
export async function scheduleTask(taskId: string, startsAt: string, durationMinutes = 60): Promise<NbEvent> {
  const startMs = Date.parse(startsAt);
  const mins = Number.isFinite(durationMinutes) ? Math.max(15, durationMinutes) : 60;
  const endsAt = new Date(startMs + mins * 60_000).toISOString();

  const response = await api.post<ApiEvent>(`/tasks/${taskId}/schedule`, {
    starts_at: startsAt,
    ends_at: endsAt,
  });

  return toNbEvent(response);
}
