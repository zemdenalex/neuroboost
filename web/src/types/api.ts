export interface ApiResponse<T> { data: T }
export interface CreateEventDTO { title: string; startsAt: string; endsAt: string; allDay?: boolean }
export interface UpdateEventDTO extends Partial<CreateEventDTO> {}
export interface CreateTaskDTO { title: string; priority?: number }
export interface UpdateTaskDTO extends Partial<CreateTaskDTO> {}
