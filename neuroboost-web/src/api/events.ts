import { api } from './client'
export async function list(from: string, to: string) { console.log('[API STUB] list events', { from, to }); return api.get(`/events?from=${from}&to=${to}`) }
export async function create(event: any) { console.log('[API STUB] create event', event); return api.post('/events', event) }
export async function get(id: string) { console.log('[API STUB] get event', id); return api.get(`/events/${id}`) }
export async function update(id: string, event: any) { console.log('[API STUB] update event', id, event); return api.patch(`/events/${id}`, event) }
export async function remove(id: string) { console.log('[API STUB] delete event', id); return api.delete(`/events/${id}`) }
export async function move(id: string, startsAt: string, endsAt: string) { console.log('[API STUB] move event', id); return api.patch(`/events/${id}/move`, { startsAt, endsAt }) }
export async function resize(id: string, endsAt: string) { console.log('[API STUB] resize event', id); return api.patch(`/events/${id}/resize`, { endsAt }) }
export async function addException(id: string, occurrence: string, skipped: boolean, replacementEventId?: string|null) { console.log('[API STUB] add exception', id); return api.post(`/events/${id}/exceptions`, { occurrence, skipped, replacementEventId: replacementEventId ?? null }) }
