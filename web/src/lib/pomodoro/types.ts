export type TimerMode = 'work' | 'shortBreak' | 'longBreak'

export type WidgetStyle = 'pill' | 'card' | 'bar'

export interface PomodoroSettings {
  workMinutes: number
  shortBreakMinutes: number
  longBreakMinutes: number
  autoStartBreaks: boolean
  soundEnabled: boolean
  sessionsBeforeLong: number
  widgetStyle: WidgetStyle
}

export const DEFAULT_SETTINGS: PomodoroSettings = {
  workMinutes: 25,
  shortBreakMinutes: 5,
  longBreakMinutes: 15,
  autoStartBreaks: true,
  soundEnabled: true,
  sessionsBeforeLong: 4,
  widgetStyle: 'card',
}

export interface PersistedTimerState {
  phase: TimerMode
  endsAt: number | null // epoch ms; null when idle/paused
  isRunning: boolean
  remainingWhenPaused: number | null // ms
  sessionsCompleted: number
  linkedTaskId: string | null
  linkedTaskTitle: string | null
  blockStartedAt: string | null // ISO; start of the current work block
}

export interface SessionRecord {
  date: string // YYYY-MM-DD
  mode: TimerMode
  durationSeconds: number
  taskId?: string
  completedAt: string // ISO
}

export interface Completion {
  eventId: string | null
  taskId: string | null
  minutes: number
  failed: boolean
}
