export interface ApiResponse<T> { data: T }
export interface CreateEventDTO { title: string; startsAt: string; endsAt: string; allDay?: boolean }
export type UpdateEventDTO = Partial<CreateEventDTO>
export interface CreateTaskDTO { title: string; priority?: number }
export type UpdateTaskDTO = Partial<CreateTaskDTO>
