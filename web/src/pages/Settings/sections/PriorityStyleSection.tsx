import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Flag } from 'lucide-react'
import { useAuthContext } from '../../../contexts/AuthContext'
import { showToast } from '../../../components/ui/Toast'
import { PRIORITY_STYLES, priorityMark, readPriorityStyle, type PriorityStyle } from '../../../lib/priorityStyle'

/**
 * How priority is drawn — the same setting as the bot's ⚙️ (settings.bot.
 * priority_style; Denis 26.09: the web follows it, gap list row 16). Each
 * option previews itself with the real marks, as the bot's buttons do.
 */
function Preview({ style }: { style: PriorityStyle }) {
  return (
    <span className="flex items-center gap-1.5" aria-hidden>
      {[1, 2, 3].map((p) => {
        const m = priorityMark(style, p)
        return m.kind === 'dot'
          ? <span key={p} className={`inline-block w-2.5 h-2.5 rounded-full ${m.className}`} />
          : <span key={p} className="font-mono text-xs text-zinc-400">{m.text}</span>
      })}
    </span>
  )
}

export function PriorityStyleSection() {
  const { t } = useTranslation('settings')
  const { user, updateBotSetting } = useAuthContext()
  const [choice, setChoice] = useState<PriorityStyle>(readPriorityStyle(user?.settings))

  useEffect(() => {
    setChoice(readPriorityStyle(user?.settings))
    // Keyed on the account, not the user object (MobileNavSection's race).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  const change = async (next: PriorityStyle) => {
    setChoice(next)
    try {
      await updateBotSetting('priority_style', next)
      showToast(t('saved'))
    } catch {
      showToast(t('priorityStyle.error'))
    }
  }

  return (
    <section data-testid="settings-priority-style" className="bg-zinc-900 border border-zinc-800 rounded-lg p-5">
      <div className="flex items-center gap-2 mb-1">
        <Flag className="w-5 h-5 text-zinc-400" />
        <h2 className="text-lg font-mono font-semibold text-white">{t('priorityStyle.title')}</h2>
      </div>
      <p className="text-xs text-zinc-500 mb-4">{t('priorityStyle.hint')}</p>
      <div className="flex flex-col gap-2">
        {PRIORITY_STYLES.map((style) => (
          <button
            key={style}
            data-testid={`priority-style-${style}`}
            aria-pressed={choice === style}
            onClick={() => void change(style)}
            className={`flex items-center justify-between gap-3 p-3 rounded-lg border text-left transition-colors ${
              choice === style ? 'bg-blue-600/20 border-blue-500' : 'bg-zinc-800 border-zinc-700 hover:border-zinc-600'
            }`}
          >
            <span className={`text-sm font-mono ${choice === style ? 'text-blue-400' : 'text-zinc-300'}`}>{t(`priorityStyle.${style}`)}</span>
            <Preview style={style} />
          </button>
        ))}
      </div>
    </section>
  )
}
