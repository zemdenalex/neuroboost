/**
 * THE camelCase TASK STACK — used by the calendar side of the app.
 *
 * There are two task stacks, and this is the boundary between them:
 *
 *   api/tasks.ts   snake_case on the wire AND in its types. The complete one:
 *                  list, create, batch, get, update, delete, schedule, logTime.
 *                  Used by Tasks, Planning, QuickAdd.
 *   api/index.ts   this file. camelCase wrappers over the same endpoints, for
 *                  consumers that speak the calendar's camelCase `Task` type
 *                  (types/index.ts) — Calendar and Dashboard.
 *
 * 🔴 Every wrapper here CONVERTS through toTask/toNbEvent. It must stay that
 * way. Asserting a camelCase type onto a snake_case response is what produced
 * T1 ("a dragged task is always 60 minutes") and, until 2026-08-14, made
 * createTask and updateTask return undefined under a type that promised a Task.
 *
 * ⚠ Merging the two stacks is deliberately NOT done. It would mean converting
 * the calendar, WeekGrid and TaskSidebar from the camelCase Task type to the
 * snake_case one — a large change across code with no regression suite. The
 * duplication that remains is of NAMES, not of behaviour; the correctness
 * problem it used to hide is fixed and covered by api/taskWrappers.test.ts.
 */
import { toTask, type RawTask } from './toTask';
import { scopeQuery, type MutationScope } from '../lib/recurrence/scope';

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

// Auth, feedback & admin
export * from './auth';
export * from './feedback';
export * from './admin';

// Reflections
export * from './reflections';

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
  const events = Array.isArray(response) ? response : (response?.events ?? []);
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
  timezone?: string;
  rrule?: string;
  /**
   * Minutes before start, one per reminder. Omitting the field means "apply my
   * default preset" on the backend; an explicit [] means "no reminders".
   */
  reminderOffsets?: number[];
  /**
   * Which calendar to create in. Omitted means the author's personal one.
   *
   * The backend verifies membership — a supplied id is never trusted — and
   * answers 404 for a calendar the caller cannot see, 403 for one they can read
   * but not write.
   */
  calendarId?: string;
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
    timezone: body.timezone,
    rrule: body.rrule,
    reminder_offsets: body.reminderOffsets,
    calendar_id: body.calendarId,
  };
  const response = await api.post<{ event: ApiEvent } | ApiEvent>('/events', apiBody);
  const event = 'event' in (response as Record<string, unknown>)
    ? (response as { event: ApiEvent }).event
    : response as ApiEvent;
  return toNbEvent(event);
}

/** Update event from camelCase body, return NbEvent */
export async function updateEvent(id: string, updates: Partial<CreateEventBody>, scope?: MutationScope): Promise<NbEvent> {
  const apiBody: Record<string, unknown> = {};
  if (updates.title !== undefined) apiBody.title = updates.title;
  if (updates.startsAt !== undefined) apiBody.starts_at = updates.startsAt;
  if (updates.endsAt !== undefined) apiBody.ends_at = updates.endsAt;
  if (updates.allDay !== undefined) apiBody.all_day = updates.allDay;
  if (updates.description !== undefined) apiBody.description = updates.description;
  if (updates.location !== undefined) apiBody.location = updates.location;
  if (updates.tags !== undefined) apiBody.tags = updates.tags;
  if (updates.color !== undefined) apiBody.color = updates.color;
  if (updates.timezone !== undefined) apiBody.timezone = updates.timezone;
  if (updates.rrule !== undefined) apiBody.rrule = updates.rrule;
  // An explicit [] must survive: it means "no reminders", which is different
  // from omitting the key (leave whatever is stored alone).
  if (updates.reminderOffsets !== undefined) apiBody.reminder_offsets = updates.reminderOffsets;
  // Moves the event to another calendar. The API checks that the caller may
  // write there and refuses this together with ?scope=occurrence — a calendar
  // change is a property of the series, not of one Tuesday.
  if (updates.calendarId !== undefined) apiBody.calendar_id = updates.calendarId;

  const response = await api.patch<{ event: ApiEvent } | ApiEvent>(`/events/${id}${scopeQuery(scope)}`, apiBody);
  const event = 'event' in (response as Record<string, unknown>)
    ? (response as { event: ApiEvent }).event
    : response as ApiEvent;
  return toNbEvent(event);
}

