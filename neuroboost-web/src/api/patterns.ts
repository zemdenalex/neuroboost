import { api } from './client'
export async function getMetrics(userId: string) { console.log('[API STUB] get metrics', userId); return api.get(`/patterns/metrics?userId=${userId}`) }
export async function getAlertStatus(userId: string) { console.log('[API STUB] get alert status', userId); return api.get(`/patterns/alert-status?userId=${userId}`) }
