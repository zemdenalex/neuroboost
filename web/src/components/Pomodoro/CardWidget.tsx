import { Play, Pause, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePomodoro } from '../../contexts/PomodoroContext'
import { formatMs, MODE_HEX, MODE_BTN } from './formatTime'

export function CardWidget() {
  const { t } = useTranslation('tools')
  const { phase, remainingMs, isRunning, sessionsCompleted, linkedTaskTitle, settings, start, pause, skip } =
    usePomodoro()
  const filled = sessionsCompleted % settings.sessionsBeforeLong
  return (
    <div
      className="fixed bottom-4 right-4 z-40 w-56 rounded-xl bg-zinc-900/95 p-3 shadow-2xl backdrop-blur"
      style={{ border: '1px solid #27272a', borderLeft: `3px solid ${MODE_HEX[phase]}` }}
    >
      <div className="mb-2 flex items-center justify-between">
        <span className="text-[10px] font-bold uppercase tracking-wide" style={{ color: MODE_HEX[phase] }}>
          {t(`pomodoro.${phase}`)}
        </span>
        <span className="flex gap-1">
          {Array.from({ length: settings.sessionsBeforeLong }).map((_, i) => (
            <span
              key={i}
              className="h-1.5 w-1.5 rounded-full"
              style={{ background: i < filled ? MODE_HEX.work : '#3f3f46' }}
            />
          ))}
        </span>
      </div>
      <div className="font-mono text-3xl font-extrabold tabular-nums text-zinc-100">{formatMs(remainingMs)}</div>
      {linkedTaskTitle && <div className="mb-2 truncate text-xs text-zinc-400">↳ {linkedTaskTitle}</div>}
      <div className="mt-2 flex gap-2">
        <button
          onClick={isRunning ? pause : start}
          className={`flex h-8 flex-[1.6] items-center justify-center rounded-lg text-white ${MODE_BTN[phase]}`}
          aria-label={isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
        >
          {isRunning ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
        </button>
        <button
          onClick={skip}
          className="flex h-8 flex-1 items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300"
          aria-label={t('pomodoro.skip')}
        >
          <SkipForward className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  )
}
