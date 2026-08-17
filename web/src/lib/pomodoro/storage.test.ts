import { describe, it, expect, beforeEach } from 'vitest'
import {
  loadSettings,
  saveSettings,
  loadTimerState,
  saveTimerState,
  clearTimerState,
  computeRemainingMs,
  isStale,
} from './storage'
import { DEFAULT_SETTINGS, type PersistedTimerState } from './types'

const runningState = (endsAt: number): PersistedTimerState => ({
  phase: 'work',
  endsAt,
  isRunning: true,
  remainingWhenPaused: null,
  sessionsCompleted: 1,
  linkedTaskId: 't1',
  linkedTaskTitle: 'Task one',
  blockStartedAt: '2026-06-01T10:00:00.000Z',
})

describe('settings persistence', () => {
  beforeEach(() => localStorage.clear())

  it('returns defaults when nothing stored', () => {
    expect(loadSettings()).toEqual(DEFAULT_SETTINGS)
  })

  it('round-trips and merges unknown-missing keys onto defaults', () => {
    saveSettings({ ...DEFAULT_SETTINGS, workMinutes: 50, widgetStyle: 'pill' })
    const loaded = loadSettings()
    expect(loaded.workMinutes).toBe(50)
    expect(loaded.widgetStyle).toBe('pill')
    expect(loaded.shortBreakMinutes).toBe(DEFAULT_SETTINGS.shortBreakMinutes)
  })
})

describe('timer state persistence', () => {
  beforeEach(() => localStorage.clear())

  it('round-trips state', () => {
    const s = runningState(1000)
    saveTimerState(s)
    expect(loadTimerState()).toEqual(s)
  })

  it('returns null when nothing stored, and after clear', () => {
    expect(loadTimerState()).toBeNull()
    saveTimerState(runningState(1000))
    clearTimerState()
    expect(loadTimerState()).toBeNull()
  })
})

describe('computeRemainingMs', () => {
  it('running: endsAt - now, floored at 0', () => {
    expect(computeRemainingMs(runningState(5000), 2000)).toBe(3000)
    expect(computeRemainingMs(runningState(5000), 9000)).toBe(0)
  })

  it('paused: returns remainingWhenPaused', () => {
    const paused: PersistedTimerState = {
      ...runningState(0),
      isRunning: false,
      endsAt: null,
      remainingWhenPaused: 4200,
    }
    expect(computeRemainingMs(paused, 999999)).toBe(4200)
  })
})

describe('isStale', () => {
  it('true when running and endsAt already passed', () => {
    expect(isStale(runningState(1000), 2000)).toBe(true)
  })
  it('false when running and endsAt still in the future', () => {
    expect(isStale(runningState(5000), 2000)).toBe(false)
  })
  it('false when not running', () => {
    const paused = { ...runningState(1000), isRunning: false, endsAt: null }
    expect(isStale(paused, 999999)).toBe(false)
  })
})
