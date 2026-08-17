import { useEffect, useState } from 'react'

export interface ToastAction {
  label: string
  onClick: () => void
}

interface ToastState {
  message: string
  action?: ToastAction
}

type ShowFn = (state: ToastState) => void

let showImpl: ShowFn | null = null

/** How long a plain toast stays up. */
const PLAIN_MS = 2000
/** Longer when there is something to click — 2s is not enough to reach Undo. */
const ACTIONABLE_MS = 6000

/**
 * Trigger a transient toast notification. No-op if <ToastHost /> is not mounted.
 * Pass an action to offer a single button (Undo, Retry).
 */
export function showToast(msg: string, action?: ToastAction): void {
  showImpl?.({ message: msg, action })
}

/**
 * Mount once near the root of the app. Renders a small status message
 * in the top-right corner that fades out on its own.
 */
export function ToastHost(): JSX.Element | null {
  const [toast, setToast] = useState<ToastState | null>(null)

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | null = null
    showImpl = (next: ToastState) => {
      if (timeoutId) clearTimeout(timeoutId)
      setToast(next)
      timeoutId = setTimeout(() => setToast(null), next.action ? ACTIONABLE_MS : PLAIN_MS)
    }
    return () => {
      showImpl = null
      if (timeoutId) clearTimeout(timeoutId)
    }
  }, [])

  if (!toast) return null
  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed top-4 right-4 z-50 flex items-center gap-3 px-4 py-2 bg-emerald-600 text-white font-mono text-sm rounded-lg shadow-lg"
    >
      <span>{toast.message}</span>
      {toast.action && (
        <button
          type="button"
          onClick={() => {
            const run = toast.action?.onClick
            // Dismiss first so a slow handler cannot leave the toast clickable twice.
            setToast(null)
            run?.()
          }}
          className="underline underline-offset-2 hover:no-underline focus:outline-none focus-visible:ring-2 focus-visible:ring-white"
        >
          {toast.action.label}
        </button>
      )}
    </div>
  )
}
