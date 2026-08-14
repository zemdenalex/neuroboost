import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import i18n from '../../i18n'
import { useAuthContext } from '../../contexts/AuthContext'
import { useMediaQuery } from '../../hooks/useMediaQuery'
import { showToast } from '../../components/ui/Toast'
import type { UserSettings } from '../../api/auth'
import { resolveQuickTaskSettings, type QuickTaskSettings } from '../../lib/quickTask/settings'
import { createDebouncedSaver, type DebouncedSaver } from '../../lib/autoSave/debouncedSaver'
import { SessionSection } from './sections/SessionSection'
import { HintStyleSection } from './sections/HintStyleSection'
import { LayoutStyleSection } from './sections/LayoutStyleSection'
import { MobileNavSection } from './sections/MobileNavSection'
import { UIScaleSection } from './sections/UIScaleSection'
import { WorkHoursSection } from './sections/WorkHoursSection'
import { FeatureTogglesSection } from './sections/FeatureTogglesSection'
import { RecurringScopeSection } from './sections/RecurringScopeSection'
import { DataSection } from './sections/DataSection'
import { RemindersSection } from './sections/RemindersSection'

/** The subset of the profile this page edits. */
type ProfilePatch = { display_name?: string; timezone?: string; locale?: string }
import { CalendarsSection } from '../../components/Calendars/CalendarsSection'
import { Globe, Zap } from 'lucide-react'


const LANGUAGES = [
  { value: 'en', label: 'English', flag: '\u{1F1EC}\u{1F1E7}' },
  { value: 'ru', label: '\u0420\u0443\u0441\u0441\u043A\u0438\u0439', flag: '\u{1F1F7}\u{1F1FA}' },
]

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

