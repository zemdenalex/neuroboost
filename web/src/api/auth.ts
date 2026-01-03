import { api, setStoredToken, clearStoredToken } from './client'

// Types
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
  is_admin?: boolean
  created_at: string
  last_login_at?: string
}

export interface AuthResponse {
  token: string
  expires_at: number
  user: User
}

export interface TelegramAuthData {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}

// Auth API calls

export async function loginWithTelegram(data: TelegramAuthData): Promise<AuthResponse> {
  const response = await api.post<AuthResponse>('/auth/telegram', data, false)
  if (response.token) {
    setStoredToken(response.token, response.expires_at)
  }
  return response
}

export async function register(email: string, password: string, name?: string): Promise<AuthResponse> {
  const response = await api.post<AuthResponse>('/auth/register', { email, password, name }, false)
  if (response.token) {
    setStoredToken(response.token, response.expires_at)
  }
  return response
}

export async function loginWithEmail(email: string, password: string): Promise<AuthResponse> {
  const response = await api.post<AuthResponse>('/auth/login', { email, password }, false)
  if (response.token) {
    setStoredToken(response.token, response.expires_at)
  }
  return response
}

export async function getMe(): Promise<User> {
  return api.get<User>('/auth/me')
}

export async function logout(): Promise<void> {
  try {
    await api.post('/auth/logout', {})
  } finally {
    clearStoredToken()
  }
}
