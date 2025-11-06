import { useState } from 'react'
import { api } from '../api/client'
export function useApi() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  async function request(method: 'get'|'post'|'patch'|'delete', url: string, body?: any) {
    setLoading(true); setError(null)
    try {
      // @ts-ignore
      const fn = api[method]
      return await fn(url, body)
    } catch (e:any) {
      setError(e?.message || 'error'); throw e
    } finally { setLoading(false) }
  }
  return { loading, error, request }
}
