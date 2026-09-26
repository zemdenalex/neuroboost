import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import i18n from '../i18n'
import { errorMessage } from '../lib/errorMessage'
import { createSettingsSaver } from '../lib/settings/saveSettings'
import { startupLanguage } from '../lib/settings/language'
import {
  User,
  UserSettings,
  TelegramUser,
  login as emailLogin,
  register as registerUser,
  telegramLogin,
  telegramWebAppLogin,
  getMe,
  logout as apiLogout,
  updateMe,
} from '../api/auth'
import {
  getStoredToken,
  isTokenExpired,
  getTokenDaysRemaining,
  setStoredToken,
  clearStoredToken,
} from '../api/client'
import { applyCurrentTheme, readThemeChoice, storeThemeChoice } from '../lib/theme/theme'
import { launchedInTelegram, loadWebApp, pickStartupAuth, prepareWebApp, sessionFitsLaunch } from '../lib/telegram/webApp'

/**
 * AuthContextValue defines the shape of the authentication context.  In addition
 * to the current user and loading state, it exposes a handful of
 * convenience methods for logging in via email or Telegram, registering
 * new users, refreshing profile data, updating settings, and logging
 * out.  The error field can be populated by failed authentication
 * attempts so that UI components can display meaningful feedback.
 */
export interface AuthContextValue {
  user: User | null
  loading: boolean
  error: string | null
  isAuthenticated: boolean
  tokenDaysRemaining: number
  loginWithEmail: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name?: string) => Promise<void>
  loginWithTelegram: (data: TelegramUser) => Promise<void>
  logout: () => Promise<void>
  clearError: () => void
  refreshUser: () => Promise<void>
  updateSettings: (settings: Partial<UserSettings>) => Promise<void>
  updateProfile: (data: { display_name?: string; timezone?: string; locale?: string }) => Promise<void>
  /** Interface language for the web AND the bot (one language per person). */
  updateLanguage: (locale: string) => Promise<void>
  /** One key of settings.bot (shared with the bot), merged on the server's copy. */
  updateBotSetting: (key: string, value: unknown) => Promise<void>
}

// One saver for the app: its queue is what keeps two quick saves apart.
const saveSettings = createSettingsSaver({ getMe, updateMe })

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

