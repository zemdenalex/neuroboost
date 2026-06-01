import { DEFAULT_SETTINGS, type PomodoroSettings, type PersistedTimerState } from './types'

const SETTINGS_KEY = 'nb-pomodoro-settings'
const STATE_KEY = 'nb-pomodoro-state'

export function loadSettings(): PomodoroSettings {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY)
    if (!raw) return DEFAULT_SETTINGS
    return { ...DEFAULT_SETTINGS, ...(JSON.parse(raw) as Partial<PomodoroSettings>) }
  } catch {
    return DEFAULT_SETTINGS
  }
}

export function saveSettings(s: PomodoroSettings): void {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(s))
}

export function loadTimerState(): PersistedTimerState | null {
  try {
    const raw = localStorage.getItem(STATE_KEY)
    if (!raw) return null
    return JSON.parse(raw) as PersistedTimerState
  } catch {
    return null
  }
}

export function saveTimerState(s: PersistedTimerState): void {
  localStorage.setItem(STATE_KEY, JSON.stringify(s))
}

export function clearTimerState(): void {
  localStorage.removeItem(STATE_KEY)
}

/** Remaining ms: derived from the wall clock when running, else the paused value. */
export function computeRemainingMs(state: PersistedTimerState, now: number): number {
  if (!state.isRunning) return state.remainingWhenPaused ?? 0
  if (state.endsAt == null) return 0
  return Math.max(0, state.endsAt - now)
}

/** A running block whose endsAt is already in the past — elapsed while the app was closed. */
export function isStale(state: PersistedTimerState, now: number): boolean {
  return state.isRunning && state.endsAt != null && state.endsAt <= now
}
