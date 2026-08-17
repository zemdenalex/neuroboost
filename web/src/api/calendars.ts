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

/** Someone on a calendar. `status` tells an accepted member from an invitee. */
export interface CalendarMember {
  user_id: string
  email: string | null
  display_name: string | null
  role: 'owner' | 'editor' | 'viewer'
  status: 'invited' | 'active'
}

/** A share link. Two hours, one use — see the API's members.go. */
export interface CalendarInvite {
  token: string
  role: 'editor' | 'viewer'
  expires_at: string
}

/** Roles that can be granted to someone else. Owner is not one of them. */
export type ShareRole = 'editor' | 'viewer'

export function listMembers(calendarId: string): Promise<CalendarMember[]> {
  return api.get<CalendarMember[]>(`/calendars/${calendarId}/members`)
}

export function inviteByEmail(
  calendarId: string,
  email: string,
  role: ShareRole,
): Promise<CalendarMember> {
  return api.post<CalendarMember>(`/calendars/${calendarId}/invites`, { email, role })
}

export function createInviteLink(calendarId: string, role: ShareRole): Promise<CalendarInvite> {
  return api.post<CalendarInvite>(`/calendars/${calendarId}/invite-links`, { role })
}

/** Accept or decline an invitation addressed to me. */
export function respondToInvitation(calendarId: string, accept: boolean): Promise<void> {
  return api.post(`/calendars/${calendarId}/invitation`, { accept })
}

/**
 * Redeem a share link.
 *
 * The token goes in the BODY even though the page received it in a URL — a
 * token in a request path lands in the API's access logs, and this one grants
 * access to someone's calendar.
 */
export function acceptInviteLink(token: string): Promise<Calendar> {
  return api.post<Calendar>('/calendars/invite-links/accept', { token })
}

/** Remove someone from a calendar, or leave it yourself (same id twice). */
export function removeMember(calendarId: string, userId: string): Promise<void> {
  return api.delete(`/calendars/${calendarId}/members/${userId}`)
}
