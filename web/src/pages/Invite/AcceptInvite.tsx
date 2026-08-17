import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { CalendarDays, Loader2, XCircle } from 'lucide-react'
import { acceptInviteLink } from '../../api/calendars'
import { describeCalendarError } from '../../lib/calendars/errors'
import { useAuthContext } from '../../contexts/AuthContext'

/**
 * Redeeming a share link: /i/:token
 *
 * 🔴 The token is spent on arrival, once, and that shapes everything here.
 *
 *   - It must NOT be redeemed while the visitor is signed out, or it would be
 *     consumed by nobody and the person would come back to a dead link. So an
 *     unauthenticated visitor is sent to login with the destination kept, and
 *     the redemption happens after they return.
 *   - React 18 StrictMode mounts effects twice in development. A second call
 *     with a spent token answers "invalid", which would show the failure screen
 *     for a link that had in fact just worked. The ref below makes the attempt
 *     once per token — this is not defensive coding, it is the difference
 *     between working and looking broken.
 */
export function AcceptInvite() {
  const { token } = useParams<{ token: string }>()
  const { t } = useTranslation('settings')
  const navigate = useNavigate()
  const { isAuthenticated, loading } = useAuthContext()

  const [state, setState] = useState<'working' | 'done' | 'failed'>('working')
  const [calendarName, setCalendarName] = useState('')
  const [error, setError] = useState('')
  const attempted = useRef<string | null>(null)

  useEffect(() => {
    if (loading || !token) return

    if (!isAuthenticated) {
      // Keep the destination so the link survives the round trip through login
      // (and through registration — for someone who does not have an account
      // yet, which is the case invite links exist for).
      navigate(`/login?next=${encodeURIComponent(`/i/${token}`)}`, { replace: true })
      return
    }

    if (attempted.current === token) return
    attempted.current = token

    acceptInviteLink(token)
      .then((calendar) => {
        setCalendarName(calendar.name)
        setState('done')
      })
      .catch((err) => {
        const msg = describeCalendarError(err, 'share.linkInvalid')
        setError(t(msg.key, msg.params))
        setState('failed')
      })
  }, [token, isAuthenticated, loading, navigate, t])

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 p-6">
      <div className="w-full max-w-sm rounded-lg border border-zinc-800 bg-zinc-900 p-6 text-center">
        {state === 'working' && (
          <>
            <Loader2 className="mx-auto mb-3 h-8 w-8 animate-spin text-blue-500" />
            <p className="text-sm text-zinc-400">{t('share.acceptingLink')}</p>
          </>
        )}

        {state === 'done' && (
          <>
            <CalendarDays className="mx-auto mb-3 h-8 w-8 text-green-500" />
            <p className="mb-4 text-sm text-zinc-200">
              {t('share.acceptedLink', { name: calendarName })}
            </p>
            <button
              type="button"
              onClick={() => navigate('/calendar')}
              className="w-full rounded bg-blue-600 px-3 py-2 text-sm text-white transition-colors hover:bg-blue-700"
            >
              {t('share.goToCalendar')}
            </button>
          </>
        )}

        {state === 'failed' && (
          <>
            <XCircle className="mx-auto mb-3 h-8 w-8 text-red-500" />
            <p className="mb-4 text-sm leading-snug text-zinc-300">{error}</p>
            <button
              type="button"
              onClick={() => navigate('/calendar')}
              className="w-full rounded border border-zinc-700 px-3 py-2 text-sm text-zinc-300 transition-colors hover:bg-zinc-800"
            >
              {t('share.goToCalendar')}
            </button>
          </>
        )}
      </div>
    </div>
  )
}
