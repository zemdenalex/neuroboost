import { ApiError } from '../../api/client'

/**
 * Restoring a session on app start. Until 07.11 any failure of `getMe()` dropped
 * the stored token, so a dropped connection or a 502 while staging/prod was
 * being redeployed logged the person out (A5 offline review, AuthContext
 * `checkAuth`). A real 401 never reaches here as a session to keep: the API
 * client clears the token and goes to /login on its own (client.ts).
 *
 * Only the server saying «this credential is not valid» ends a session. A
 * network error or a 5xx is the line, not the person, and the token is kept.
 */
export function endsSession(err: unknown): boolean {
  return err instanceof ApiError && (err.status === 401 || err.status === 403)
}

/** Worth asking again: no answer at all, or the server was down. */
function transient(err: unknown): boolean {
  if (err instanceof ApiError) return err.status === undefined || err.status >= 500
  return true // fetch rejects with a TypeError when there is no network
}

export const STARTUP_RETRY_MS = [1000, 3000] as const

/**
 * `load` with two retries for transient failures. The last error is rethrown so
 * the caller decides, with `endsSession`, whether to drop the token.
 */
export async function loadWithRetry<T>(
  load: () => Promise<T>,
  wait: (ms: number) => Promise<void> = (ms) => new Promise((r) => setTimeout(r, ms)),
  delays: readonly number[] = STARTUP_RETRY_MS,
): Promise<T> {
  for (let attempt = 0; ; attempt++) {
    try {
      return await load()
    } catch (err) {
      if (attempt >= delays.length || !transient(err)) throw err
      await wait(delays[attempt])
    }
  }
}
