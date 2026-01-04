import { useState, useEffect, useRef } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthContext } from '../../contexts/AuthContext'
import { LogIn, UserPlus, AlertCircle, Loader2 } from 'lucide-react'

type AuthMode = 'login' | 'register'

declare global {
  interface Window {
    onTelegramAuth: (user: TelegramUser) => void
  }
}

interface TelegramUser {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}

export function Login() {
  const navigate = useNavigate()
  const location = useLocation()
  const { loginWithEmail, register, loginWithTelegram, loading, error, clearError } = useAuthContext()
  const telegramContainerRef = useRef<HTMLDivElement>(null)

  const [mode, setMode] = useState<AuthMode>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  const from = (location.state as any)?.from?.pathname || '/calendar'

  // Load Telegram widget
  useEffect(() => {
    const handleTelegramAuth = (user: TelegramUser) => {
      loginWithTelegram({
        id: user.id,
        first_name: user.first_name,
        last_name: user.last_name,
        username: user.username,
        photo_url: user.photo_url,
        auth_date: user.auth_date,
        hash: user.hash,
      })
        .then(() => navigate(from, { replace: true }))
        .catch(() => {})
    }

    // Expose callback globally
    window.onTelegramAuth = handleTelegramAuth

    // Load Telegram script
    const container = telegramContainerRef.current
    if (container && !container.querySelector('iframe')) {
      const script = document.createElement('script')
      script.src = 'https://telegram.org/js/telegram-widget.js?22'
      script.setAttribute('data-telegram-login', 'NeuroBoost_assistant_bot')
      script.setAttribute('data-size', 'large')
      script.setAttribute('data-radius', '8')
      script.setAttribute('data-onauth', 'onTelegramAuth(user)')
      script.setAttribute('data-request-access', 'write')
      script.async = true
      container.appendChild(script)
    }

    return () => {
      delete (window as any).onTelegramAuth
    }
  }, [loginWithTelegram, navigate, from])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError(null)
    clearError()

    if (!email || !password) {
      setLocalError('Email and password are required')
      return
    }

    if (mode === 'register' && password.length < 8) {
      setLocalError('Password must be at least 8 characters')
      return
    }

    try {
      if (mode === 'login') {
        await loginWithEmail(email, password)
      } else {
        await register(email, password, name)
      }
      navigate(from, { replace: true })
    } catch {
      // Error is handled by AuthContext
    }
  }

  const toggleMode = () => {
    setMode(mode === 'login' ? 'register' : 'login')
    setLocalError(null)
    clearError()
  }

  const displayError = localError || error

  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-mono font-bold text-white">NeuroBoost</h1>
          <p className="text-zinc-400 mt-2">Calendar-first productivity</p>
        </div>

        {/* Auth Card */}
        <div className="bg-zinc-900 rounded-lg border border-zinc-800 p-6">
          <h2 className="text-xl font-mono text-white mb-6">
            {mode === 'login' ? 'Welcome back' : 'Create account'}
          </h2>

          {/* Error */}
          {displayError && (
            <div className="mb-4 p-3 bg-red-900/20 border border-red-800 rounded-lg flex items-center gap-2 text-red-400">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span className="text-sm">{displayError}</span>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            {mode === 'register' && (
              <div>
                <label className="block text-sm text-zinc-400 mb-1">Name (optional)</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                  placeholder="Your name"
                />
              </div>
            )}

            <div>
              <label className="block text-sm text-zinc-400 mb-1">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                placeholder="you@example.com"
                required
              />
            </div>

            <div>
              <label className="block text-sm text-zinc-400 mb-1">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                placeholder={mode === 'register' ? 'Min 8 characters' : '••••••••'}
                required
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white font-mono rounded-lg flex items-center justify-center gap-2 transition-colors"
            >
              {loading ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : mode === 'login' ? (
                <LogIn className="w-4 h-4" />
              ) : (
                <UserPlus className="w-4 h-4" />
              )}
              {mode === 'login' ? 'Sign in' : 'Create account'}
            </button>
          </form>

          {/* Toggle mode */}
          <button
            onClick={toggleMode}
            className="w-full mt-4 text-sm text-zinc-400 hover:text-white transition-colors"
          >
            {mode === 'login' ? "Don't have an account? " : 'Already have an account? '}
            <span className="text-blue-400 hover:text-blue-300 underline">
              {mode === 'login' ? 'Sign up' : 'Sign in'}
            </span>
          </button>

          {/* Divider */}
          <div className="my-6 flex items-center gap-4">
            <div className="flex-1 h-px bg-zinc-800" />
            <span className="text-xs text-zinc-500">OR</span>
            <div className="flex-1 h-px bg-zinc-800" />
          </div>

          {/* Telegram Login */}
          <div className="flex justify-center">
            <div ref={telegramContainerRef} id="telegram-login-container" />
          </div>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-zinc-500 mt-6">
          By signing in, you agree to our Terms of Service
        </p>
      </div>
    </div>
  )
}

export default Login