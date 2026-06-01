import { Play, Pause } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX } from './formatTime'

export function PillWidget() {
  const { t } = useTranslation('tools')
  const { phase, remainingMs, isRunning, start, pause } = usePomodoro()
  return (
    <div
      className="fixed bottom-4 right-4 z-40 flex items-center gap-2 rounded-full bg-zinc-900/95 px-3 py-2 shadow-xl backdrop-blur"
      style={{ border: `1px solid ${MODE_HEX[phase]}` }}
    >
      <Link to="/tools/pomodoro" className="flex items-center gap-2">
        <span className="h-2.5 w-2.5 rounded-full" style={{ background: MODE_HEX[phase] }} />
        <span className="font-mono text-base font-bold tabular-nums text-zinc-100">{formatMs(remainingMs)}</span>
      </Link>
      <button
        onClick={isRunning ? pause : start}
        className="grid h-6 w-6 place-items-center rounded-full bg-zinc-800 text-zinc-300 hover:text-zinc-100"
        aria-label={isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
      >
        {isRunning ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
      </button>
    </div>
  )
}