export default function Settings() {
  const { t } = useTranslation('settings')
  const { user, updateSettings, updateProfile } = useAuthContext()
  const [quickTask, setQuickTask] = useState<QuickTaskSettings>(() => resolveQuickTaskSettings(user?.settings))
  const isMobile = useMediaQuery('(max-width: 767px)')
  const [error, setError] = useState<string | null>(null)
  const [language, setLanguage] = useState(i18n.language?.startsWith('ru') ? 'ru' : 'en')

  // Debounced auto-save. The machinery lives in lib/autoSave so it can be
  // tested: both copies that used to sit here cancelled their timer on unmount
  // under a comment claiming they flushed it, so a change followed by leaving
  // the page inside 300ms was lost silently. See debouncedSaver.test.ts.
  //
  // Held in refs and built once: rebuilding the saver on re-render would drop
  // the pending patch it was holding.
  const settingsSaverRef = useRef<DebouncedSaver<UserSettings> | null>(null)
  const profileSaverRef = useRef<DebouncedSaver<ProfilePatch> | null>(null)

  // Callbacks change identity every render; the savers are created once, so
  // they read through this ref rather than closing over the first render's
  // versions.
  const handlersRef = useRef({ updateSettings, updateProfile, showToast, setError, t })
  handlersRef.current = { updateSettings, updateProfile, showToast, setError, t }

  if (settingsSaverRef.current === null) {
    settingsSaverRef.current = createDebouncedSaver<UserSettings>({
      save: (patch) => handlersRef.current.updateSettings(patch),
      onSaved: () => handlersRef.current.showToast(handlersRef.current.t('saved')),
      onError: (err) => {
        console.error('Auto-save failed:', err)
        handlersRef.current.setError(handlersRef.current.t('error.saveSettings'))
      },
      onFlushError: (err) => console.error('Auto-save flush failed:', err),
    })
  }

  if (profileSaverRef.current === null) {
    profileSaverRef.current = createDebouncedSaver<ProfilePatch>({
      save: (patch) => handlersRef.current.updateProfile(patch),
      onSaved: () => handlersRef.current.showToast(handlersRef.current.t('saved')),
      onError: (err) => {
        console.error('Auto-save failed:', err)
        handlersRef.current.setError(handlersRef.current.t('error.saveSettings'))
      },
      onFlushError: (err) => console.error('Auto-save flush failed:', err),
    })
  }

  const autoSaveSettings = useCallback((patch: Partial<UserSettings>) => {
    settingsSaverRef.current?.schedule(patch)
  }, [])

  const autoSaveProfile = useCallback((patch: Partial<ProfilePatch>) => {
    profileSaverRef.current?.schedule(patch)
  }, [])

  // 🔴 Flush, not cancel. This is the whole point of the extraction: what the
  // old comment here promised and the old body did not do.
  useEffect(() => {
    return () => {
      settingsSaverRef.current?.flush()
      profileSaverRef.current?.flush()
    }
  }, [])

  // Settings state - initialize from user settings or defaults
  const [timezone, setTimezone] = useState('Europe/Moscow')

  // Feature toggles

  // All that is left of the effect that used to resync eight fields here and
  // silently skip three others. Every section now reads its own slice, so there
  // is nothing to keep in step — timezone is simply the last value the parent
  // still owns.
  useEffect(() => {
    if (user) setTimezone(user.timezone || 'Europe/Moscow')
  }, [user])

  // Apply header style immediately when changed

  // Hint style is localStorage-only (v1); broadcast so the live OnboardingProvider updates immediately.

  const handleLanguageChange = async (locale: string) => {
    setLanguage(locale)
    i18n.changeLanguage(locale)
    localStorage.setItem('neuroboost-locale', locale)
    try {
      await updateProfile({ locale })
      showToast(t('saved'))
    } catch {
      setError(t('error.languageFailed'))
    }
  }



  return (
    <div className="max-w-3xl mx-auto p-6 space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-mono font-bold text-white">{t('title')}</h1>
        </div>

        {error && (
          <div className="p-3 bg-red-900/30 border border-red-800 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        <LayoutStyleSection />

        <HintStyleSection />

        {/* The isMobile gate stays here: it decides whether the section exists. */}
        {isMobile && <MobileNavSection />}

        <UIScaleSection autoSave={autoSaveSettings} />

        <RecurringScopeSection autoSave={autoSaveSettings} />

        {/* Quick task defaults */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('quickTask.title')}</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-due">{t('quickTask.due')}</label>
              <select
                id="qt-due"
                value={quickTask.default_due}
                onChange={(e) => {
                  const value = e.target.value as QuickTaskSettings['default_due']
                  setQuickTask(prev => ({ ...prev, default_due: value }))
                  autoSaveSettings({ quick_task: { ...quickTask, default_due: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              >
                <option value="tomorrow">{t('quickTask.dueTomorrow')}</option>
                <option value="today">{t('quickTask.dueToday')}</option>
                <option value="none">{t('quickTask.dueNone')}</option>
              </select>
            </div>

            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-priority">{t('quickTask.priority')}</label>
              <select
                id="qt-priority"
                value={quickTask.default_priority}
                onChange={(e) => {
                  const value = Number(e.target.value)
                  setQuickTask(prev => ({ ...prev, default_priority: value }))
                  autoSaveSettings({ quick_task: { ...quickTask, default_priority: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              >
                {[1, 2, 3, 4, 5].map(level => (
                  <option key={level} value={level}>{level}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="qt-estimate">{t('quickTask.estimate')}</label>
              <input
                id="qt-estimate"
                type="number"
                min="1"
                value={quickTask.default_estimate_minutes ?? ''}
                placeholder={t('quickTask.estimateNone')}
                onChange={(e) => {
                  const value = e.target.value === '' ? null : Number(e.target.value)
                  setQuickTask(prev => ({ ...prev, default_estimate_minutes: value }))
                  autoSaveSettings({ quick_task: { ...quickTask, default_estimate_minutes: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              />
              <p className="mt-1 text-xs text-zinc-500">{t('quickTask.estimateHint')}</p>
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={quickTask.inherit_filters}
                onChange={(e) => {
                  const value = e.target.checked
                  setQuickTask(prev => ({ ...prev, inherit_filters: value }))
                  autoSaveSettings({ quick_task: { ...quickTask, inherit_filters: value } })
                }}
                className="mt-1 accent-blue-500"
              />
              <span>
                <span className="block text-sm text-white">{t('quickTask.inherit')}</span>
                <span className="block text-xs text-zinc-500">{t('quickTask.inheritHint')}</span>
              </span>
            </label>
          </div>
        </section>

        <RemindersSection autoSave={autoSaveSettings} />

        <CalendarsSection />

        <WorkHoursSection autoSave={autoSaveSettings} />

        {/* Timezone */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Globe className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('regional.title')}</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm text-zinc-400 mb-1">{t('regional.timezone')}</label>
              <div className="relative">
                <select
                  value={timezone}
                  onChange={(e) => {
                    const val = e.target.value
                    setTimezone(val)
                    autoSaveProfile({ timezone: val })
                  }}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500 appearance-none pr-10"
                >
                  {TIMEZONES.map((tz) => (
                    <option key={tz.value} value={tz.value}>
                      {tz.label}
                    </option>
                  ))}
                </select>
                <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-zinc-400">
                  ▼
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm text-zinc-400 mb-1">{t('regional.language')}</label>
              <div className="flex gap-3">
                {LANGUAGES.map((lang) => (
                  <button
                    key={lang.value}
                    onClick={() => handleLanguageChange(lang.value)}
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

        <FeatureTogglesSection autoSave={autoSaveSettings} />

        <DataSection />

        <SessionSection />
    </div>
  )
}