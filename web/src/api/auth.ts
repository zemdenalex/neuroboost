import { api } from './client'

export interface User {
  id: string
  email?: string
  tg_id?: number
  tg_username?: string
  tg_first_name?: string
  tg_last_name?: string
  tg_photo_url?: string
  display_name?: string
  timezone: string
  locale: string
  is_admin: boolean
  settings?: UserSettings
  created_at: string
  last_login_at?: string
}

export interface UserSettings {
  header_variant?: 'horizontal' | 'vertical'
  mobile_nav?: 'bottom_tabs' | 'hamburger' | 'fab'
  ui_scale?: number
  work_days?: string[]
  work_start?: string
  work_end?: string
  features?: Record<string, boolean>
  quiet_hours_start?: string
  quiet_hours_end?: string
  quick_task?: {
    default_due?: 'today' | 'tomorrow' | 'none'
    default_priority?: number
    default_estimate_minutes?: number | null
    inherit_filters?: boolean
    keys?: Partial<Record<'submit' | 'submit_expanded' | 'expand' | 'global_capture' | 'indent' | 'outdent', string>>
  }
}

export interface AuthResponse {
  token: string
  expires_at: number
  user: User
}

export interface TelegramUser {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}

export interface RegisterRequest {
  email: string
  password: string
  name?: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface UpdateUserRequest {
  display_name?: string
  timezone?: string
  locale?: string
  settings?: UserSettings
}

export async function telegramLogin(telegramUser: TelegramUser): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/telegram', telegramUser)
}

export async function register(data: RegisterRequest): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/register', data)
}

export async function login(data: LoginRequest): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/login', data)
}

export async function logout(): Promise<void> {
  return api.post('/auth/logout', {})
}

export async function getMe(): Promise<User> {
  return api.get<User>('/auth/me')
}

export async function updateMe(data: UpdateUserRequest): Promise<User> {
  return api.patch<User>('/auth/me', data)
}
