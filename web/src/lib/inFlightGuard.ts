/**
 * Creates a guard that runs an async task but ignores re-entrant calls while a
 * previous run is still in flight. Use it to prevent double-submit: two clicks
 * in the same tick both read the same closure flag, so only the first proceeds.
 *
 * Hold the returned function across renders (e.g. `useRef(createInFlightGuard()).current`)
 * — a `useState` flag cannot block a synchronous double-click because React has
 * not re-rendered with the updated value before the second handler fires.
 *
 * The task's rejection is propagated to the caller (so existing try/catch still
 * works); the in-flight flag is always reset in `finally`, so a failed submit
 * never leaves the guard stuck.
 */
export function createInFlightGuard() {
  let inFlight = false
  return async function run(task: () => Promise<void>): Promise<void> {
    if (inFlight) return
    inFlight = true
    try {
      await task()
    } finally {
      inFlight = false
    }
  }
}
