import { describe, it, expect } from 'vitest'
import { nextPhase, minutesForMode, durationMsForMode } from './machine'
import { DEFAULT_SETTINGS } from './types'

describe('nextPhase', () => {
  it('work → shortBreak when the completed count is not a multiple of sessionsBeforeLong', () => {
    expect(nextPhase('work', 1, 4)).toBe('shortBreak')
    expect(nextPhase('work', 2, 4)).toBe('shortBreak')
    expect(nextPhase('work', 3, 4)).toBe('shortBreak')
  })

  it('work → longBreak when the completed count hits a multiple of sessionsBeforeLong', () => {
    expect(nextPhase('work', 4, 4)).toBe('longBreak')
    expect(nextPhase('work', 8, 4)).toBe('longBreak')
  })

  it('any break → work', () => {
    expect(nextPhase('shortBreak', 2, 4)).toBe('work')
    expect(nextPhase('longBreak', 4, 4)).toBe('work')
  })

  it('walks a full set: 4th work block tips into the long break', () => {
    const before = 4
    const seq: string[] = []
    let completed = 0
    for (let i = 0; i < 4; i++) {
      completed += 1
      seq.push(nextPhase('work', completed, before))
    }
    expect(seq).toEqual(['shortBreak', 'shortBreak', 'shortBreak', 'longBreak'])
  })
})

describe('durations', () => {
  it('minutesForMode reads the matching setting', () => {
    expect(minutesForMode('work', DEFAULT_SETTINGS)).toBe(25)
    expect(minutesForMode('shortBreak', DEFAULT_SETTINGS)).toBe(5)
    expect(minutesForMode('longBreak', DEFAULT_SETTINGS)).toBe(15)
  })

  it('durationMsForMode converts minutes to ms', () => {
    expect(durationMsForMode('work', DEFAULT_SETTINGS)).toBe(25 * 60 * 1000)
  })
})
