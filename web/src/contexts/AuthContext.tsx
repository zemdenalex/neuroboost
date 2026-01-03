import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import {
  User,
  TelegramAuthData,
  loginWithTelegram,
  loginWithEmail,
  register,
  getMe,
  logout as apiLogout,
} from '../api/auth'
import { getStoredToken, isTokenExpired, getTokenDaysRemaining, clearStoredToken } from '../api/client'

interface AuthContextValue {
  user: User | null
  loading: boolean
  error: string | null
  isAuthenticated: boolean
  tokenDaysRemaining: number
  
  // Auth methods
  loginWithTelegram: (data: TelegramAuthData) => Promise<void>
  loginWithEmail: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name?: string) => Promise<void>
  logout: () => Promise<void>
  clearError: () => void
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Check for existing session on mount
  useEffect(() => {
    const checkAuth = async () => {
      const token = getStoredToken()
      if (token && !isTokenExpired()) {
        try {
          const userData = await getMe()
          setUser(userData)
        } catch {
          clearStoredToken()
        }
      }
      setLoading(false)
    }
    checkAuth()
  }, [])

  const handleLoginWithTelegram = useCallback(async (data: TelegramAuthData) => {
    setLoading(true)
    setError(null)
    try {
      const response = await loginWithTelegram(data)
      setUser(response.user)
    } catch (err: any) {
      setError(err.message || 'Telegram login failed')
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const handleLoginWithEmail = useCallback(async (email: string, password: string) => {
    setLoading(true)
    setError(null)
    try {
      const response = await loginWithEmail(email, password)
      setUser(response.user)
    } catch (err: any) {
      setError(err.message || 'Login failed')
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const handleRegister = useCallback(async (email: string, password: string, name?: string) => {
    setLoading(true)
    setError(null)
    try {
      const response = await register(email, password, name)
      setUser(response.user)
    } catch (err: any) {
      setError(err.message || 'Registration failed')
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const handleLogout = useCallback(async () => {
    try {
      await apiLogout()
    } finally {
      setUser(null)
    }
  }, [])

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const value: AuthContextValue = {
    user,
    loading,
    error,
    isAuthenticated: !!user,
    tokenDaysRemaining: getTokenDaysRemaining(),
    loginWithTelegram: handleLoginWithTelegram,
    loginWithEmail: handleLoginWithEmail,
    register: handleRegister,
    logout: handleLogout,
    clearError,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuthContext() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuthContext must be used within AuthProvider')
  }
  return ctx
}

// Convenience hook for protected routes
export function useRequireAuth() {
  const { isAuthenticated, loading } = useAuthContext()
  return { isAuthenticated, loading }
}

// Convenience hook for admin-only features
export function useRequireAdmin() {
  const { user, isAuthenticated, loading } = useAuthContext()
  const isAdmin = user?.is_admin ?? false
  return { isAdmin, isAuthenticated, loading }
}
