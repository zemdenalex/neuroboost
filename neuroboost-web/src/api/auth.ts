import { api } from './client'
export async function loginWithTelegram(initData: string) { console.log('[API STUB] loginWithTelegram', initData); return api.post('/auth/telegram', { initData }) }
export async function getMe() { console.log('[API STUB] getMe'); return api.get('/auth/me') }
export async function logout() { console.log('[API STUB] logout'); return api.post('/auth/logout', {}) }
