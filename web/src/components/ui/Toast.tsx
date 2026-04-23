import { useEffect, useState } from 'react'

type ShowFn = (msg: string) => void

let showImpl: ShowFn | null = null

/**
 * Trigger a transient toast notification. No-op if <ToastHost /> is not mounted.
 */
export function showToast(msg: string): void {
  showImpl?.(msg)
}

/**
 * Mount once near the root of the app. Renders a small status message
 * in the top-right corner that fades out after ~2s.
 */
export function ToastHost(): JSX.Element | null {
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | null = null
    showImpl = (msg: string) => {
      if (timeoutId) clearTimeout(timeoutId)
      setMessage(msg)
      timeoutId = setTimeout(() => setMessage(null), 2000)
    }
    return () => {
      showImpl = null
      if (timeoutId) clearTimeout(timeoutId)
    }
  }, [])

  if (!message) return null
  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed top-4 right-4 z-50 px-4 py-2 bg-emerald-600 text-white font-mono text-sm rounded-lg shadow-lg"
    >
      {message}
    </div>
  )
}
