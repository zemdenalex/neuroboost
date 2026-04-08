import { useEffect, useState, useCallback } from 'react'
import { listEvents, type Event } from '../api/events'

export function useEvents(from: string, to: string) {
  const [data, setData] = useState<Event[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listEvents(from, to)
      .then(setData)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load events'))
      .finally(() => setLoading(false))
  }, [from, to])

  useEffect(() => { load() }, [load])

  return { events: data, loading, error, reload: load }
}
