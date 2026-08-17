import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { LogOut, AlertTriangle } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'

/**
 * Sign out, with a confirmation step.
 *
 * First section moved out of the 866-line Settings.tsx on 2026-08-14, and
 * first because it shares nothing with the rest of the page: no settings
 * state, no auto-save, no error banner. It owns its own confirmation flag,
 * which is why that flag no longer sits among the parent's twenty-odd
 * useStates.
 *
 * Pattern follows components/Calendars/CalendarsSection.tsx — the section reads
 * what it needs from context itself and the parent mounts it with one line.
 */
export function SessionSection() {
  const { t } = useTranslation('settings')
  const { t: tc } = useTranslation('common')
  const { logout } = useAuthContext()
  const navigate = useNavigate()
  const [showConfirm, setShowConfirm] = useState(false)

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <LogOut className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('session.title')}</h2>
      </div>

      {showConfirm ? (
        <div className="flex items-center gap-3 p-3 bg-red-900/20 border border-red-800 rounded-lg">
          <AlertTriangle className="w-5 h-5 text-red-400 shrink-0" />
          <span className="flex-1 text-sm text-red-400">{t('session.signOutConfirm')}</span>
          <button
            onClick={handleLogout}
            className="px-3 py-1 bg-red-600 hover:bg-red-700 text-white text-sm font-mono rounded transition-colors"
          >
            {t('session.yesSignOut')}
          </button>
          <button
            onClick={() => setShowConfirm(false)}
            className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 text-white text-sm font-mono rounded transition-colors"
          >
            {tc('action.cancel')}
          </button>
        </div>
      ) : (
        <button
          onClick={() => setShowConfirm(true)}
          className="flex items-center gap-2 px-3 py-2 text-red-400 hover:bg-zinc-800 rounded-lg transition-colors"
        >
          <LogOut className="w-4 h-4" />
          {t('session.signOut')}
        </button>
      )}
    </section>
  )
}
