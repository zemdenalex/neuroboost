import { api } from './client'
import type { ToTaskBody } from '../lib/convert/linkFlow'

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
  /** Omitted = the author's personal calendar. */
  calendar_id?: string
  /** Omitted = the user's default preset; [] = no reminders at all. */
  reminder_offsets?: number[]
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

/** The task an event becomes, as POST /api/events/{id}/to-task words it (events/totask.go). */
export interface ToTaskResult {
  task: {
    id?: string
    title: string
    due_date: string
    estimated_minutes?: number
    rrule?: string
  }
  /** Codes: start_time, color, location, reminders. */
  lost: string[]
  event_id: string | null
  dry_run: boolean
}

/**
 * Event → task. The id may be an occurrence («<uuid>:<YYYY-MM-DD>»), which is
 * how «only this once» names its day, as the bot sends it. 🔴 Not
 * encodeURIComponent: chi matches on the raw path, and «%3A» would reach the
 * handler undecoded.
 */
export async function eventToTask(id: string, body: ToTaskBody): Promise<ToTaskResult> {
  return api.post<ToTaskResult>(`/events/${id}/to-task`, body)
}
