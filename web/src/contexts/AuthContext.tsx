import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react'
import { User, UserSettings, getMe, logout as apiLogout, updateMe } from '../api/auth'
import { api } from '../api/client'

interface AuthContextType {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (token: string, user: User) => void
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
  updateSettings: (settings: Partial<UserSettings>) => Promise<void>
  updateProfile: (data: { display_name?: string; timezone?: string; locale?: string }) => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // Load user on mount if token exists
  useEffect(() => {
    const loadUser = async () => {
      const token = localStorage.getItem('neuroboost-token')
      if (token) {
        api.setToken(token)
        try {
          const userData = await getMe()
          setUser(userData)
          // Apply settings from DB to localStorage for components that read from there
          if (userData.settings) {
            applySettingsToLocalStorage(userData.settings)
          }
        } catch {
          // Token invalid, clear it
          localStorage.removeItem('neuroboost-token')
          api.setToken(null)
        }
      }
      setIsLoading(false)
    }
    loadUser()
  }, [])

  const login = useCallback((token: string, userData: User) => {
    localStorage.setItem('neuroboost-token', token)
    api.setToken(token)
    setUser(userData)
    // Apply settings from DB
    if (userData.settings) {
      applySettingsToLocalStorage(userData.settings)
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiLogout()
    } catch {
      // Ignore errors during logout
    }
    localStorage.removeItem('neuroboost-token')
    api.setToken(null)
    setUser(null)
  }, [])

  const refreshUser = useCallback(async () => {
    try {
      const userData = await getMe()
      setUser(userData)
      if (userData.settings) {
        applySettingsToLocalStorage(userData.settings)
      }
    } catch {
      // User fetch failed
    }
  }, [])

  const updateSettings = useCallback(async (settings: Partial<UserSettings>) => {
    if (!user) return
    
    // Merge with existing settings
    const newSettings = { ...user.settings, ...settings }
    
    try {
      const updatedUser = await updateMe({ settings: newSettings })
      setUser(updatedUser)
      applySettingsToLocalStorage(updatedUser.settings)
      
      // Dispatch custom event for components that listen
      if (settings.header_variant) {
        window.dispatchEvent(new CustomEvent('neuroboost-layout-change', { 
          detail: settings.header_variant 
        }))
      }
      if (settings.ui_scale) {
        window.dispatchEvent(new CustomEvent('neuroboost-scale-change', { 
          detail: settings.ui_scale 
        }))
      }
    } catch (error) {
      console.error('Failed to update settings:', error)
      throw error
    }
  }, [user])

  const updateProfile = useCallback(async (data: { display_name?: string; timezone?: string; locale?: string }) => {
    if (!user) return
    
    try {
      const updatedUser = await updateMe(data)
      setUser(updatedUser)
    } catch (error) {
      console.error('Failed to update profile:', error)
      throw error
    }
  }, [user])

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        logout,
        refreshUser,
        updateSettings,
        updateProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuthContext() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuthContext must be used within an AuthProvider')
  }
  return context
}

// Helper to sync DB settings to localStorage for components that read from there
function applySettingsToLocalStorage(settings: UserSettings) {
  if (settings.header_variant) {
    localStorage.setItem('neuroboost-header-variant', settings.header_variant)
  }
  if (settings.ui_scale) {
    localStorage.setItem('neuroboost-ui-scale', String(settings.ui_scale))
  }
  if (settings.work_days) {
    localStorage.setItem('neuroboost-work-days', JSON.stringify(settings.work_days))
  }
  if (settings.work_start) {
    localStorage.setItem('neuroboost-work-start', settings.work_start)
  }
  if (settings.work_end) {
    localStorage.setItem('neuroboost-work-end', settings.work_end)
  }
  if (settings.features) {
    localStorage.setItem('neuroboost-features', JSON.stringify(settings.features))
  }
}