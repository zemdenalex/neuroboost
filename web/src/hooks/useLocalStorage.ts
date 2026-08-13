import { useEffect, useState } from 'react'

export function useLocalStorage<T>(key: string, initialValue: T) {
  const [value, setValue] = useState<T>(initialValue)

  useEffect(() => {
    try {
      const raw = localStorage.getItem(key)
      if (raw) setValue(JSON.parse(raw))
    } catch (_) {
      // Swallowed on purpose: localStorage throws in private mode and when the
      // stored value is not JSON. Either way the caller's initialValue stands,
      // which is the correct fallback — there is nothing to report and nothing
      // to retry.
    }
  }, [key])

  useEffect(() => {
    try {
      localStorage.setItem(key, JSON.stringify(value))
    } catch (_) {
      // Same reasoning: a full or unavailable quota must not break the render
      // that triggered the write.
    }
  }, [key, value])

  return [value, setValue] as const
}
