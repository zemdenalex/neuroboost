import { api } from './client'
export async function list() { console.log('[API STUB] list tasks'); return api.get('/tasks') }
export async function create(task: any) { console.log('[API STUB] create task', task); return api.post('/tasks', task) }
export async function get(id: string) { console.log('[API STUB] get task', id); return api.get(`/tasks/${id}`) }
export async function update(id: string, task: any) { console.log('[API STUB] update task', id, task); return api.patch(`/tasks/${id}`, task) }
export async function remove(id: string) { console.log('[API STUB] delete task', id); return api.delete(`/tasks/${id}`) }
export async function schedule(id: string, startsAt: string, endsAt: string) { console.log('[API STUB] schedule task', id); return api.post(`/tasks/${id}/schedule`, { startsAt, endsAt }) }
