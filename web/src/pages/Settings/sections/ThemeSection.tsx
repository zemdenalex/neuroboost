import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Palette } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'
import { applyCurrentTheme, readThemeChoice, storeThemeChoice, THEME_CHOICES, type ThemeChoice } from '../../../lib/theme/theme'

/**
 * Dark, light or the device's (Denis 25.09: «dark light system is fine»; more
 * themes later). Inside the Telegram Mini App Telegram's own theme wins, which
 * the hint under the options says.
 */
export function ThemeSection() {
  const { t } = useTranslation('settings')
  const { user, updateSettings } = useAuthContext()
  const [choice, setChoice] = useState<ThemeChoice>(readThemeChoice(user?.settings))

  useEffect(() => {
    setChoice(readThemeChoice(user?.settings))
    // Keyed on the account (see MobileNavSection: a dependency on `user` let an
    // earlier save's answer overwrite a newer change).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const change = async (next: ThemeChoice) => {
    setChoice(next)
    storeThemeChoice(next)
    applyCurrentTheme(next)
    try {
      await updateSettings({ theme: next })
      showToast(t('saved'))
    } catch {
      showToast(t('error.saveTheme'))
    }
  }

  return (
    <section data-testid="settings-theme" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Palette className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('theme.title')}</h2>
      </div>
      <div className="grid grid-cols-3 gap-3">
        {THEME_CHOICES.map((value) => (
          <button
            key={value}
            data-testid={`theme-option-${value}`}
            onClick={() => change(value)}
            aria-pressed={choice === value}
            className={`p-3 rounded-lg border text-sm font-mono transition-colors ${
              choice === value
                ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                : 'bg-zinc-800 border-zinc-700 text-zinc-300 hover:border-zinc-600'
            }`}
          >
            {t(`theme.${value}`)}
          </button>
        ))}
      </div>
      <p className="text-xs text-zinc-500 mt-3">{t('theme.telegramNote')}</p>
    </section>
  )
}
