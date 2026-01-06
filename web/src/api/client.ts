import type { NbEvent, Task, ApiEvent } from '../types';

const API_BASE = import.meta.env?.VITE_API_URL?.replace(/\/$/, '') || '/api';

// Token management
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem('nb_token', token);
  } else {
    localStorage.removeItem('nb_token');
  }
}

export function getAuthToken(): string | null {
  if (!authToken) {
    authToken = localStorage.getItem('nb_token');
  }
  return authToken;
}

// Base request function with auth
async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  options?: { skipAuth?: boolean }
): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  
  const token = getAuthToken();
  if (token && !options?.skipAuth) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 401) {
    setAuthToken(null);
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || error.message || 'Request failed');
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

// Convert API event (snake_case) to frontend event (camelCase)
function toNbEvent(api: ApiEvent): NbEvent {
  return {
    id: api.id,
    title: api.title,
    startsAt: api.starts_at,
    endsAt: api.ends_at,
    allDay: api.all_day ?? false,
    description: api.description,
    location: api.location,
    color: api.color,
    tags: api.tags ?? [],
    timezone: api.timezone ?? 'Europe/Moscow',
    rrule: api.rrule,
    taskId: api.task_id,
    reflections: api.reflections?.map(r => ({
      id: r.id,
      focusPct: r.focus_pct,
      goalPct: r.goal_pct,
      mood: r.mood,
      note: r.note,
      wasCompleted: r.was_completed,
      wasOnTime: r.was_on_time,
    })),
  };
}

// ============ AUTH API ============

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    email?: string;
    tgId?: number;
    tgUsername?: string;
    timezone: string;
  };
}

export async function loginWithTelegram(telegramData: unknown): Promise<LoginResponse> {
  return request<LoginResponse>('POST', '/auth/telegram', telegramData, { skipAuth: true });
}

export async function loginWithEmail(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>('POST', '/auth/login', { email, password }, { skipAuth: true });
}

export async function getCurrentUser(): Promise<LoginResponse['user']> {
  const response = await request<{ user: LoginResponse['user'] }>('GET', '/auth/me');
  return response.user;
}

export async function logout(): Promise<void> {
  await request<void>('POST', '/auth/logout');
  setAuthToken(null);
}

// ============ EVENTS API ============

export async function getEvents(startISO: string, endISO: string): Promise<NbEvent[]> {
  const params = new URLSearchParams({ start: startISO, end: endISO });
  const response = await request<{ events: ApiEvent[] } | ApiEvent[]>('GET', `/events?${params}`);
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
  const response = await request<{ event: ApiEvent }>('POST', '/events', apiBody);
  return toNbEvent(response.event);
}

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
  
  const response = await request<{ event: ApiEvent }>('PATCH', `/events/${id}`, apiBody);
  return toNbEvent(response.event);
}

export async function deleteEvent(id: string): Promise<void> {
  await request<void>('DELETE', `/events/${id}`);
}

export async function moveEvent(id: string, startsAt: string, endsAt: string): Promise<NbEvent> {
  const response = await request<{ event: ApiEvent }>('PATCH', `/events/${id}/move`, {
    starts_at: startsAt,
    ends_at: endsAt,
  });
  return toNbEvent(response.event);
}

// ============ REFLECTIONS API ============

export interface ReflectionBody {
  focusPct: number;
  goalPct: number;
  mood: number;
  note?: string;
  wasCompleted?: boolean;
  wasOnTime?: boolean;
}

export async function saveReflection(eventId: string, reflection: ReflectionBody): Promise<void> {
  await request<void>('POST', `/events/${eventId}/reflection`, {
    focus_pct: reflection.focusPct,
    goal_pct: reflection.goalPct,
    mood: reflection.mood,
    note: reflection.note,
    was_completed: reflection.wasCompleted,
    was_on_time: reflection.wasOnTime,
  });
}

// ============ TASKS API ============

export async function getTasks(status?: string, priority?: number): Promise<Task[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (priority !== undefined) params.append('priority', String(priority));
  
  const response = await request<{ tasks: Task[] }>('GET', `/tasks?${params}`);
  return response.tasks || [];
}

export async function createTask(body: {
  title: string;
  description?: string;
  priority?: number;
  tags?: string[];
  dueDate?: string;
  estimatedMinutes?: number;
}): Promise<Task> {
  const response = await request<{ task: Task }>('POST', '/tasks', {
    title: body.title,
    description: body.description,
    priority: body.priority,
    tags: body.tags,
    due_date: body.dueDate,
    estimated_minutes: body.estimatedMinutes,
  });
  return response.task;
}

export async function updateTask(id: string, updates: {
  title?: string;
  status?: string;
  priority?: number;
}): Promise<Task> {
  const response = await request<{ task: Task }>('PATCH', `/tasks/${id}`, updates);
  return response.task;
}

export async function deleteTask(id: string): Promise<void> {
  await request<void>('DELETE', `/tasks/${id}`);
}

export async function scheduleTask(
  taskId: string,
  startsAt: string,
  duration?: number,
  keepTaskOpen?: boolean
): Promise<NbEvent> {
  const response = await request<{ event: ApiEvent }>('POST', `/tasks/${taskId}/schedule`, {
    starts_at: startsAt,
    duration,
    keep_task_open: keepTaskOpen,
  });
  return toNbEvent(response.event);
}

// ============ SETTINGS API ============

export interface UserSettings {
  timezone: string;
  workingHoursStart: number;
  workingHoursEnd: number;
  workingDays: number[];
}

export async function getUserSettings(): Promise<UserSettings> {
  const response = await request<{ settings: UserSettings }>('GET', '/settings');
  return response.settings;
}

export async function updateUserSettings(updates: Partial<UserSettings>): Promise<UserSettings> {
  const response = await request<{ settings: UserSettings }>('PATCH', '/settings', updates);
  return response.settings;
}

// ============ HEALTH API ============

export async function checkHealth(): Promise<{ ok: boolean; db?: string }> {
  return request<{ ok: boolean; db?: string }>('GET', '/health', undefined, { skipAuth: true });
}
