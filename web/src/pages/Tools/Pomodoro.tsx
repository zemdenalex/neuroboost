import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Timer,
  Play,
  Pause,
  RotateCcw,
  SkipForward,
  Settings,
  Volume2,
  VolumeX,
  ChevronDown,
  ChevronUp,
  LinkIcon,
} from 'lucide-react'
import { listTasks } from '../../api/tasks'
import type { Task } from '../../api/tasks'

// ── Types ────────────────────────────────────────────────────────────────────

export interface PomodoroSettings {
  workMinutes: number
  shortBreakMinutes: number
  longBreakMinutes: number
  autoStartBreaks: boolean
  soundEnabled: boolean
  sessionsBeforeLong: number
}

export type TimerMode = 'work' | 'shortBreak' | 'longBreak'

interface SessionRecord {
  date: string // ISO date YYYY-MM-DD
  mode: TimerMode
  durationSeconds: number
  taskId?: string
  completedAt: string // ISO timestamp
}

// ── Constants ────────────────────────────────────────────────────────────────

const STORAGE_KEY_SETTINGS = 'nb-pomodoro-settings'
const STORAGE_KEY_HISTORY = 'nb-pomodoro-history'

const DEFAULT_SETTINGS: PomodoroSettings = {
  workMinutes: 25,
  shortBreakMinutes: 5,
  longBreakMinutes: 15,
  autoStartBreaks: true,
  soundEnabled: true,
  sessionsBeforeLong: 4,
}

const MODE_COLORS: Record<TimerMode, string> = {
  work: 'text-red-400 border-red-500',
  shortBreak: 'text-green-400 border-green-500',
  longBreak: 'text-blue-400 border-blue-500',
}

const MODE_RING_COLORS: Record<TimerMode, string> = {
  work: '#ef4444',
  shortBreak: '#22c55e',
  longBreak: '#3b82f6',
}

const MODE_BADGE_COLORS: Record<TimerMode, string> = {
  work: 'bg-red-500/20 text-red-300 border border-red-500/30',
  shortBreak: 'bg-green-500/20 text-green-300 border border-green-500/30',
  longBreak: 'bg-blue-500/20 text-blue-300 border border-blue-500/30',
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function loadSettings(): PomodoroSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_SETTINGS)
    if (!raw) return DEFAULT_SETTINGS
    return { ...DEFAULT_SETTINGS, ...JSON.parse(raw) } as PomodoroSettings
  } catch {
    return DEFAULT_SETTINGS
  }
}

function saveSettings(s: PomodoroSettings): void {
  localStorage.setItem(STORAGE_KEY_SETTINGS, JSON.stringify(s))
}

function loadHistory(): SessionRecord[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_HISTORY)
    if (!raw) return []
    return JSON.parse(raw) as SessionRecord[]
  } catch {
    return []
  }
}

function saveHistory(h: SessionRecord[]): void {
  localStorage.setItem(STORAGE_KEY_HISTORY, JSON.stringify(h))
}

function todayString(): string {
  return new Date().toISOString().slice(0, 10)
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

function formatDuration(totalMinutes: number): { hours: number; minutes: number } {
  return { hours: Math.floor(totalMinutes / 60), minutes: totalMinutes % 60 }
}

function playBeep(enabled: boolean): void {
  if (!enabled) return
  try {
    const ctx = new AudioContext()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'sine'
    osc.frequency.setValueAtTime(800, ctx.currentTime)
    gain.gain.setValueAtTime(0.4, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.4)
    osc.onended = () => ctx.close()
  } catch {
    // AudioContext not available (e.g. test env) — silently ignore
  }
}

function requestNotificationPermission(): void {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
}

function showNotification(title: string, body: string): void {
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification(title, { body, icon: '/favicon.ico' })
  }
}

// ── Progress ring ─────────────────────────────────────────────────────────────

interface ProgressRingProps {
  progress: number // 0–1
  mode: TimerMode
  size?: number
  strokeWidth?: number
}

