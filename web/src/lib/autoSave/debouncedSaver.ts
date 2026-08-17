/**
 * Debounced save that settles its debt when told to stop.
 *
 * Extracted from Settings.tsx on 2026-08-14. The machinery existed there twice
 * — once for settings, once for profile — and both copies had the same defect:
 * the unmount cleanup was commented "flush any pending save on unmount so a
 * quick navigation doesn't drop changes" and its body only called clearTimeout.
 * It cancelled rather than flushed. Any change followed by leaving the page
 * inside the debounce window was lost, and lost silently, because the "saved"
 * toast is printed after a successful save and never ran either.
 *
 * Deliberately plain: no React. The repository tests pure cores in `lib/` and
 * keeps hooks thin (see lib/calendars/order.ts, lib/tools/*), and a defect that
 * only reproduces through a rendered component is a defect nothing guards —
 * there is no component-testing setup here, and adding one to prove a
 * ten-line timer would be the tail wagging the dog.
 */
export interface DebouncedSaverOptions<T extends object> {
  /** Sends the merged patch. Called at most once per debounce window. */
  save: (patch: T) => Promise<unknown>
  /** Runs after a save that completed normally. */
  onSaved?: () => void
  /** Runs when a save rejected. */
  onError?: (err: unknown) => void
  /** Runs when a flush-time save rejected; the caller may already be gone. */
  onFlushError?: (err: unknown) => void
  delay?: number
}

export interface DebouncedSaver<T extends object> {
  /** Merge a patch into what is pending and (re)start the timer. */
  schedule: (patch: Partial<T>) => void
  /** Send whatever is owed right now. Safe to call with nothing pending. */
  flush: () => void
  /** For tests and diagnostics: is anything owed? */
  hasPending: () => boolean
}

export function createDebouncedSaver<T extends object>({
  save,
  onSaved,
  onError,
  onFlushError,
  delay = 300,
}: DebouncedSaverOptions<T>): DebouncedSaver<T> {
  let timer: ReturnType<typeof setTimeout> | null = null
  let pending: Partial<T> = {}

  const takePending = (): Partial<T> => {
    const taken = pending
    pending = {}
    return taken
  }

  return {
    schedule(patch) {
      // Merge, do not replace: two fields changed inside one window must both
      // survive, rather than the second overwriting the first.
      pending = { ...pending, ...patch }
      if (timer) clearTimeout(timer)
      timer = setTimeout(async () => {
        timer = null
        const toSave = takePending()
        try {
          await save(toSave as T)
          onSaved?.()
        } catch (err) {
          onError?.(err)
        }
      }, delay)
    },

    flush() {
      if (timer) {
        clearTimeout(timer)
        timer = null
      }
      const toSave = takePending()
      // Nothing owed: stay silent rather than sending an empty patch, which
      // some endpoints treat as "clear these fields".
      if (Object.keys(toSave).length === 0) return

      // 🔴 No success callback here. A flush runs when the owner is going away,
      // and a toast or a setState at that point lands on a component that no
      // longer exists. A failure is reported through onFlushError, whose only
      // sane implementation is a log.
      void Promise.resolve(save(toSave as T)).catch((err) => onFlushError?.(err))
    },

    hasPending() {
      return Object.keys(pending).length > 0
    },
  }
}
