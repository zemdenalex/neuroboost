const API_BASE = import.meta.env.VITE_API_URL || '/api'

// Token management
const TOKEN_KEY = 'neuroboost_token'
const TOKEN_EXPIRY_KEY = 'neuroboost_token_expiry'

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setStoredToken(token: string, expiresAt: number): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(TOKEN_EXPIRY_KEY, expiresAt.toString())
}

export function clearStoredToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(TOKEN_EXPIRY_KEY)
}

export function isTokenExpired(): boolean {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY)
  if (!expiry) return true
  return Date.now() / 1000 > parseInt(expiry)
}

export function getTokenDaysRemaining(): number {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY)
  if (!expiry) return 0
  const expiryTime = parseInt(expiry)
  const now = Date.now() / 1000
  return Math.max(0, Math.floor((expiryTime - now) / 86400))
}

// API Error class
export class ApiError extends Error {
  code: string
  status: number

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
    this.name = 'ApiError'
  }
}

// Generic fetch wrapper
async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  requireAuth: boolean = true
): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  }

  const token = getStoredToken()
  if (requireAuth && token && !isTokenExpired()) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  const data = await response.json().catch(() => ({}))

  if (!response.ok) {
    throw new ApiError(
      response.status,
      data.error?.code || 'UNKNOWN_ERROR',
      data.error?.message || 'An error occurred'
    )
  }

  return data.data !== undefined ? data.data : data
}

// API client object
export const api = {
  get: <T>(path: string, requireAuth = true) => request<T>('GET', path, undefined, requireAuth),
  post: <T>(path: string, body: unknown, requireAuth = true) => request<T>('POST', path, body, requireAuth),
  patch: <T>(path: string, body: unknown, requireAuth = true) => request<T>('PATCH', path, body, requireAuth),
  delete: <T>(path: string, requireAuth = true) => request<T>('DELETE', path, undefined, requireAuth),
}
