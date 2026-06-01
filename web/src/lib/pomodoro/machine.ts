import type { TimerMode, PomodoroSettings } from './types'

/**
 * Given the phase that just finished, returns the next phase.
 * For work, pass the sessionsCompleted count AFTER incrementing it.
 */
export function nextPhase(
  finishedPhase: TimerMode,
  sessionsCompletedAfter: number,
  sessionsBeforeLong: number
): TimerMode {
  if (finishedPhase === 'work') {
    return sessionsCompletedAfter % sessionsBeforeLong === 0 ? 'longBreak' : 'shortBreak'
  }
  return 'work'
}

export function minutesForMode(mode: TimerMode, settings: PomodoroSettings): number {
  switch (mode) {
    case 'work':
      return settings.workMinutes
    case 'shortBreak':
      return settings.shortBreakMinutes
    case 'longBreak':
      return settings.longBreakMinutes
  }
}

export function durationMsForMode(mode: TimerMode, settings: PomodoroSettings): number {
  return minutesForMode(mode, settings) * 60 * 1000
}
