const API_BASE = (import.meta.env?.VITE_API_URL ?? '/api').replace(/\/$/, '');

/**
 * Thrown by `request()` on any non-2xx / non-204 response. Extends Error so every
 * existing `catch (err) { ... err.message ... }` call site keeps working unchanged
 * (`instanceof Error` and `.message` both still hold) — this is a superset, not a
 * replacement. `code` and `raw` let a caller that cares (e.g. calendar delete,
 * which needs the CALENDAR_NOT_EMPTY counts) narrow with `instanceof ApiError`
 * instead of losing that data to string coercion.
 */
export class ApiError extends Error {
  code?: string;
  raw: unknown;

  constructor(message: string, code: string | undefined, raw: unknown) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.raw = raw;
  }
}

// Token storage keys
const TOKEN_KEY = 'nb_token';
const TOKEN_EXPIRY_KEY = 'nb_token_expiry';

// Token management functions
export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string, expiresAt?: number): void {
  localStorage.setItem(TOKEN_KEY, token);
  if (expiresAt) {
    localStorage.setItem(TOKEN_EXPIRY_KEY, String(expiresAt));
  }
}

export function clearStoredToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(TOKEN_EXPIRY_KEY);
}

export function isTokenExpired(): boolean {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (!expiry) return false;
  return Date.now() > Number(expiry) * 1000;
}

export function getTokenDaysRemaining(): number {
  const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);
  if (!expiry) return 0;
  const remaining = Number(expiry) * 1000 - Date.now();
  return Math.max(0, Math.floor(remaining / (24 * 60 * 60 * 1000)));
}

// Legacy aliases for backward compatibility
export function setAuthToken(token: string | null): void {
  if (token) {
    setStoredToken(token);
  } else {
    clearStoredToken();
  }
}

export function getAuthToken(): string | null {
  return getStoredToken();
}

/**
 * Narrows an unknown JSON value to an indexable object.
 *
 * `typeof null === 'object'` in JavaScript, so the null check is load-bearing:
 * without it a `null` body would pass and every property read below would throw
 * instead of falling through to the default message.
 */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

// Base request function with auth
async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  requireAuth = true
): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };

  const token = getStoredToken();
  if (token && requireAuth) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (response.status === 401) {
    clearStoredToken();
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (response.status === 204) {
    return {} as T;
  }

  // `unknown`, not `any`: this is the funnel every response passes through, so
  // an `any` here silently disables checking for the whole client. The reads
  // below narrow explicitly instead.
  const payload: unknown = await response.json().catch(() => null);
  const envelope = isRecord(payload) ? payload : undefined;
  const errorField = envelope?.error;

  if (!response.ok) {
    const msg =
      (isRecord(errorField) && typeof errorField.message === 'string' ? errorField.message : null) ??
      (typeof envelope?.message === 'string' ? envelope.message : null) ??
      (typeof errorField === 'string' ? errorField : null) ??
      'Request failed';

    const code = isRecord(errorField) && typeof errorField.code === 'string' ? errorField.code : undefined;
    throw new ApiError(msg, code, errorField);
  }

  if (envelope && 'data' in envelope) {
    return envelope.data as T;
  }

  return payload as T;
}

// API object for use by other modules
export const api = {
  get<T>(path: string, requireAuth = true): Promise<T> {
    return request<T>('GET', path, undefined, requireAuth);
  },

  post<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('POST', path, body, requireAuth);
  },

  patch<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('PATCH', path, body, requireAuth);
  },

  put<T>(path: string, body?: unknown, requireAuth = true): Promise<T> {
    return request<T>('PUT', path, body, requireAuth);
  },

  delete<T = void>(path: string, requireAuth = true): Promise<T> {
    return request<T>('DELETE', path, undefined, requireAuth);
  },
};

// ============ HEALTH API ============

export async function checkHealth(): Promise<{ ok: boolean; db?: string }> {
  return api.get<{ ok: boolean; db?: string }>('/health', false);
}