function ProgressRing({ progress, mode, size = 220, strokeWidth = 6 }: ProgressRingProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const dash = circumference * (1 - progress)
  const color = MODE_RING_COLORS[mode]

  return (
    <svg
      width={size}
      height={size}
      className="absolute top-0 left-0"
      style={{ transform: 'rotate(-90deg)' }}
      aria-hidden="true"
    >
      {/* Track */}
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        className="text-zinc-700"
      />
      {/* Progress */}
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={color}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
        strokeDasharray={circumference}
        strokeDashoffset={dash}
        style={{ transition: 'stroke-dashoffset 0.5s ease' }}
      />
    </svg>
  )
}

// ── Main component ───────────────────────────────────────────────────────────

export default function Pomodoro() {
  const { t } = useTranslation('tools')

  // Settings
  const [settings, setSettings] = useState<PomodoroSettings>(loadSettings)
  const [showSettings, setShowSettings] = useState(false)

  // Timer state
  const [mode, setMode] = useState<TimerMode>('work')
  const [timeLeft, setTimeLeft] = useState(settings.workMinutes * 60)
  const [isRunning, setIsRunning] = useState(false)
  const [sessionsCompleted, setSessionsCompleted] = useState(0)

  // Task linking
  const [tasks, setTasks] = useState<Task[]>([])
  const [linkedTaskId, setLinkedTaskId] = useState<string | null>(null)
  const [tasksLoading, setTasksLoading] = useState(false)

  // History
  const [history, setHistory] = useState<SessionRecord[]>(loadHistory)

  // Track the total duration of current session for history recording
  const totalSecondsRef = useRef<number>(settings.workMinutes * 60)

  // ── Derived ────────────────────────────────────────────────────────────────

  const totalForMode = useCallback(
    (m: TimerMode): number => {
      if (m === 'work') return settings.workMinutes * 60
      if (m === 'shortBreak') return settings.shortBreakMinutes * 60
      return settings.longBreakMinutes * 60
    },
    [settings]
  )

  const progress = timeLeft / totalSecondsRef.current

  const todaySessions = history.filter(
    (r) => r.date === todayString() && r.mode === 'work'
  )
  const todayCount = todaySessions.length
  const todayMinutes = todaySessions.reduce((acc, r) => acc + Math.floor(r.durationSeconds / 60), 0)
  const { hours: todayHours, minutes: todayMins } = formatDuration(todayMinutes)

  const linkedTask = tasks.find((t) => t.id === linkedTaskId) ?? null

  const currentSession = (sessionsCompleted % settings.sessionsBeforeLong) + 1

  // ── Fetch tasks ───────────────────────────────────────────────────────────

  useEffect(() => {
    setTasksLoading(true)
    listTasks()
      .then((data) => setTasks(data.filter((t) => t.status !== 'DONE' && t.status !== 'CANCELLED')))
      .catch(console.error)
      .finally(() => setTasksLoading(false))
  }, [])

  // ── Timer logic ───────────────────────────────────────────────────────────

  const handleTimerEnd = useCallback(() => {
    setIsRunning(false)
    playBeep(settings.soundEnabled)

    const record: SessionRecord = {
      date: todayString(),
      mode,
      durationSeconds: totalSecondsRef.current,
      taskId: linkedTaskId ?? undefined,
      completedAt: new Date().toISOString(),
    }

    const updated = [...loadHistory(), record]
    saveHistory(updated)
    setHistory(updated)

    if (mode === 'work') {
      const newCompleted = sessionsCompleted + 1
      setSessionsCompleted(newCompleted)
      const isLong = newCompleted % settings.sessionsBeforeLong === 0
      const nextMode: TimerMode = isLong ? 'longBreak' : 'shortBreak'

      showNotification(
        t('pomodoro.workComplete'),
        isLong ? t('pomodoro.timeForBreak') : t('pomodoro.timeForBreak')
      )

      if (settings.autoStartBreaks) {
        const nextTotal = isLong
          ? settings.longBreakMinutes * 60
          : settings.shortBreakMinutes * 60
        setMode(nextMode)
        totalSecondsRef.current = nextTotal
        setTimeLeft(nextTotal)
        setIsRunning(true)
      } else {
        const nextTotal = isLong
          ? settings.longBreakMinutes * 60
          : settings.shortBreakMinutes * 60
        setMode(nextMode)
        totalSecondsRef.current = nextTotal
        setTimeLeft(nextTotal)
      }
    } else {
      showNotification(t('pomodoro.breakComplete'), t('pomodoro.timeToWork'))
      const nextTotal = settings.workMinutes * 60
      setMode('work')
      totalSecondsRef.current = nextTotal
      setTimeLeft(nextTotal)
    }
  }, [settings, mode, sessionsCompleted, linkedTaskId, t])

  useEffect(() => {
    if (!isRunning) return
    const id = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          handleTimerEnd()
          return 0
        }
        return prev - 1
      })
    }, 1000)
    return () => clearInterval(id)
  }, [isRunning, handleTimerEnd])

  // ── Handlers ──────────────────────────────────────────────────────────────

  const handleStartPause = () => {
    if (!isRunning) requestNotificationPermission()
    setIsRunning((v) => !v)
  }

  const handleReset = () => {
    setIsRunning(false)
    const total = totalForMode(mode)
    totalSecondsRef.current = total
    setTimeLeft(total)
  }

  const handleSkip = () => {
    setIsRunning(false)
    if (mode === 'work') {
      const newCompleted = sessionsCompleted + 1
      setSessionsCompleted(newCompleted)
      const isLong = newCompleted % settings.sessionsBeforeLong === 0
      const nextMode: TimerMode = isLong ? 'longBreak' : 'shortBreak'
      const nextTotal = isLong
        ? settings.longBreakMinutes * 60
        : settings.shortBreakMinutes * 60
      setMode(nextMode)
      totalSecondsRef.current = nextTotal
      setTimeLeft(nextTotal)
    } else {
      const nextTotal = settings.workMinutes * 60
      setMode('work')
      totalSecondsRef.current = nextTotal
      setTimeLeft(nextTotal)
    }
  }

  const handleModeSelect = (m: TimerMode) => {
    setIsRunning(false)
    const total = totalForMode(m)
    setMode(m)
    totalSecondsRef.current = total
    setTimeLeft(total)
  }

  const handleSettingChange = <K extends keyof PomodoroSettings>(
    key: K,
    value: PomodoroSettings[K]
  ) => {
    setSettings((prev) => {
      const next = { ...prev, [key]: value }
      saveSettings(next)
      // If the active mode's duration changed, update the timer (only when not running)
      if (!isRunning) {
        const modeKey: Record<TimerMode, keyof PomodoroSettings> = {
          work: 'workMinutes',
          shortBreak: 'shortBreakMinutes',
          longBreak: 'longBreakMinutes',
        }
        if (key === modeKey[mode]) {
          const newTotal = (value as number) * 60
          totalSecondsRef.current = newTotal
          setTimeLeft(newTotal)
        }
      }
      return next
    })
  }

  // ── Render ────────────────────────────────────────────────────────────────

  const ringSize = 220

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex flex-col items-center px-4 py-8">
      {/* Page title */}
      <div className="flex items-center gap-2 mb-8 self-start w-full max-w-lg">
        <Timer className="w-5 h-5 text-zinc-400" />
        <h1 className="text-xl font-semibold">{t('pomodoro.title')}</h1>
      </div>

      <div className="w-full max-w-lg flex flex-col items-center gap-6">
        {/* Mode selector */}
        <div className="flex gap-2 bg-zinc-800 p-1 rounded-lg w-full">
          {(['work', 'shortBreak', 'longBreak'] as TimerMode[]).map((m) => (
            <button
              key={m}
              onClick={() => handleModeSelect(m)}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-colors ${
                mode === m
                  ? 'bg-zinc-700 text-zinc-100'
                  : 'text-zinc-400 hover:text-zinc-200'
              }`}
            >
              {t(`pomodoro.${m}`)}
            </button>
          ))}
        </div>

        {/* Mode badge */}
        <div className={`px-3 py-1 rounded-full text-xs font-medium ${MODE_BADGE_COLORS[mode]}`}>
          {t(`pomodoro.${mode}`)}
        </div>

        {/* Linked task */}
        {linkedTask && (
          <div className="flex items-center gap-2 bg-zinc-800/60 border border-zinc-700 rounded-lg px-4 py-2 text-sm w-full">
            <LinkIcon className="w-4 h-4 text-zinc-400 shrink-0" />
            <span className="text-zinc-300 truncate">{linkedTask.title}</span>
          </div>
        )}

        {/* Timer ring + display */}
        <div
          className="relative flex items-center justify-center"
          style={{ width: ringSize, height: ringSize }}
        >
          <ProgressRing progress={progress} mode={mode} size={ringSize} strokeWidth={7} />
          <div className="flex flex-col items-center gap-1 z-10">
            <span
              className={`font-mono text-6xl font-bold tabular-nums ${MODE_COLORS[mode].split(' ')[0]}`}
            >
              {formatTime(timeLeft)}
            </span>
            <span className="text-zinc-500 text-sm">
              {t('pomodoro.session', {
                current: currentSession,
                total: settings.sessionsBeforeLong,
              })}
            </span>
          </div>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-3">
          <button
            onClick={handleReset}
            className="p-3 rounded-full bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 transition-colors"
            title={t('pomodoro.reset')}
            aria-label={t('pomodoro.reset')}
          >
            <RotateCcw className="w-5 h-5" />
          </button>

          <button
            onClick={handleStartPause}
            className={`px-8 py-3 rounded-full font-semibold text-base transition-colors flex items-center gap-2 ${
              mode === 'work'
                ? 'bg-red-600 hover:bg-red-500 text-white'
                : mode === 'shortBreak'
                ? 'bg-green-600 hover:bg-green-500 text-white'
                : 'bg-blue-600 hover:bg-blue-500 text-white'
            }`}
          >
            {isRunning ? (
              <>
                <Pause className="w-5 h-5" />
                {t('pomodoro.pause')}
              </>
            ) : (
              <>
                <Play className="w-5 h-5" />
                {t('pomodoro.start')}
              </>
            )}
          </button>

          <button
            onClick={handleSkip}
            className="p-3 rounded-full bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 transition-colors"
            title={t('pomodoro.skip')}
            aria-label={t('pomodoro.skip')}
          >
            <SkipForward className="w-5 h-5" />
          </button>
        </div>

        {/* Sound toggle */}
        <button
          onClick={() => handleSettingChange('soundEnabled', !settings.soundEnabled)}
          className="flex items-center gap-2 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
        >
          {settings.soundEnabled ? (
            <Volume2 className="w-4 h-4" />
          ) : (
            <VolumeX className="w-4 h-4" />
          )}
          {t('pomodoro.sound')}
        </button>

        {/* Today stats */}
        {todayCount > 0 && (
          <div className="text-sm text-zinc-400 bg-zinc-800/60 border border-zinc-700 rounded-lg px-4 py-2 w-full text-center">
            {todayHours > 0
              ? t('pomodoro.todayStats', {
                  count: todayCount,
                  hours: todayHours,
                  minutes: todayMins,
                })
              : t('pomodoro.todayStatsShort', { count: todayCount })}
          </div>
        )}

        {/* Task linker */}
        <div className="w-full">
          <label className="block text-xs text-zinc-500 mb-1">{t('pomodoro.linkTask')}</label>
          <select
            className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-200 focus:outline-none focus:ring-2 focus:ring-zinc-600"
            value={linkedTaskId ?? ''}
            onChange={(e) => setLinkedTaskId(e.target.value || null)}
            disabled={tasksLoading}
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
        <div className="w-full border border-zinc-700 rounded-xl overflow-hidden">
          <button
            onClick={() => setShowSettings((v) => !v)}
            className="w-full flex items-center justify-between px-4 py-3 bg-zinc-800 hover:bg-zinc-700 transition-colors text-sm font-medium"
          >
            <span className="flex items-center gap-2">
              <Settings className="w-4 h-4 text-zinc-400" />
              {t('pomodoro.settings')}
            </span>
            {showSettings ? (
              <ChevronUp className="w-4 h-4 text-zinc-400" />
            ) : (
              <ChevronDown className="w-4 h-4 text-zinc-400" />
            )}
          </button>

          {showSettings && (
            <div className="bg-zinc-900 px-4 py-4 flex flex-col gap-4">
              {/* Duration inputs */}
              {(
                [
                  ['workMinutes', 'pomodoro.workDuration'],
                  ['shortBreakMinutes', 'pomodoro.shortBreakDuration'],
                  ['longBreakMinutes', 'pomodoro.longBreakDuration'],
                  ['sessionsBeforeLong', 'pomodoro.sessionsBeforeLong'],
                ] as [keyof PomodoroSettings, string][]
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
                      if (!isNaN(v) && v > 0) {
                        handleSettingChange(key, v as PomodoroSettings[typeof key])
                      }
                    }}
                    className="w-20 bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-center text-zinc-200 focus:outline-none focus:ring-2 focus:ring-zinc-600"
                  />
                </div>
              ))}

              {/* Toggles */}
              <div className="flex items-center justify-between">
                <label className="text-sm text-zinc-300">{t('pomodoro.autoStartBreaks')}</label>
                <button
                  role="switch"
                  aria-checked={settings.autoStartBreaks}
                  onClick={() =>
                    handleSettingChange('autoStartBreaks', !settings.autoStartBreaks)
                  }
                  className={`relative w-10 h-6 rounded-full transition-colors ${
                    settings.autoStartBreaks ? 'bg-blue-600' : 'bg-zinc-700'
                  }`}
                >
                  <span
                    className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${
                      settings.autoStartBreaks ? 'left-5' : 'left-1'
                    }`}
                  />
                </button>
              </div>

              <div className="flex items-center justify-between">
                <label className="text-sm text-zinc-300">{t('pomodoro.sound')}</label>
                <button
                  role="switch"
                  aria-checked={settings.soundEnabled}
                  onClick={() => handleSettingChange('soundEnabled', !settings.soundEnabled)}
                  className={`relative w-10 h-6 rounded-full transition-colors ${
                    settings.soundEnabled ? 'bg-blue-600' : 'bg-zinc-700'
                  }`}
                >
                  <span
                    className={`absolute top-1 w-4 h-4 rounded-full bg-white transition-transform ${
                      settings.soundEnabled ? 'left-5' : 'left-1'
                    }`}
                  />
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Session history — today */}
        {history.filter((r) => r.date === todayString()).length > 0 && (
          <div className="w-full">
            <h2 className="text-xs text-zinc-500 uppercase tracking-wider mb-2">
              {t('pomodoro.history')}
            </h2>
            <ul className="space-y-1">
              {history
                .filter((r) => r.date === todayString())
                .slice()
                .reverse()
                .slice(0, 8)
                .map((record, idx) => {
                  const linkedName = record.taskId
                    ? tasks.find((tk) => tk.id === record.taskId)?.title
                    : undefined
                  return (
                    <li
                      key={idx}
                      className="flex items-center justify-between bg-zinc-800/40 border border-zinc-700/50 rounded-lg px-3 py-2 text-xs"
                    >
                      <span
                        className={`font-medium ${
                          record.mode === 'work'
                            ? 'text-red-400'
                            : record.mode === 'shortBreak'
                            ? 'text-green-400'
                            : 'text-blue-400'
                        }`}
                      >
                        {t(`pomodoro.${record.mode}`)}
                        {linkedName && (
                          <span className="text-zinc-500 font-normal ml-1">— {linkedName}</span>
                        )}
                      </span>
                      <span className="text-zinc-500">
                        {new Date(record.completedAt).toLocaleTimeString([], {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </li>
                  )
                })}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