/** Delete an event */
export async function deleteEvent(id: string, scope?: MutationScope): Promise<void> {
  await api.delete(`/events/${id}${scopeQuery(scope)}`);
}

/** Move event by camelCase times */
export async function moveEvent(id: string, startsAt: string, endsAt: string, scope?: MutationScope): Promise<NbEvent> {
  return updateEvent(id, { startsAt, endsAt }, scope);
}

export interface ReflectionBody {
  focus: number;
  energy: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

/** Save reflection with camelCase body */
export async function saveReflection(eventId: string, reflection: ReflectionBody): Promise<void> {
  await api.post(`/events/${eventId}/reflection`, {
    focus: reflection.focus,
    energy: reflection.energy,
    mood: reflection.mood,
    notes: reflection.note,
    was_completed: reflection.wasCompleted,
    was_on_time: reflection.wasOnTime,
  });
}

/** Fetch tasks with optional filters */
export async function getTasks(status?: string, priority?: number): Promise<import('../types').Task[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (priority !== undefined) params.append('priority', String(priority));

  const response = await api.get<{ tasks: RawTask[] } | RawTask[]>(`/tasks?${params}`);
  const rows = Array.isArray(response) ? response : response.tasks || [];
  return rows.map(toTask);
}

/** Create task with camelCase body */
export async function createTask(body: {
  title: string;
  description?: string;
  priority?: number;
  tags?: string[];
  dueDate?: string;
  estimatedMinutes?: number;
  /**
   * Which calendar the task belongs to. Omitted means "my personal calendar",
   * which is the API's own default — and was the ONLY behaviour available here
   * until 23.08, because this wrapper had no way to say anything else. A task
   * created from a shared week landed out of sight of the person it was for.
   */
  calendarId?: string;
}): Promise<import('../types').Task> {
  // 🔴 The API returns the task object itself inside the { data } envelope,
  // and api.post already unwraps that envelope — so asking for a `.task` key
  // returned undefined under a type that promised a Task. Every current caller
  // throws the value away, which is the only reason nobody saw it.
  //
  // This is the T1 mechanism exactly: a type asserted onto a response nobody
  // checked. T1 cost "a dragged task is always 60 minutes"; this one was
  // waiting for the first caller to use the result.
  const raw = await api.post<RawTask>('/tasks', {
    title: body.title,
    description: body.description,
    priority: body.priority,
    tags: body.tags,
    due_date: body.dueDate,
    estimated_minutes: body.estimatedMinutes,
    // Omitted rather than sent empty, matching the event editor: the backend
    // reads an absent field as "the author's personal calendar".
    calendar_id: body.calendarId || undefined,
  });
  return toTask(raw);
}

/** Update task */
export async function updateTask(id: string, updates: {
  title?: string;
  status?: string;
  priority?: number;
}): Promise<import('../types').Task> {
  // Same defect as createTask above, same fix.
  const raw = await api.patch<RawTask>(`/tasks/${id}`, updates);
  return toTask(raw);
}

/** Delete a task */
export async function deleteTask(id: string): Promise<void> {
  await api.delete(`/tasks/${id}`);
}

/** Schedule task to calendar, creating an event */
/**
 * Schedule a task by start time and duration, returning the created event in
 * the calendar's camelCase shape.
 *
 * 🔴 Named `scheduleTaskAt` since 2026-08-14. It was `scheduleTask`, which is
 * also the name of a DIFFERENT function in api/tasks.ts — same identifier, two
 * modules, incompatible arity (`(id, startsAt, minutes)` here against
 * `(id, { ... })` there) and different return types. Both are in active use:
 * Calendar and Tasks call this one, Planning calls the other. Whichever a
 * reader had in mind, the wrong import compiled or failed for reasons that had
 * nothing to do with what they were doing.
 *
 * The name says what distinguishes it: this one takes a moment in time.
 */
export async function scheduleTaskAt(taskId: string, startsAt: string, durationMinutes = 60): Promise<NbEvent> {
  const startMs = Date.parse(startsAt);
  const mins = Number.isFinite(durationMinutes) ? Math.max(15, durationMinutes) : 60;
  const endsAt = new Date(startMs + mins * 60_000).toISOString();

  const response = await api.post<ApiEvent>(`/tasks/${taskId}/schedule`, {
    starts_at: startsAt,
    ends_at: endsAt,
  });

  return toNbEvent(response);
}
