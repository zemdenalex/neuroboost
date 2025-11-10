import { useEffect, useState } from 'react'
import * as auth from '../api/auth'
export function useAuth() {
  const [user, setUser] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  useEffect(() => {
    setLoading(true)
    auth.getMe().then(() => setUser(null)).finally(() => setLoading(false))
  }, [])
  async function login(initData: string) { await auth.loginWithTelegram(initData); setUser({ id: 'stub' }) }
  async function logout() { await auth.logout(); setUser(null) }
  return { user, loading, login, logout }
}
