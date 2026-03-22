import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import i18n from '../../i18n'
import { useAuthContext } from '../../contexts/AuthContext'
import {
  Clock,
  Globe,
  Sliders,
  Database,
  LogOut,
  AlertTriangle,
  Save,
  Loader2,
  Check,
  LayoutGrid,
  Maximize,
  Smartphone,
} from 'lucide-react'

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
  const navigate = useNavigate()
  const { user, logout, updateSettings, updateProfile } = useAuthContext()
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [showConfirmLogout, setShowConfirmLogout] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [language, setLanguage] = useState(i18n.language?.startsWith('ru') ? 'ru' : 'en')

  // Settings state - initialize from user settings or defaults
  const [headerStyle, setHeaderStyle] = useState<HeaderVariant>('horizontal')
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

  // Apply UI scale to document
  useEffect(() => {
    document.documentElement.style.fontSize = `${uiScale}%`
    return () => {
      document.documentElement.style.fontSize = '100%'
    }
  }, [uiScale])

  // Apply header style immediately when changed
  const handleHeaderStyleChange = async (style: HeaderVariant) => {
    setHeaderStyle(style)
    try {
      await updateSettings({ header_variant: style })
    } catch {
      setError(t('error.saveLayout'))
    }
  }

  const handleMobileNavChange = async (nav: MobileNavType) => {
    setMobileNav(nav)
    try {
      await updateSettings({ mobile_nav: nav })
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
    } catch {
      setError(t('error.languageFailed'))
    }
  }

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    
    try {
      // Save profile and settings in parallel
      await Promise.all([
        updateProfile({ timezone }),
        updateSettings({
          header_variant: headerStyle,
          mobile_nav: mobileNav,
          ui_scale: uiScale,
          work_days: workDays,
          work_start: workStart,
          work_end: workEnd,
          features,
        }),
      ])

      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError(t('error.saveSettings'))
    } finally {
      setSaving(false)
    }
  }

  const toggleFeature = (key: keyof typeof features) => {
    setFeatures(prev => ({ ...prev, [key]: !prev[key] }))
  }

  return (
    <div className="max-w-3xl mx-auto p-6 space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-mono font-bold text-white">{t('title')}</h1>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 text-white font-mono text-sm rounded-lg transition-colors"
          >
            {saving ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : saved ? (
              <Check className="w-4 h-4" />
            ) : (
              <Save className="w-4 h-4" />
            )}
            {saved ? t('saved') : t('saveChanges')}
          </button>
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

        {/* Mobile Navigation */}
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
                onChange={(e) => setUIScale(Number(e.target.value))}
                className="flex-1 h-2 bg-zinc-700 rounded-lg appearance-none cursor-pointer accent-blue-600"
              />
              <span className="text-sm text-zinc-400 w-10">150%</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-zinc-400">{t('uiScale.current')} <strong className="text-white">{uiScale}%</strong></span>
              <button
                onClick={() => setUIScale(100)}
                className="text-xs text-blue-400 hover:text-blue-300"
              >
                {t('uiScale.reset')}
              </button>
            </div>
          </div>
        </section>

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
              <div className="flex gap-2">
                {(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'] as const).map((day) => (
                  <button
                    key={day}
                    onClick={() =>
                      setWorkDays((prev) =>
                        prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
                      )
                    }
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
                  onChange={(e) => setWorkStart(e.target.value)}
                  className="w-full px-3 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white font-mono focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex-1">
                <label className="block text-sm text-zinc-400 mb-1">{t('workHours.end')}</label>
                <input
                  type="time"
                  value={workEnd}
                  onChange={(e) => setWorkEnd(e.target.value)}
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
                  onChange={(e) => setTimezone(e.target.value)}
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
            <button className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors">
              {t('dataManagement.export')}
            </button>
            <button className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 font-mono text-sm rounded-lg transition-colors">
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