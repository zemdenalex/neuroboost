import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import i18n from '../../i18n'
import { useAuthContext } from '../../contexts/AuthContext'
import { api } from '../../api/client'
import { useMediaQuery } from '../../hooks/useMediaQuery'
import { showToast } from '../../components/ui/Toast'
import type { UserSettings } from '../../api/auth'
import { resolveQuickTaskSettings, type QuickTaskSettings } from '../../lib/quickTask/settings'
import { resolveRememberedScope, type RememberedScope } from '../../lib/recurrence/scope'
import { resolveReminderSettings, type ReminderSettings } from '../../lib/reminders/offsets'
import { createDebouncedSaver, type DebouncedSaver } from '../../lib/autoSave/debouncedSaver'

/** The subset of the profile this page edits. */
type ProfilePatch = { display_name?: string; timezone?: string; locale?: string }
import { ReminderOffsets } from '../../components/ReminderOffsets/ReminderOffsets'
import { CalendarsSection } from '../../components/Calendars/CalendarsSection'
import { AlertTriangle, Bell, Clock, Database, Globe, LayoutGrid, Lightbulb, LogOut, Maximize, Repeat, Sliders, Smartphone, Zap } from 'lucide-react'
import { getHintStyle, setHintStyle as persistHintStyle, type HintStyle } from '../../lib/onboarding/hintStyle'

