const API_BASE = (import.meta.env?.VITE_API_URL ?? '/api').replace(/\/$/, '');

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

  const payload: any = await response.json().catch(() => null);

  if (!response.ok) {
    const msg =
      payload?.error?.message ??
      payload?.message ??
      (typeof payload?.error === 'string' ? payload.error : null) ??
      'Request failed';

    throw new Error(typeof msg === 'string' ? msg : JSON.stringify(msg));
  }

  if (payload && typeof payload === 'object' && 'data' in payload) {
    return payload.data as T;
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
