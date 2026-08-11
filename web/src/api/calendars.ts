import { api } from './client'

/**
 * Calendar as the API returns it.
 *
 * Field names are snake_case because that is what the API sends — every other
 * module in this backend does the same. There is deliberately no camelCase
 * mapping layer here: defects T1 (27.07) and T7 (11.08) were both a type that
 * promised camelCase over a payload that was snake_case, and the field simply
 * arrived undefined. No mapping means nothing to get wrong.
 */
export interface Calendar {
  id: string
  name: string
  color: string | null
  kind: 'personal' | 'shared'
  role: 'owner' | 'editor' | 'viewer'
  status: 'invited' | 'active'
  created_at: string
}

export interface CalendarNotEmpty {
  code: 'CALENDAR_NOT_EMPTY'
  events: number
  tasks: number
}

export function listCalendars(): Promise<Calendar[]> {
  return api.get<Calendar[]>('/calendars')
}

export function createCalendar(name: string, color?: string): Promise<Calendar> {
  return api.post<Calendar>('/calendars', { name, color: color ?? null })
}

export function updateCalendar(
  id: string,
  patch: { name?: string; color?: string },
): Promise<Calendar> {
  return api.patch<Calendar>(`/calendars/${id}`, patch)
}

export function deleteCalendar(id: string): Promise<void> {
  return api.delete(`/calendars/${id}`)
}
