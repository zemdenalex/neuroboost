import { useEffect, useState } from 'react'
import * as events from '../api/events'
export function useEvents(from: string, to: string) {
  const [data, setData] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    setLoading(true)
    events.list(from, to).then(() => setData([])).finally(() => setLoading(false))
  }, [from, to])
  return { events: data, loading }
}
