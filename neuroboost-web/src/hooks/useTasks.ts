import { useEffect, useState } from 'react'
import * as tasks from '../api/tasks'
export function useTasks() {
  const [data, setData] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    setLoading(true)
    tasks.list().then(() => setData([])).finally(() => setLoading(false))
  }, [])
  return { tasks: data, loading }
}
