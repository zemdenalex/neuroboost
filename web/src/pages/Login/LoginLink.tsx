import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Loader2, XCircle } from 'lucide-react'
import { redeemLoginLink } from '../../api/auth'
import { setStoredToken } from '../../api/client'

/**
 * Signing in from the link the bot sent: /login/link?t=…
 *
 * 🔴 The token is spent on arrival, once. Two consequences shape this page:
 *
 *   - React 18 StrictMode mounts effects twice in development. The second call
 *     carries a spent token and answers 410, which would show the failure
 *     screen for a link that had just worked. The ref makes the attempt once
 *     per token — the same reason AcceptInvite has one, and for exactly the
 *     same failure.
 *   - The page does NOT require a session; it creates one. A ProtectedRoute
 *     here would send the visitor to the login screen they have no way to pass,
 *     which is the whole problem this link exists to solve (Denis, 17.09: «dev
 *     website doesn't work with telegram login, I can't test it»).
 *
 * ⚠ The token is read from the query string and never written anywhere else —
 * not to state that outlives the attempt, not to a log, not into the URL after
 * redirect. `replace: true` on the navigation takes it out of history.
 */
export function LoginLink() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get('t') ?? ''

  const [state, setState] = useState<'working' | 'failed'>('working')
  const attempted = useRef<string | null>(null)

  useEffect(() => {
    if (!token) {
      setState('failed')
      return
    }
    if (attempted.current === token) return
    attempted.current = token

    redeemLoginLink(token)
      .then((response) => {
        setStoredToken(response.token, response.expires_at)
        // A full reload rather than a client-side route change: AuthProvider
        // reads the token on mount, and navigating inside the running app would
        // leave it holding the signed-out state it started with.
        window.location.replace('/profile')
      })
      .catch(() => setState('failed'))
  }, [token, navigate])

  if (state === 'working') {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="flex items-center gap-3 text-gray-600 dark:text-gray-300">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>Входим…</span>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-sm rounded-lg border border-gray-200 p-6 text-center dark:border-gray-700">
        <XCircle className="mx-auto mb-3 h-8 w-8 text-red-500" />
        <h1 className="mb-2 text-lg font-semibold">Ссылка не сработала</h1>
        <p className="mb-4 text-sm text-gray-600 dark:text-gray-300">
          Она работает один раз и живёт десять минут. Попроси в боте новую:
          ⚙️ Настройки → 🔗 Аккаунт на сайте.
        </p>
        <button
          type="button"
          onClick={() => navigate('/login', { replace: true })}
          className="w-full rounded-md bg-gray-900 px-4 py-2 text-sm text-white dark:bg-gray-100 dark:text-gray-900"
        >
          Ко входу
        </button>
      </div>
    </div>
  )
}

export default LoginLink