type HeaderVariant = 'horizontal' | 'vertical'
type MobileNavType = 'bottom_tabs' | 'hamburger' | 'fab'

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
  const { t: tc } = useTranslation('common')
  const { t: tr } = useTranslation('reminders')
  const navigate = useNavigate()
  const { user, logout, updateSettings, updateProfile } = useAuthContext()
  const [quickTask, setQuickTask] = useState<QuickTaskSettings>(() => resolveQuickTaskSettings(user?.settings))
  const [recurringScope, setRecurringScope] = useState<RememberedScope>(() => resolveRememberedScope(user?.settings))
  const [reminders, setReminders] = useState<ReminderSettings>(() => resolveReminderSettings(user?.settings))
  const isMobile = useMediaQuery('(max-width: 767px)')
  const [showConfirmLogout, setShowConfirmLogout] = useState(false)
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
  const [headerStyle, setHeaderStyle] = useState<HeaderVariant>('horizontal')
  const [hintStyle, setHintStyle] = useState<HintStyle>(() => getHintStyle())
  const [mobileNav, setMobileNav] = useState<MobileNavType>('bottom_tabs')
  const [uiScale, setUIScale] = useState(100)
  const [timezone, setTimezone] = useState('Europe/Moscow')
  const [workDays, setWorkDays] = useState(['Mon', 'Tue', 'Wed', 'Thu', 'Fri'])
  const [workStart, setWorkStart] = useState('09:00')
  const [workEnd, setWorkEnd] = useState('17:00')

  // Feature toggles
  const [features, setFeatures] = useState({
    dreams: false,
    goals: false,
    projects: false,
    opportunities: false,
    needs: false,
    graph: false,
    timeline: false,
    tools: true,
  })

  // Load settings from user on mount
  useEffect(() => {
    if (user) {
      setTimezone(user.timezone || 'Europe/Moscow')
      
      if (user.settings) {
        if (user.settings.header_variant) {
          setHeaderStyle(user.settings.header_variant)
        }
        if (user.settings.mobile_nav) {
          setMobileNav(user.settings.mobile_nav)
        }
        if (user.settings.ui_scale) {
          setUIScale(user.settings.ui_scale)
        }
        if (user.settings.work_days) {
          setWorkDays(user.settings.work_days)
        }
        if (user.settings.work_start) {
          setWorkStart(user.settings.work_start)
        }
        if (user.settings.work_end) {
          setWorkEnd(user.settings.work_end)
        }
        const userFeatures = user.settings.features
        if (userFeatures) {
          setFeatures(prev => ({ ...prev, ...userFeatures }))
        }
      }
    }
  }, [user])

  // Apply UI scale to document (no cleanup — scale should persist when leaving page)
  useEffect(() => {
    document.documentElement.style.fontSize = `${uiScale}%`
  }, [uiScale])

  // Apply header style immediately when changed
  const handleHeaderStyleChange = async (style: HeaderVariant) => {
    setHeaderStyle(style)
    try {
      await updateSettings({ header_variant: style })
      showToast(t('saved'))
    } catch {
      setError(t('error.saveLayout'))
    }
  }

  // Hint style is localStorage-only (v1); broadcast so the live OnboardingProvider updates immediately.
  const handleHintStyleChange = (style: HintStyle) => {
    setHintStyle(style)
    persistHintStyle(style)
    window.dispatchEvent(new CustomEvent('neuroboost-hints-style-change', { detail: style }))
  }

  const handleMobileNavChange = async (nav: MobileNavType) => {
    setMobileNav(nav)
    try {
      await updateSettings({ mobile_nav: nav })
      showToast(t('saved'))
    } catch {
      setError(t('error.saveMobileNav'))
    }
  }

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

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const toggleFeature = (key: keyof typeof features) => {
    const nextFeatures = { ...features, [key]: !features[key] }
    setFeatures(nextFeatures)
    autoSaveSettings({ features: nextFeatures })
  }

  const handleExport = async () => {
    try {
      // api.get throws on non-2xx / 401 (and unwraps the { data } envelope), so a
      // failed export no longer downloads an "undefined" blob as if it succeeded.
      const exported = await api.get('/export')
      const blob = new Blob([JSON.stringify(exported, null, 2)], {
        type: 'application/json',
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `neuroboost-export-${new Date().toISOString().split('T')[0]}.json`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      setError(t('dataManagement.exportFailed'))
    }
  }

  const handleImport = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      try {
        const text = await file.text()
        const data = JSON.parse(text) as unknown
        // api.post throws on non-2xx / 401, so the reload only happens on a real
        // success — a failed import no longer reloads and masks itself as done.
        await api.post('/import', data)
        window.location.reload()
      } catch {
        setError(t('dataManagement.importFailed'))
      }
    }
    input.click()
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

        {/* Layout Style */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <LayoutGrid className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('layout.title')}</h2>
          </div>

          <div className="flex gap-3">
            <button
              onClick={() => handleHeaderStyleChange('horizontal')}
              className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
                headerStyle === 'horizontal'
                  ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                  : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
              }`}
            >
              {t('layout.horizontal')}
            </button>
            <button
              onClick={() => handleHeaderStyleChange('vertical')}
              className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
                headerStyle === 'vertical'
                  ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                  : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
              }`}
            >
              {t('layout.vertical')}
            </button>
          </div>
          <p className="text-xs text-zinc-500 mt-2">{t('layout.note')}</p>
        </section>

        {/* Hint Style */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Lightbulb className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('hints.title')}</h2>
          </div>
          <div className="flex gap-3">
            {(['bubbles', 'walkthrough', 'markers'] as const).map((style) => (
              <button
                key={style}
                onClick={() => handleHintStyleChange(style)}
                className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
                  hintStyle === style
                    ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                    : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
                }`}
              >
                {t(`hints.${style}`)}
              </button>
            ))}
          </div>
          <p className="text-xs text-zinc-500 mt-2">{t('hints.note')}</p>
        </section>

        {/* Mobile Navigation — only shown on mobile viewports */}
        {isMobile && (
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Smartphone className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('mobileNav.title')}</h2>
          </div>

          <div className="flex flex-col gap-3">
            <button
              onClick={() => handleMobileNavChange('bottom_tabs')}
              className={`flex items-start gap-3 p-3 rounded-lg border text-left transition-colors ${
                mobileNav === 'bottom_tabs'
                  ? 'bg-blue-600/20 border-blue-500'
                  : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
              }`}
            >
              <div className="flex-1">
                <p className={`text-sm font-mono ${mobileNav === 'bottom_tabs' ? 'text-blue-400' : 'text-zinc-300'}`}>
                  {t('mobileNav.bottomTabs')}
                </p>
                <p className="text-xs text-zinc-500 mt-0.5">
                  {t('mobileNav.bottomTabsDesc')}
                </p>
              </div>
            </button>
            <button
              onClick={() => handleMobileNavChange('hamburger')}
              className={`flex items-start gap-3 p-3 rounded-lg border text-left transition-colors ${
                mobileNav === 'hamburger'
                  ? 'bg-blue-600/20 border-blue-500'
                  : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
              }`}
            >
              <div className="flex-1">
                <p className={`text-sm font-mono ${mobileNav === 'hamburger' ? 'text-blue-400' : 'text-zinc-300'}`}>
                  {t('mobileNav.hamburger')}
                </p>
                <p className="text-xs text-zinc-500 mt-0.5">
                  {t('mobileNav.hamburgerDesc')}
                </p>
              </div>
            </button>
            <button
              onClick={() => handleMobileNavChange('fab')}
              className={`flex items-start gap-3 p-3 rounded-lg border text-left transition-colors ${
                mobileNav === 'fab'
                  ? 'bg-blue-600/20 border-blue-500'
                  : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
              }`}
            >
              <div className="flex-1">
                <p className={`text-sm font-mono ${mobileNav === 'fab' ? 'text-blue-400' : 'text-zinc-300'}`}>
                  {t('mobileNav.fab')}
                </p>
                <p className="text-xs text-zinc-500 mt-0.5">
                  {t('mobileNav.fabDesc')}
                </p>
              </div>
            </button>
          </div>
          <p className="text-xs text-zinc-500 mt-2">{t('mobileNav.note')}</p>
        </section>
        )}

        {/* UI Scale */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Maximize className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('uiScale.title')}</h2>
          </div>

          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <span className="text-sm text-zinc-400 w-8">80%</span>
              <input
                type="range"
                min="80"
                max="150"
                step="5"
                value={uiScale}
                onChange={(e) => {
                  const val = Number(e.target.value)
                  setUIScale(val)
                  autoSaveSettings({ ui_scale: val })
                }}
                className="flex-1 h-2 bg-zinc-700 rounded-lg appearance-none cursor-pointer accent-blue-600"
              />
              <span className="text-sm text-zinc-400 w-10">150%</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-zinc-400">{t('uiScale.current')} <strong className="text-white">{uiScale}%</strong></span>
              <button
                onClick={() => {
                  setUIScale(100)
                  autoSaveSettings({ ui_scale: 100 })
                }}
                className="text-xs text-blue-400 hover:text-blue-300"
              >
                {t('uiScale.reset')}
              </button>
            </div>
          </div>
        </section>

        {/* Recurring-event scope */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Repeat className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('recurringScope.title')}</h2>
          </div>

          <label className="block text-sm text-zinc-400 mb-1" htmlFor="recurring-scope">
            {t('recurringScope.label')}
          </label>
          <select
            id="recurring-scope"
            value={recurringScope}
            onChange={e => {
              const value = e.target.value as RememberedScope
              setRecurringScope(value)
              autoSaveSettings({ recurring_scope: value } as Partial<UserSettings>)
            }}
            className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          >
            <option value="ask">{t('recurringScope.ask')}</option>
            <option value="occurrence">{t('recurringScope.occurrence')}</option>
            <option value="series">{t('recurringScope.series')}</option>
          </select>
          <p className="mt-2 text-xs text-zinc-500">{t('recurringScope.hint')}</p>
        </section>

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

        {/* Reminders */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Bell className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{tr('settings.title')}</h2>
          </div>
          <p className="mb-4 text-xs text-zinc-500">{tr('settings.description')}</p>

          <div className="space-y-4">
            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-event-preset">{tr('settings.defaultEventPreset')}</label>
              <select
                id="rem-event-preset"
                value={reminders.default_event_preset}
                onChange={(e) => {
                  const value = e.target.value
                  setReminders(prev => ({ ...prev, default_event_preset: value }))
                  autoSaveSettings({ reminders: { ...reminders, default_event_preset: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              >
                {Object.keys(reminders.presets).map(name => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-task-preset">{tr('settings.defaultTaskPreset')}</label>
              <select
                id="rem-task-preset"
                value={reminders.default_task_preset}
                onChange={(e) => {
                  const value = e.target.value
                  setReminders(prev => ({ ...prev, default_task_preset: value }))
                  autoSaveSettings({ reminders: { ...reminders, default_task_preset: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500"
              >
                {Object.keys(reminders.presets).map(name => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={reminders.digest_enabled}
                onChange={(e) => {
                  const value = e.target.checked
                  setReminders(prev => ({ ...prev, digest_enabled: value }))
                  autoSaveSettings({ reminders: { ...reminders, digest_enabled: value } })
                }}
                className="mt-1 accent-blue-500"
              />
              <span className="block text-sm text-white">{tr('settings.digestEnabled')}</span>
            </label>

            <div>
              <label className="block text-sm text-zinc-400 mb-1" htmlFor="rem-digest-at">{tr('settings.digestAt')}</label>
              <input
                id="rem-digest-at"
                type="time"
                value={reminders.digest_at}
                disabled={!reminders.digest_enabled}
                onChange={(e) => {
                  const value = e.target.value
                  setReminders(prev => ({ ...prev, digest_at: value }))
                  autoSaveSettings({ reminders: { ...reminders, digest_at: value } })
                }}
                className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono text-sm focus:outline-none focus:border-blue-500 disabled:opacity-40"
              />
            </div>

            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={reminders.quiet_hours_respected}
                onChange={(e) => {
                  const value = e.target.checked
                  setReminders(prev => ({ ...prev, quiet_hours_respected: value }))
                  autoSaveSettings({ reminders: { ...reminders, quiet_hours_respected: value } })
                }}
                className="mt-1 accent-blue-500"
              />
              <span>
                <span className="block text-sm text-white">{tr('settings.quietHoursRespected')}</span>
                <span className="block text-xs text-zinc-500">{tr('settings.quietHoursHint')}</span>
              </span>
            </label>

            <div>
              <h3 className="text-sm text-zinc-400 mb-1">{tr('settings.presetsTitle')}</h3>
              <p className="mb-2 text-xs text-zinc-500">{tr('settings.presetsHint')}</p>
              <div className="space-y-3">
                {Object.entries(reminders.presets).map(([name, offsets]) => (
                  <div key={name} className="rounded-lg border border-zinc-800 p-3">
                    <div className="mb-2 font-mono text-sm text-zinc-300">{name}</div>
                    <ReminderOffsets
                      value={offsets}
                      presets={reminders.presets}
                      onChange={(next) => {
                        const presets = { ...reminders.presets, [name]: next }
                        setReminders(prev => ({ ...prev, presets }))
                        autoSaveSettings({ reminders: { ...reminders, presets } })
                      }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <CalendarsSection />

        {/* Work Hours */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Clock className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('workHours.title')}</h2>
          </div>

          <div className="space-y-4">
            {/* Working days */}
            <div>
              <label className="block text-sm text-zinc-400 mb-2">{t('workHours.days')}</label>
              <div className="flex flex-wrap gap-2">
                {(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const).map((day) => (
                  <button
                    key={day}
                    onClick={() => {
                      const next = workDays.includes(day)
                        ? workDays.filter((d) => d !== day)
                        : [...workDays, day]
                      setWorkDays(next)
                      autoSaveSettings({ work_days: next })
                    }}
                    className={`px-3 py-1.5 rounded text-sm font-mono transition-colors ${
                      workDays.includes(day)
                        ? 'bg-blue-600 text-white'
                        : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
                    }`}
                  >
                    {t(`workHours.${day}`)}
                  </button>
                ))}
              </div>
            </div>

            {/* Time range */}
            <div className="flex gap-4">
              <div className="flex-1">
                <label className="block text-sm text-zinc-400 mb-1">{t('workHours.start')}</label>
                <input
                  type="time"
                  value={workStart}
                  onChange={(e) => {
                    const val = e.target.value
                    setWorkStart(val)
                    autoSaveSettings({ work_start: val })
                  }}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex-1">
                <label className="block text-sm text-zinc-400 mb-1">{t('workHours.end')}</label>
                <input
                  type="time"
                  value={workEnd}
                  onChange={(e) => {
                    const val = e.target.value
                    setWorkEnd(val)
                    autoSaveSettings({ work_end: val })
                  }}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
          </div>
        </section>

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

        {/* Feature Toggles */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Sliders className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('featureToggles.title')}</h2>
          </div>

          <div className="grid grid-cols-2 gap-3">
            {Object.entries(features).map(([key, enabled]) => (
              <button
                key={key}
                onClick={() => toggleFeature(key as keyof typeof features)}
                className="flex items-center justify-between p-3 bg-zinc-800/50 rounded-lg hover:bg-zinc-800 transition-colors"
              >
                <span className="text-sm text-zinc-300">{t(`featureToggles.${key}View`)}</span>
                {/* Custom toggle switch */}
                <div
                  className={`relative w-10 h-5 rounded-full transition-colors ${
                    enabled ? 'bg-blue-600' : 'bg-zinc-600'
                  }`}
                >
                  <div
                    className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                      enabled ? 'translate-x-5' : 'translate-x-0.5'
                    }`}
                  />
                </div>
              </button>
            ))}
          </div>
        </section>

        {/* Data Management */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <Database className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('dataManagement.title')}</h2>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleExport}
              className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors"
            >
              {t('dataManagement.export')}
            </button>
            <button
              onClick={handleImport}
              className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors"
            >
              {t('dataManagement.import')}
            </button>
            <button className="px-4 py-2 bg-red-900/30 hover:bg-red-900/50 text-red-400 font-mono text-sm rounded-lg transition-colors border border-red-800">
              {t('dataManagement.clearAll')}
            </button>
          </div>
        </section>

        {/* Sign Out */}
        <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
          <div className="flex items-center gap-2 mb-4">
            <LogOut className="w-5 h-5 text-zinc-400" />
            <h2 className="text-lg font-mono font-semibold text-white">{t('session.title')}</h2>
          </div>

          {showConfirmLogout ? (
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
                onClick={() => setShowConfirmLogout(false)}
                className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 text-white text-sm font-mono rounded transition-colors"
              >
                {tc('action.cancel')}
              </button>
            </div>
          ) : (
            <button
              onClick={() => setShowConfirmLogout(true)}
              className="flex items-center gap-2 px-3 py-2 text-red-400 hover:bg-zinc-800 rounded-lg transition-colors"
            >
              <LogOut className="w-4 h-4" />
              {t('session.signOut')}
            </button>
          )}
        </section>
    </div>
  )
}