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
  /**
   * Which scope to use when editing a recurring event, or 'ask' to prompt.
   *
   * Declared here on 2026-08-14. The settings blob is free-form JSONB and the
   * Go side has no field for it, so the frontend was writing and reading it
   * through casts on BOTH sides — `as Partial<UserSettings>` when saving and a
   * `Record<string, unknown>` widening when reading. The value was real and
   * only the type system was unaware; naming it removes both casts.
   */
  recurring_scope?: 'ask' | 'occurrence' | 'series'
  /**
   * Day tasks (spec 2026-09-22 §11). Top-level, written by the bot and the
   * web alike. No key = on, 5 a day, days before the start not coloured.
   */
  day_tasks_enabled?: boolean
  day_tasks_target?: number
  day_tasks_paint_before?: boolean
  /**
   * Web month view variant (spec V003-20260924-arc-web-month-view): list,
   * classic, heat, split or commit. Unknown or missing = list.
   */
  month_view_variant?: string
  /** Month view: how long a click waits for a second click, ms (150–800, default 300). */
  month_click_wait_ms?: number
  quiet_hours_start?: string
  quiet_hours_end?: string
  quick_task?: {
    default_due?: 'today' | 'tomorrow' | 'none'
    default_priority?: number
    default_estimate_minutes?: number | null
    inherit_filters?: boolean
    keys?: Partial<Record<'submit' | 'submit_expanded' | 'expand' | 'global_capture' | 'indent' | 'outdent', string>>
  }
  /**
   * Mirrors the Go `reminders` settings section. quiet_hours_start/end above
   * finally do something now: a reminder landing in that window is delayed to
   * the end of it, unless it has 15 minutes' notice or less, in which case it
   * is skipped — delaying that one would deliver it after the event.
   */
  reminders?: {
    presets?: Record<string, number[]>
    default_event_preset?: string
    default_task_preset?: string
    digest_at?: string
    digest_enabled?: boolean
    quiet_hours_respected?: boolean
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

/** Sign-in from inside the Telegram Mini App: the raw initData string, untouched. */
export async function telegramWebAppLogin(initData: string): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/telegram-webapp', { init_data: initData })
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

/**
 * Account linking, v0.4.11.5.
 *
 * Denis, 17.09: the Telegram login widget is bound to the production domain, so
 * on dev there is no way to sign in as a Telegram user at all. These two calls
 * are the way round it — and they are useful in production for the same reason
 * a person with two accounts wants them to be one.
 */

/** Redeems the one-shot link the bot sent. The token IS the credential. */
export async function redeemLoginLink(token: string): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/login-link/redeem', { token })
}

/**
 * Sends the six digits from the bot.
 *
 * 🔴 Returns a PENDING request, not a finished link: nothing is merged until
 * the person confirms in the bot. A UI that says «привязано» here would be
 * announcing something that has not happened.
 */
export async function submitLinkCode(code: string): Promise<{ request_id: string; status: string }> {
  return api.post<{ request_id: string; status: string }>('/auth/link-code/redeem', { code })
}

/** Gives a Telegram-only account an email and a password. */
export async function setCredentials(email: string, password: string): Promise<void> {
  await api.post('/auth/credentials', { email, password })
}