/**
 * AuthProvider wraps the React component tree and manages authentication
 * state.  On mount it checks for an existing session token in localStorage
 * and, if valid, fetches the current user profile.  It also exposes
 * methods to perform email/Telegram login, registration, logout, and
 * profile updates.  Errors during authentication are captured in
 * state for easy consumption by UI consumers.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // On mount, attempt to restore session from localStorage
  useEffect(() => {
    const checkAuth = async () => {
      // Telegram Mini App: the launch hash carries a signed identity, so the
      // person lands in the app without a login screen. A failed exchange
      // falls through to the ordinary path below rather than stranding them.
      const initData = launchedInTelegram()
      void loadWebApp().then((wa) => wa && prepareWebApp(wa))
      const storedTokenValid = Boolean(getStoredToken()) && !isTokenExpired()
      if (pickStartupAuth({ initData, storedTokenValid }) === 'webapp' && initData) {
        try {
          const response = await telegramWebAppLogin(initData)
          setStoredToken(response.token, response.expires_at)
        } catch {
          // A stored session survives only if it is this person's: checked
          // against the launch below (sessionFitsLaunch, review I2).
        }
      }
      const token = getStoredToken()
      if (token && !isTokenExpired()) {
        try {
          const userData = await getMe()
          if (!sessionFitsLaunch(initData, userData.tg_id)) {
            // Somebody else's session in this WebView: drop it, show the login.
            clearStoredToken()
            setLoading(false)
            return
          }
          setUser(userData)
          if (userData.settings) {
            applySettingsToLocalStorage(userData.settings)
          }
          // Sync the language to i18n. Inside Telegram a bot choice that
          // differs wins once and is saved to both (one language per person).
          const lang = startupLanguage({
            locale: userData.locale,
            botLang: (userData.settings as { bot?: { lang?: unknown } } | undefined)?.bot?.lang,
            inTelegram: Boolean(initData),
          })
          if (lang.use && i18n.language !== lang.use) {
            i18n.changeLanguage(lang.use)
            localStorage.setItem('neuroboost-locale', lang.use)
          }
          if (lang.save && lang.use) {
            saveSettings.language(lang.use).then(setUser).catch(() => {
              // Shown in the chosen language already; the next launch retries.
            })
          }
        } catch {
          // If fetching the user fails, clear the stored token to avoid
          // repeatedly attempting with a stale credential.
          clearStoredToken()
        }
      } else {
        // Token missing or expired
        clearStoredToken()
      }
      setLoading(false)
    }
    checkAuth()
  }, [])

  /**
   * Perform email/password login.  On success, persist the returned
   * token to localStorage and update the current user.  On failure, set
   * an error message and rethrow so callers can handle rejection.
   */
  const handleLoginWithEmail = useCallback(
    async (email: string, password: string) => {
      setLoading(true)
      setError(null)
      try {
        const response = await emailLogin({ email, password })
        setStoredToken(response.token, response.expires_at)
        setUser(response.user)
        if (response.user.settings) {
          applySettingsToLocalStorage(response.user.settings)
        }
      } catch (err) {
        setError(errorMessage(err, 'Login failed'))
        throw err
      } finally {
        setLoading(false)
      }
    },
    []
  )

  /**
   * Register a new user.  On success, persist the token and current
   * user.  Behaviour mirrors that of handleLoginWithEmail.
   */
  const handleRegister = useCallback(
    async (email: string, password: string, name?: string) => {
      setLoading(true)
      setError(null)
      try {
        const response = await registerUser({ email, password, name })
        setStoredToken(response.token, response.expires_at)
        setUser(response.user)
        if (response.user.settings) {
          applySettingsToLocalStorage(response.user.settings)
        }
      } catch (err) {
        setError(errorMessage(err, 'Registration failed'))
        throw err
      } finally {
        setLoading(false)
      }
    },
    []
  )

  /**
   * Perform Telegram login using the provided auth data.  On success,
   * persist the token and current user.
   */
  const handleLoginWithTelegram = useCallback(
    async (data: TelegramUser) => {
      setLoading(true)
      setError(null)
      try {
        const response = await telegramLogin(data)
        setStoredToken(response.token, response.expires_at)
        setUser(response.user)
        if (response.user.settings) {
          applySettingsToLocalStorage(response.user.settings)
        }
      } catch (err) {
        setError(errorMessage(err, 'Telegram login failed'))
        throw err
      } finally {
        setLoading(false)
      }
    },
    []
  )

  /**
   * Clear the current session and log the user out.  Always clears the
   * local token even if the API call fails.
   */
  const handleLogout = useCallback(async () => {
    try {
      await apiLogout()
    } finally {
      clearStoredToken()
      setUser(null)
    }
  }, [])

  /**
   * Refresh the current user's data from the server.  Useful when
   * navigating between protected routes or after updating profile
   * information.
   */
  const refreshUser = useCallback(async () => {
    try {
      const userData = await getMe()
      setUser(userData)
      if (userData.settings) {
        applySettingsToLocalStorage(userData.settings)
      }
    } catch {
      // Ignore refresh errors – these can be handled by callers
    }
  }, [])

  /**
   * Update user settings.  Merges incoming changes with existing
   * settings, persists them to the server, updates the local user
   * state, and syncs relevant settings to localStorage for UI
   * consumers.
   */
  const updateSettings = useCallback(
    async (settings: Partial<UserSettings>) => {
      if (!user) return
      try {
        // Merged over what the server holds now, not over this tab's copy:
        // the bot writes settings too (see saveSettings.ts).
        const updatedUser = await saveSettings(settings)
        setUser(updatedUser)
        if (updatedUser.settings) {
          applySettingsToLocalStorage(updatedUser.settings)
        }
        // Dispatch custom events for components that listen for layout or scale changes
        if (settings.header_variant) {
          window.dispatchEvent(
            new CustomEvent('neuroboost-layout-change', { detail: settings.header_variant })
          )
        }
        if (settings.ui_scale) {
          window.dispatchEvent(
            new CustomEvent('neuroboost-scale-change', { detail: settings.ui_scale })
          )
        }
        if (settings.mobile_nav) {
          window.dispatchEvent(
            new CustomEvent('neuroboost-mobile-nav-change', { detail: settings.mobile_nav })
          )
        }
      } catch (err) {
        console.error('Failed to update settings:', err)
        throw err
      }
    },
    [user]
  )

  /**
   * Update user profile fields (display name, timezone, locale).  Does
   * not modify settings.
   */
  const updateProfile = useCallback(
    async (data: { display_name?: string; timezone?: string; locale?: string }) => {
      if (!user) return
      try {
        const updatedUser = await updateMe(data)
        setUser(updatedUser)
      } catch (err) {
        console.error('Failed to update profile:', err)
        throw err
      }
    },
    [user]
  )

  const updateLanguage = useCallback(async (locale: string) => {
    setUser(await saveSettings.language(locale))
  }, [])

  const updateBotSetting = useCallback(async (key: string, value: unknown) => {
    setUser(await saveSettings.botSetting(key, value))
  }, [])

  /**
   * Reset the current error state.
   */
  const clearError = useCallback(() => {
    setError(null)
  }, [])

  const value: AuthContextValue = {
    user,
    loading,
    error,
    isAuthenticated: !!user,
    tokenDaysRemaining: getTokenDaysRemaining(),
    loginWithEmail: handleLoginWithEmail,
    register: handleRegister,
    loginWithTelegram: handleLoginWithTelegram,
    logout: handleLogout,
    clearError,
    refreshUser,
    updateSettings,
    updateProfile,
    updateLanguage,
    updateBotSetting,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

/**
 * Hook to access the authentication context.  Throws an error if
 * called outside of an AuthProvider.
 */
export function useAuthContext() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuthContext must be used within an AuthProvider')
  }
  return ctx
}

/**
 * Convenience hook for checking authentication state inside route
 * guards.  Returns the current authentication status and loading
 * state.
 */
export function useRequireAuth() {
  const { isAuthenticated, loading } = useAuthContext()
  return { isAuthenticated, loading }
}

/**
 * Convenience hook for admin-only features.  Returns whether the
 * current user is an admin, along with authentication and loading
 * state.
 */
export function useRequireAdmin() {
  const { user, isAuthenticated, loading } = useAuthContext()
  const isAdmin = user?.is_admin ?? false
  return { isAdmin, isAuthenticated, loading }
}

/**
 * Sync relevant settings to localStorage so that non-React
 * components (e.g. Tailwind classes applied directly to the DOM)
 * can read user preferences.  This helper mirrors the logic used in
 * older versions of the app and is invoked whenever the user
 * settings are updated or loaded.
 */
function applySettingsToLocalStorage(settings: UserSettings) {
  // The account's theme choice, mirrored on the device for the next first frame.
  const theme = readThemeChoice(settings)
  storeThemeChoice(theme)
  applyCurrentTheme(theme)
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
  if (settings.mobile_nav) {
    localStorage.setItem('neuroboost-mobile-nav', settings.mobile_nav)
  }
}