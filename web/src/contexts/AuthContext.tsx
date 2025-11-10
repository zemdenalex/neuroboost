import React, { createContext, useContext } from 'react'
import { useAuth } from '../hooks/useAuth'
interface AuthContextValue {
  user: any
  loading: boolean
  login: (initData: string) => Promise<void>
  logout: () => Promise<void>
}
const AuthContext = createContext<AuthContextValue | undefined>(undefined)
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const { user, loading, login, logout } = useAuth()
  return <AuthContext.Provider value={{ user, loading, login, logout }}>{children}</AuthContext.Provider>
}
export function useAuthContext() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuthContext must be used within AuthProvider')
  return ctx
}
