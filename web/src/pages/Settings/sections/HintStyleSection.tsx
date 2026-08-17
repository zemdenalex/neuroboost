import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Lightbulb } from 'lucide-react'
import { getHintStyle, setHintStyle as persistHintStyle, type HintStyle } from '../../../lib/onboarding/hintStyle'

const STYLES = ['bubbles', 'walkthrough', 'markers'] as const

/**
 * How contextual hints are presented.
 *
 * Second section out of Settings.tsx (2026-08-14). Like the first, it shares
 * nothing with the page: the value lives in localStorage, not in user settings,
 * so it needs neither the auto-save machinery nor the profile.
 *
 * The custom event is how already-mounted hint consumers learn about the change
 * — a plain localStorage write is invisible to the tab that made it, so
 * dropping this line would make the setting appear to need a reload.
 */
export function HintStyleSection() {
  const { t } = useTranslation('settings')
  const [style, setStyle] = useState<HintStyle>(() => getHintStyle())

  const change = (next: HintStyle) => {
    setStyle(next)
    persistHintStyle(next)
    window.dispatchEvent(new CustomEvent('neuroboost-hints-style-change', { detail: next }))
  }

  return (
    <section data-hint="settings.hintStyle" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-4">
        <Lightbulb className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('hints.title')}</h2>
      </div>
      <div className="flex gap-3">
        {STYLES.map((s) => (
          <button
            key={s}
            onClick={() => change(s)}
            className={`flex-1 p-3 rounded-lg border text-sm font-mono transition-colors ${
              style === s
                ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                : 'bg-zinc-800 border-zinc-700 text-zinc-400 hover:border-zinc-600'
            }`}
          >
            {t(`hints.${s}`)}
          </button>
        ))}
      </div>
      <p className="text-xs text-zinc-500 mt-2">{t('hints.note')}</p>
    </section>
  )
}
