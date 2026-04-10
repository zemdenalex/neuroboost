import { api } from './client'

// snake_case matches the Go backend JSON tags
export interface ApiReflection {
  id: string
  user_id: string
  event_id: string
  energy?: number | null
  mood?: number | null
  focus?: number | null
  notes?: string | null
  was_completed: boolean
  was_on_time: boolean
  created_at: string
}

// camelCase version for frontend use
export interface Reflection {
  id: string
  userId: string
  eventId: string
  energy: number | null
  mood: number | null
  focus: number | null
  notes: string | null
  wasCompleted: boolean
  wasOnTime: boolean
  createdAt: string
}

function toReflection(r: ApiReflection): Reflection {
  return {
    id: r.id,
    userId: r.user_id,
    eventId: r.event_id,
    energy: r.energy ?? null,
    mood: r.mood ?? null,
    focus: r.focus ?? null,
    notes: r.notes ?? null,
    wasCompleted: r.was_completed,
    wasOnTime: r.was_on_time,
    createdAt: r.created_at,
  }
}

export async function getReflections(eventId?: string): Promise<Reflection[]> {
  const params = eventId ? `?event_id=${encodeURIComponent(eventId)}` : ''
  const raw = await api.get<ApiReflection[]>(`/reflections${params}`)
  return (raw ?? []).map(toReflection)
}

export interface CreateReflectionBody {
  eventId: string
  energy?: number
  mood?: number
  focus?: number
  notes?: string
  wasCompleted?: boolean
  wasOnTime?: boolean
}

export async function createReflection(body: CreateReflectionBody): Promise<Reflection> {
  const raw = await api.post<ApiReflection>('/reflections', {
    event_id: body.eventId,
    energy: body.energy,
    mood: body.mood,
    focus: body.focus,
    notes: body.notes,
    was_completed: body.wasCompleted,
    was_on_time: body.wasOnTime,
  })
  return toReflection(raw)
}

export interface UpdateReflectionBody {
  energy?: number
  mood?: number
  focus?: number
  notes?: string
  wasCompleted?: boolean
  wasOnTime?: boolean
}

export async function updateReflection(id: string, body: UpdateReflectionBody): Promise<Reflection> {
  const raw = await api.patch<ApiReflection>(`/reflections/${id}`, {
    energy: body.energy,
    mood: body.mood,
    focus: body.focus,
    notes: body.notes,
    was_completed: body.wasCompleted,
    was_on_time: body.wasOnTime,
  })
  return toReflection(raw)
}
