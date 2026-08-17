import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Timer,
  Play,
  Pause,
  RotateCcw,
  SkipForward,
  Settings as SettingsIcon,
  Volume2,
  VolumeX,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { listTasks, type Task } from '../../api/tasks'
import { usePomodoro } from '../../contexts/PomodoroContext'
import type { TimerMode, WidgetStyle } from '../../lib/pomodoro/types'
import { summarizeToday, loadHistory } from '../../lib/pomodoro/history'
import { formatMs, MODE_HEX } from '../../components/Pomodoro/formatTime'

export default function Pomodoro() {
  const { t } = useTranslation('tools')
  const {
    phase,
    remainingMs,
    isRunning,
    sessionsCompleted,
    linkedTaskId,
    settings,
    start,
    pause,
    reset,
    skip,
    selectPhase,
    setLinkedTask,
    updateSettings,
  } = usePomodoro()

  const [tasks, setTasks] = useState<Task[]>([])
  const [showSettings, setShowSettings] = useState(false)

  useEffect(() => {
    listTasks()
      .then((data) => setTasks(data.filter((t) => t.status !== 'DONE' && t.status !== 'CANCELLED')))
      .catch(console.error)
  }, [])

  const { count: todayCount, minutes: todayMinutes } = summarizeToday(loadHistory(), new Date())
  const filled = sessionsCompleted % settings.sessionsBeforeLong

  const onSelectTask = (id: string) => {
    const task = tasks.find((tk) => tk.id === id) ?? null
    setLinkedTask(id || null, task?.title ?? null)
  }

  return (
    <div className="min-h-screen bg-zinc-950 px-4 py-8 text-zinc-100">
      <div data-hint="tools.pomodoro.timer" className="mx-auto flex w-full max-w-lg flex-col items-center gap-6">
        <div className="flex w-full items-center gap-2">
          <Timer className="h-5 w-5 text-zinc-400" />
          <h1 className="text-xl font-semibold">{t('pomodoro.title')}</h1>
        </div>

        {/* Mode selector */}
        <div className="flex w-full gap-2 rounded-lg bg-zinc-800 p-1">
          {(['work', 'shortBreak', 'longBreak'] as TimerMode[]).map((m) => (
            <button
              key={m}
              onClick={() => selectPhase(m)}
              className={`flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                phase === m ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {t(`pomodoro.${m}`)}
            </button>
          ))}
        </div>

        {/* Cycle pips */}
        <div className="flex items-center gap-2.5 text-xs text-zinc-500">
          <span className="uppercase tracking-wide">{t('pomodoro.cycle')}</span>
          <span className="flex gap-1.5">
            {Array.from({ length: settings.sessionsBeforeLong }).map((_, i) => (
              <span
                key={i}
                className="h-2.5 w-2.5 rounded-full border"
                style={{
                  background: i < filled ? MODE_HEX.work : 'transparent',
                  borderColor: i < filled ? MODE_HEX.work : '#3f3f46',
                  boxShadow: i === filled && phase === 'work' ? `0 0 0 3px rgba(239,68,68,0.18)` : undefined,
                }}
              />
            ))}
          </span>
          <span>{t('pomodoro.cycleProgress', { done: sessionsCompleted, total: settings.sessionsBeforeLong })}</span>
        </div>

        {/* Time */}
        <div className="flex flex-col items-center gap-1">
          <span className="font-mono text-6xl font-bold tabular-nums" style={{ color: MODE_HEX[phase] }}>
            {formatMs(remainingMs)}
          </span>
          <span className="text-sm text-zinc-500">{phase === 'work' ? t('pomodoro.focusBlock', { minutes: settings.workMinutes }) : t(`pomodoro.${phase}`)}</span>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-3">
          <button
            onClick={reset}
            className="rounded-full bg-zinc-800 p-3 text-zinc-400 hover:text-zinc-200"
            aria-label={t('pomodoro.reset')}
          >
            <RotateCcw className="h-5 w-5" />
          </button>
          <button
            onClick={isRunning ? pause : start}
            className="flex items-center gap-2 rounded-full px-8 py-3 font-semibold text-white"
            style={{ background: MODE_HEX[phase] }}
          >
            {isRunning ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
            {isRunning ? t('pomodoro.pause') : t('pomodoro.start')}
          </button>
          <button
            onClick={skip}
            className="rounded-full bg-zinc-800 p-3 text-zinc-400 hover:text-zinc-200"
            aria-label={t('pomodoro.skip')}
          >
            <SkipForward className="h-5 w-5" />
          </button>
        </div>

        {/* Sound toggle */}
        <button
          onClick={() => updateSettings({ soundEnabled: !settings.soundEnabled })}
          className="flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200"
        >
          {settings.soundEnabled ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
          {t('pomodoro.sound')}
        </button>

        {/* Today stats */}
        {todayCount > 0 && (
          <div className="w-full rounded-lg border border-zinc-700 bg-zinc-800/60 px-4 py-2 text-center text-sm text-zinc-400">
            {t('pomodoro.todayBlocks', {
              count: todayCount,
              hours: Math.floor(todayMinutes / 60),
              minutes: todayMinutes % 60,
            })}
          </div>
        )}

        {/* Task linker */}
        <div className="w-full">
          <label className="mb-1 block text-xs text-zinc-500">{t('pomodoro.linkTask')}</label>
          <select
            className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-200"
            value={linkedTaskId ?? ''}
            onChange={(e) => onSelectTask(e.target.value)}
          >
            <option value="">{t('pomodoro.noTask')}</option>
            {tasks.map((task) => (
              <option key={task.id} value={task.id}>
                {task.title}
              </option>
            ))}
          </select>
        </div>

        {/* Settings panel */}
        <div className="w-full overflow-hidden rounded-xl border border-zinc-700">
          <button
            onClick={() => setShowSettings((v) => !v)}
            className="flex w-full items-center justify-between bg-zinc-800 px-4 py-3 text-sm font-medium hover:bg-zinc-700"
          >
            <span className="flex items-center gap-2">
              <SettingsIcon className="h-4 w-4 text-zinc-400" />
              {t('pomodoro.settings')}
            </span>
            {showSettings ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
          {showSettings && (
            <div className="flex flex-col gap-4 bg-zinc-900 px-4 py-4">
              {(
                [
                  ['workMinutes', 'pomodoro.workDuration'],
                  ['shortBreakMinutes', 'pomodoro.shortBreakDuration'],
                  ['longBreakMinutes', 'pomodoro.longBreakDuration'],
                  ['sessionsBeforeLong', 'pomodoro.sessionsBeforeLong'],
                ] as [keyof typeof settings, string][]
              ).map(([key, labelKey]) => (
                <div key={key} className="flex items-center justify-between gap-4">
                  <label className="text-sm text-zinc-300">{t(labelKey)}</label>
                  <input
                    type="number"
                    min={1}
                    max={key === 'sessionsBeforeLong' ? 10 : 120}
                    value={settings[key] as number}
                    onChange={(e) => {
                      const v = parseInt(e.target.value, 10)
                      if (!isNaN(v) && v > 0) updateSettings({ [key]: v })
                    }}
                    className="w-20 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-center text-sm text-zinc-200"
                  />
                </div>
              ))}

              {/* Widget style */}
              <div className="flex items-center justify-between gap-4">
                <label className="text-sm text-zinc-300">{t('pomodoro.widgetStyle')}</label>
                <div className="flex gap-1 rounded-lg bg-zinc-800 p-1">
                  {(['pill', 'card', 'bar'] as WidgetStyle[]).map((style) => (
                    <button
                      key={style}
                      onClick={() => updateSettings({ widgetStyle: style })}
                      className={`rounded-md px-2.5 py-1 text-xs ${
                        settings.widgetStyle === style ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400'
                      }`}
                    >
                      {t(`pomodoro.widget.${style}`)}
                    </button>
                  ))}
                </div>
              </div>

              {/* Auto-start toggle */}
              <div className="flex items-center justify-between">
                <label className="text-sm text-zinc-300">{t('pomodoro.autoStartBreaks')}</label>
                <button
                  role="switch"
                  aria-checked={settings.autoStartBreaks}
                  onClick={() => updateSettings({ autoStartBreaks: !settings.autoStartBreaks })}
                  className={`relative h-6 w-10 rounded-full transition-colors ${
                    settings.autoStartBreaks ? 'bg-blue-600' : 'bg-zinc-700'
                  }`}
                >
                  <span
                    className={`absolute top-1 h-4 w-4 rounded-full bg-white transition-transform ${
                      settings.autoStartBreaks ? 'left-5' : 'left-1'
                    }`}
                  />
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
