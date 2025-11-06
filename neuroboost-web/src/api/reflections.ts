import { api } from './client'
export async function list(eventId?: string) { console.log('[API STUB] list reflections', eventId); const q = eventId ? `?eventId=${eventId}`: '' ; return api.get(`/reflections${q}`) }
export async function create(reflection: any) { console.log('[API STUB] create reflection', reflection); return api.post('/reflections', reflection) }
export async function update(id: string, reflection: any) { console.log('[API STUB] update reflection', id, reflection); return api.patch(`/reflections/${id}`, reflection) }
