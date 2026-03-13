import { useEffect, useState, useCallback } from 'react'
import { listTasks, type Task } from '../api/tasks'

export function useTasks() {
  const [data, setData] = useState<Task[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listTasks()
      .then(setData)
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Failed to load tasks'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  return { tasks: data, loading, error, reload: load }
}
