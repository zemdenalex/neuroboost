import { useEffect, useState } from 'react'
export function useLocalStorage<T>(key: string, initialValue: T) {
  const [value, setValue] = useState<T>(initialValue)
  useEffect(() => { try { const raw = localStorage.getItem(key); if (raw) setValue(JSON.parse(raw)) } catch(_){} }, [key])
  useEffect(() => { try { localStorage.setItem(key, JSON.stringify(value)) } catch(_){} }, [key, value])
  return [value, setValue] as const
}
