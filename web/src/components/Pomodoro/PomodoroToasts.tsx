import { useTranslation } from 'react-i18next'
import { Check, X } from 'lucide-react'
import { usePomodoro } from '../../contexts/PomodoroContext'

/**
 * Global Pomodoro toast stack. Rendered once at the app layout level so the
 * completion/undo, break-over, and interrupted notes appear on every page —
 * not just the Pomodoro tool page. Completions are queued (newest at the
 * bottom) so a second off-page completion never drops the first's Undo handle.
 */
export function PomodoroToasts() {
  const { t } = useTranslation('tools')
  const {
    completions,
    breakOver,
    interruptedWhileAway,
    start,
    dismissCompletion,
    undoCompletion,
    dismissBreakOver,
    dismissInterrupted,
  } = usePomodoro()

  if (!interruptedWhileAway && !breakOver && completions.length === 0) return null

  return (
    <div className="pointer-events-none fixed bottom-4 left-1/2 z-50 flex w-full max-w-sm -translate-x-1/2 flex-col items-center gap-2 px-4">
      {/* Interrupted note */}
      {interruptedWhileAway && (
        <div className="pointer-events-auto flex items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl">
          <span className="text-sm text-zinc-300">{t('pomodoro.interrupted')}</span>
          <button onClick={dismissInterrupted} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Break-over toast */}
      {breakOver && (
        <div className="pointer-events-auto flex items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl">
          <span className="text-sm text-zinc-300">{t('pomodoro.breakOver')}</span>
          <button onClick={() => { dismissBreakOver(); start() }} className="rounded-lg bg-red-600 px-3 py-1 text-sm font-semibold text-white">
            {t('pomodoro.start')}
          </button>
          <button onClick={dismissBreakOver} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Completion toasts (queued, newest last) */}
      {completions.slice(-3).map((c) => (
        <div
          key={c.id}
          className="pointer-events-auto flex w-full items-center gap-3 rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 shadow-2xl"
          style={{ borderLeft: `3px solid ${c.failed ? '#f59e0b' : '#22c55e'}` }}
        >
          <span
            className="grid h-7 w-7 place-items-center rounded-lg"
            style={{
              background: c.failed ? 'rgba(245,158,11,0.15)' : 'rgba(34,197,94,0.15)',
              color: c.failed ? '#f59e0b' : '#22c55e',
            }}
          >
            {c.failed ? <X className="h-4 w-4" /> : <Check className="h-4 w-4" />}
          </span>
          <span className="flex-1 text-xs text-zinc-300">
            {c.failed
              ? t('pomodoro.saveFailed')
              : c.taskId
              ? t('pomodoro.loggedToTask', { minutes: c.minutes, task: c.taskTitle ?? '' })
              : t('pomodoro.loggedNoTask', { minutes: c.minutes })}
          </span>
          {!c.failed && (
            <button
              onClick={() => undoCompletion(c.id)}
              className="whitespace-nowrap rounded-lg border border-blue-500/40 px-3 py-1 text-xs font-bold text-blue-400"
            >
              {t('pomodoro.undo')}
            </button>
          )}
          <button onClick={() => dismissCompletion(c.id)} className="text-zinc-500 hover:text-zinc-300" aria-label={t('pomodoro.dismiss')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      ))}
    </div>
  )
}
