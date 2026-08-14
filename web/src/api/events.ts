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


export async function deleteEvent(id: string): Promise<void> {
  return api.delete(`/events/${id}`)
}


