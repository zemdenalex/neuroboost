import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Globe } from 'lucide-react'
import i18n from '../../../i18n'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'

const LANGUAGES = [
  { value: 'en', label: 'English', flag: '\u{1F1EC}\u{1F1E7}' },
  { value: 'ru', label: '\u0420\u0443\u0441\u0441\u043A\u0438\u0439', flag: '\u{1F1F7}\u{1F1FA}' },
]

// Copied verbatim from Settings.tsx. 🔴 Not curated during the move: my first
// draft of this file invented a different list — Russian cities, no London or
// New York — which would have silently changed what the user can pick. A
// refactor that edits data while relocating it is the hardest kind of change to
// find later, because the diff reads as a move.
//
// The backend does NOT constrain the value to this list: validTimezone checks
// it against the system tz database (auth/validation.go). The list is purely
// what the UI offers.
const TIMEZONES = [
  { value: 'Europe/Moscow', label: 'Europe/Moscow (MSK)', offset: '+3' },
  { value: 'Europe/London', label: 'Europe/London (GMT)', offset: '+0' },
  { value: 'Europe/Berlin', label: 'Europe/Berlin (CET)', offset: '+1' },
  { value: 'America/New_York', label: 'America/New_York (EST)', offset: '-5' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST)', offset: '-8' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST)', offset: '+9' },
  { value: 'Asia/Shanghai', label: 'Asia/Shanghai (CST)', offset: '+8' },
  { value: 'UTC', label: 'UTC', offset: '+0' },
]

interface Props {
  /**
   * The page's debounced profile saver. Timezone goes through it because the
   * select can be scrolled with a keyboard, firing a change per option.
   */
  autoSaveProfile: (patch: { display_name?: string; timezone?: string; locale?: string }) => void
}

/**
 * Timezone and interface language.
 *
 * Step 6 of the Settings split (2026-08-14). This was the only consumer of the
 * profile auto-save, so the parent keeps that saver solely to hand it here —
 * everything else on the page saves settings, not profile.
 *
 * 🔴 Language does NOT go through the debounce. It is applied immediately and
 * saved immediately, because i18n.changeLanguage takes effect at once and a
 * 300ms window between "the UI is now English" and "the server knows" is a
 * window in which a refresh silently reverts it.
 *
 * The localStorage write is a fallback for a cold open before the profile
 * loads. ⚠ It does not decide anything on its own: AuthContext applies
 * `user.locale` over it on mount, which is what made the e2e suite look for
 * English labels on a Russian page for months.
 */
export function RegionalSection({ autoSaveProfile }: Props) {
  const { t } = useTranslation('settings')
  const { user, updateProfile } = useAuthContext()

  const [timezone, setTimezone] = useState(user?.timezone || 'Europe/Moscow')
  const [language, setLanguage] = useState(i18n.language?.startsWith('ru') ? 'ru' : 'en')

  useEffect(() => {
    if (user) setTimezone(user.timezone || 'Europe/Moscow')
  // 🔴 Keyed on the account, not on the `user` OBJECT. updateSettings calls
  // setUser after every save, so a dependency on `user` re-ran this on every
  // server response — including one answering an EARLIER save, which then
  // overwrote a change the user had made in the meantime. On its own that
  // looked like a flicker; it lost the change for good as soon as the next
  // edit was built from the reverted state. Reproduced in
  // web/e2e/settings-race.spec.ts (expected 07:11, got the stored 08:00).
  //
  // Cost of the narrower key: settings changed on another device no longer
  // appear without a reload. They did not appear reliably before either — this
  // effect only ever fired on this tab's own saves.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const changeLanguage = async (locale: string) => {
    setLanguage(locale)
    i18n.changeLanguage(locale)
    localStorage.setItem('neuroboost-locale', locale)
    try {
      await updateProfile({ locale })
      showToast(t('saved'))
    } catch {
      showToast(t('error.languageFailed'))
    }
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Globe className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('regional.title')}</h2>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-sm text-zinc-400 mb-1" htmlFor="regional-tz">
            {t('regional.timezone')}
          </label>
          <div className="relative">
            <select
              id="regional-tz"
              value={timezone}
              onChange={(e) => {
                setTimezone(e.target.value)
                autoSaveProfile({ timezone: e.target.value })
              }}
              className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500 appearance-none pr-10"
            >
              {TIMEZONES.map((tz) => (
                <option key={tz.value} value={tz.value}>
                  {tz.label}
                </option>
              ))}
            </select>
            <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-zinc-400">▼</div>
          </div>
        </div>

        <div>
          <label className="block text-sm text-zinc-400 mb-1">{t('regional.language')}</label>
          <div className="flex gap-3">
            {LANGUAGES.map((lang) => (
              <button
                key={lang.value}
                onClick={() => changeLanguage(lang.value)}
                className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border text-sm font-mono transition-colors ${
                  language === lang.value
                    ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                    : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
                }`}
              >
                <span className="text-lg">{lang.flag}</span>
                <span>{lang.label}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
