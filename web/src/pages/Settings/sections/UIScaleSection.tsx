import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Maximize } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import type { UserSettings } from '../../../api/auth'

const MIN = 80
const MAX = 150
const STEP = 5
const DEFAULT = 100

interface Props {
  /**
   * The page's debounced saver, passed in rather than created here.
   *
   * 🔴 Deliberate: the flush-on-unmount that stops a change being lost belongs
   * to the page, and one saver per section would mean one flush per section —
   * several requests where the user made one edit. A slider is also the reason
   * the debounce exists at all: dragging it fires dozens of changes that must
   * coalesce into a single save.
   */
  autoSave: (patch: Partial<UserSettings>) => void
}

/**
 * Interface size, as a percentage applied to the document's root font size.
 *
 * Step 8 of the Settings split (2026-08-14). The effect that writes
 * `document.documentElement.style.fontSize` moves here with the value it
 * depends on — it was one of the parent's effects, three hundred lines away
 * from the slider that drives it.
 *
 * ⚠ That effect has no cleanup, on purpose: the scale must survive leaving the
 * page. Restoring it on unmount would reset the whole interface the moment the
 * user navigates away from Settings.
 */
export function UIScaleSection({ autoSave }: Props) {
  const { t } = useTranslation('settings')
  const { user } = useAuthContext()
  const [scale, setScale] = useState(user?.settings?.ui_scale ?? DEFAULT)

  useEffect(() => {
    if (user?.settings?.ui_scale) setScale(user.settings.ui_scale)
  }, [user])

  useEffect(() => {
    document.documentElement.style.fontSize = `${scale}%`
  }, [scale])

  const change = (next: number) => {
    setScale(next)
    autoSave({ ui_scale: next })
  }

  return (
    <section className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Maximize className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('uiScale.title')}</h2>
      </div>

      <div className="space-y-4">
        <div className="flex items-center gap-4">
          <span className="text-sm text-zinc-400 w-8">{MIN}%</span>
          <input
            type="range"
            min={MIN}
            max={MAX}
            step={STEP}
            value={scale}
            onChange={(e) => change(Number(e.target.value))}
            className="flex-1 h-2 bg-zinc-700 rounded-lg appearance-none cursor-pointer accent-blue-600"
          />
          <span className="text-sm text-zinc-400 w-10">{MAX}%</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-zinc-400">
            {t('uiScale.current')} <strong className="text-white">{scale}%</strong>
          </span>
          <button onClick={() => change(DEFAULT)} className="text-xs text-blue-400 hover:text-blue-300">
            {t('uiScale.reset')}
          </button>
        </div>
      </div>
    </section>
  )
}
