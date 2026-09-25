import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Pin } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { readDayPrefs, type DayPrefs } from '../../../lib/dayTasks/dayView'
import type { UserSettings } from '../../../api/auth'

interface Props {
  autoSave: (patch: Partial<UserSettings>) => void
}

const TARGETS = [3, 4, 5, 6, 7] as const

/**
 * ⚙️ → 📌 Day tasks: the same three keys the bot writes (spec 2026-09-22 §11).
 * Off leaves only the switch, as in the bot.
 */
export function DayTasksSection({ autoSave }: Props) {
  const { t } = useTranslation('daytasks')
  const { user } = useAuthContext()
  const [prefs, setPrefs] = useState<DayPrefs>(() => readDayPrefs(user?.settings))

  // Keyed on the account, not the user object: see FeatureTogglesSection.
  useEffect(() => {
    setPrefs(readDayPrefs(user?.settings))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const set = (next: DayPrefs, patch: Partial<UserSettings>) => {
    setPrefs(next)
    autoSave(patch)
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Pin className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('settings.title')}</h2>
      </div>

      <Toggle
        label={t('settings.enabled')}
        hint={t('settings.enabledHint')}
        on={prefs.enabled}
        onClick={() => set({ ...prefs, enabled: !prefs.enabled }, { day_tasks_enabled: !prefs.enabled })}
      />

      {prefs.enabled && (
        <>
          <div className="mt-4">
            <div className="text-sm text-zinc-300">{t('settings.target')}</div>
            <div className="flex gap-2 mt-2" role="radiogroup" aria-label={t('settings.target')}>
              {TARGETS.map((n) => (
                <button
                  key={n}
                  role="radio"
                  aria-checked={prefs.target === n}
                  onClick={() => set({ ...prefs, target: n }, { day_tasks_target: n })}
                  className={`w-10 h-9 rounded-md text-sm font-mono transition-colors ${
                    prefs.target === n ? 'bg-blue-600 text-onaccent' : 'bg-zinc-800 text-zinc-300 hover:bg-zinc-700'
                  }`}
                >
                  {n}
                </button>
              ))}
            </div>
            <p className="text-xs text-zinc-500 mt-1">{t('settings.targetHint')}</p>
          </div>

          <div className="mt-4">
            <Toggle
              label={t('settings.paintBefore')}
              hint={t('settings.paintBeforeHint')}
              on={prefs.paintBefore}
              onClick={() =>
                set({ ...prefs, paintBefore: !prefs.paintBefore }, { day_tasks_paint_before: !prefs.paintBefore })
              }
            />
          </div>
        </>
      )}
    </section>
  )
}

function Toggle({ label, hint, on, onClick }: { label: string; hint: string; on: boolean; onClick: () => void }) {
  return (
    <button
      role="switch"
      aria-checked={on}
      onClick={onClick}
      className="w-full flex items-center justify-between gap-3 p-3 bg-zinc-800/50 rounded-lg hover:bg-zinc-800 transition-colors text-left"
    >
      <span>
        <span className="block text-sm text-zinc-300">{label}</span>
        <span className="block text-xs text-zinc-500 mt-0.5">{hint}</span>
      </span>
      <span className={`relative shrink-0 w-10 h-5 rounded-full transition-colors ${on ? 'bg-blue-600' : 'bg-zinc-600'}`}>
        <span
          className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
            on ? 'translate-x-5' : 'translate-x-0.5'
          }`}
        />
      </span>
    </button>
  )
}
