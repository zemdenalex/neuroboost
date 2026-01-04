import { api } from './client'

export interface Event {
  id: string
  user_id: string
  title: string
  description?: string
  starts_at: string
  ends_at: string
  all_day: boolean
  rrule?: string
  timezone: string
  location?: string
  color?: string
  tags: string[]
  task_id?: string
  is_work_event: boolean
  created_at: string
  updated_at: string
}

export interface CreateEventRequest {
  title: string
  description?: string
  starts_at: string
  ends_at: string
  all_day?: boolean
  rrule?: string
  timezone?: string
  location?: string
  color?: string
  tags?: string[]
  task_id?: string
  is_work_event?: boolean
}

export interface UpdateEventRequest {
  title?: string
  description?: string
  starts_at?: string
  ends_at?: string
  all_day?: boolean
  rrule?: string
  timezone?: string
  location?: string
  color?: string
  tags?: string[]
  is_work_event?: boolean
}

export interface MoveEventRequest {
  starts_at: string
  ends_at: string
}

export interface ResizeEventRequest {
  ends_at: string
}

export async function listEvents(start: string, end: string): Promise<Event[]> {
  return api.get<Event[]>(`/events?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`)
}

export async function createEvent(data: CreateEventRequest): Promise<Event> {
  return api.post<Event>('/events', data)
}

export async function getEvent(id: string): Promise<Event> {
  return api.get<Event>(`/events/${id}`)
}

export async function updateEvent(id: string, data: UpdateEventRequest): Promise<Event> {
  return api.patch<Event>(`/events/${id}`, data)
}

export async function deleteEvent(id: string): Promise<void> {
  return api.delete(`/events/${id}`)
}

export async function moveEvent(id: string, data: MoveEventRequest): Promise<Event> {
  return api.patch<Event>(`/events/${id}/move`, data)
}

export async function resizeEvent(id: string, data: ResizeEventRequest): Promise<Event> {
  return api.patch<Event>(`/events/${id}/resize`, data)
}

// Convert API event to internal NbEvent format
export function toNbEvent(event: Event): NbEvent {
  return {
    id: event.id,
    title: event.title,
    description: event.description,
    startUtc: event.starts_at,
    endUtc: event.ends_at,
    allDay: event.all_day,
    rrule: event.rrule,
    color: event.color,
    tags: event.tags,
    taskId: event.task_id,
  }
}

// Internal event format used by WeekGrid
export interface NbEvent {
  id: string
  title: string
  description?: string
  startUtc: string
  endUtc: string
  allDay: boolean
  rrule?: string
  color?: string
  tags: string[]
  taskId?: string
}