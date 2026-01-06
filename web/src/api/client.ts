import type { NbEvent, ApiEvent } from '../types';

const API_BASE = (import.meta.env?.VITE_API_URL ?? '/api').replace(/\/$/, '');

// Token storage keys
const TOKEN_KEY = 'nb_token';
const TOKEN_EXPIRY_KEY = 'nb_token_expiry';

// Token management functions
export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string, expiresAt?: number): void {
  localStorage.setItem(TOKEN_KEY, token);
  if (expiresAt) {
    localStorage.setItem(TOKEN_EXPIRY_KEY, String(expiresAt));
  }
}

export function clearStoredToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(TOKEN_EXPIRY_KEY);
}

export function isTokenExpired(): boolean {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (!expiry) return false;
  return Date.now() > Number(expiry) * 1000;
}

export function getTokenDaysRemaining(): number {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (!expiry) return 0;
  const remaining = Number(expiry) * 1000 - Date.now();
  return Math.max(0, Math.floor(remaining / (24 * 60 * 60 * 1000)));
}

// Legacy aliases for backward compatibility
export function setAuthToken(token: string | null): void {
  if (token) {
    setStoredToken(token);
  } else {
    clearStoredToken();
  }
}

export function getAuthToken(): string | null {
  return getStoredToken();
}

// Base request function with auth
async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  requireAuth = true
): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  const token = getStoredToken();
  if (token && requireAuth) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 401) {
    clearStoredToken();
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

// API object for use by other modules
export const api = {
  get<T>(path: string, requireAuth = true): Promise<T> {
    return request<T>('GET', path, undefined, requireAuth);
  },

  post<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('POST', path, body, requireAuth);
  },

  patch<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('PATCH', path, body, requireAuth);
  },

  put<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('PUT', path, body, requireAuth);
  },

  delete<T = void>(path: string, requireAuth = true): Promise<T> {
    return request<T>('DELETE', path, undefined, requireAuth);
  },
};

// Convert API event (snake_case) to frontend event (camelCase)
function toNbEvent(apiEvent: ApiEvent): NbEvent {
  return {
    id: apiEvent.id,
    title: apiEvent.title,
    startsAt: apiEvent.starts_at,
    endsAt: apiEvent.ends_at,
    allDay: apiEvent.all_day ?? false,
    description: apiEvent.description,
    location: apiEvent.location,
    color: apiEvent.color,
    tags: apiEvent.tags ?? [],
    timezone: apiEvent.timezone ?? 'Europe/Moscow',
    rrule: apiEvent.rrule,
    taskId: apiEvent.task_id,
    reflections: apiEvent.reflections?.map(r => ({
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

// ============ EVENTS API ============

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
  const response = await api.post<{ event: ApiEvent }>('/events', apiBody);
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

  const response = await api.patch<{ event: ApiEvent }>(`/events/${id}`, apiBody);
  return toNbEvent(response.event);
}

export async function deleteEvent(id: string): Promise<void> {
  await api.delete(`/events/${id}`);
}

export async function moveEvent(id: string, startsAt: string, endsAt: string): Promise<NbEvent> {
  const response = await api.patch<{ event: ApiEvent }>(`/events/${id}/move`, {
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
  await api.post(`/events/${eventId}/reflection`, {
    focus_pct: reflection.focusPct,
    goal_pct: reflection.goalPct,
    mood: reflection.mood,
    note: reflection.note,
    was_completed: reflection.wasCompleted,
    was_on_time: reflection.wasOnTime,
  });
}

// ============ TASKS API ============

export async function getTasks(status?: string, priority?: number): Promise<import('../types').Task[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (priority !== undefined) params.append('priority', String(priority));

  const response = await api.get<{ tasks: import('../types').Task[] } | import('../types').Task[]>(`/tasks?${params}`);
  return Array.isArray(response) ? response : response.tasks || [];
}

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

export async function updateTask(id: string, updates: {
  title?: string;
  status?: string;
  priority?: number;
}): Promise<import('../types').Task> {
  const response = await api.patch<{ task: import('../types').Task }>(`/tasks/${id}`, updates);
  return response.task;
}

export async function deleteTask(id: string): Promise<void> {
  await api.delete(`/tasks/${id}`);
}

export async function scheduleTask(
  taskId: string,
  startsAt: string,
  duration?: number,
  keepTaskOpen?: boolean
): Promise<NbEvent> {
  const response = await api.post<{ event: ApiEvent }>(`/tasks/${taskId}/schedule`, {
    starts_at: startsAt,
    duration,
    keep_task_open: keepTaskOpen,
  });
  return toNbEvent(response.event);
}

// ============ HEALTH API ============

export async function checkHealth(): Promise<{ ok: boolean; db?: string }> {
  return api.get<{ ok: boolean; db?: string }>('/health', false);
}
