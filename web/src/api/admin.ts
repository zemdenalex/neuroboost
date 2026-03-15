import { api } from './client'

export interface HealthResponse {
  version: string
  uptime_seconds: number
  go_version: string
  db_status: string
  db_ping_ms: number
  migration_version: number
  memory_mb: number
  goroutines: number
}

export interface LogEntry {
  timestamp: string
  level: string
  message: string
  fields?: Record<string, string>
}

export async function getAdminHealth(): Promise<HealthResponse> {
  return api.get<HealthResponse>('/admin/health')
}

export async function getAdminLogs(params?: {
  level?: string
  limit?: number
}): Promise<LogEntry[]> {
  const searchParams = new URLSearchParams()
  if (params?.level) searchParams.set('level', params.level)
  if (params?.limit) searchParams.set('limit', String(params.limit))
  const query = searchParams.toString()
  return api.get<LogEntry[]>(`/admin/logs${query ? '?' + query : ''}`)
}
