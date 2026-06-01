import { Play, Pause } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX, MODE_BTN } from './formatTime'
import { durationMsForMode } from '../../lib/pomodoro/machine'

export function BarWidget() {
  const { t } = useTranslation('tools')
  const { phase, remainingMs, isRunning, settings, start, pause } = usePomodoro()
  const full = durationMsForMode(phase, settings)
  const pct = full > 0 ? Math.min(100, Math.max(0, ((full - remainingMs) / full) * 100)) : 0
  return (
    <div className="fixed left-1/2 top-2 z-40 flex -translate-x-1/2 items-center gap-3 rounded-full border border-zinc-700 bg-zinc-900/95 py-1.5 pl-4 pr-2 shadow-xl backdrop-blur">
      <span className="text-[11px] font-bold" style={{ color: MODE_HEX[phase] }}>
        {t(`pomodoro.${phase}`)}
      </span>
      <span className="relative h-1 w-20 overflow-hidden rounded bg-zinc-700">
        <span className="absolute left-0 top-0 h-full" style={{ width: `${pct}%`, background: MODE_HEX[phase] }} />
      </span>
      <span className="font-mono text-sm font-bold tabular-nums text-zinc-100">{formatMs(remainingMs)}</span>
      <button
        onClick={isRunning ? pause : start}
        className={`grid h-6 w-6 place-items-center rounded-full text-white ${MODE_BTN[phase]}`}
        aria-label={isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
      >
        {isRunning ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
      </button>
    </div>
  )
}
