import { describe, it, expect } from 'vitest'
import { profileStats } from './profileStats'
import type { Day } from '../dayTasks/dayColour'

const day = (d: string, confirmed: boolean, done = 0, target = 5): Day => ({
  day: d, target, confirmed, items: [], done, level: 0, before_start: false,
})

describe('profileStats', () => {
  const today = '2026-09-24' // a Thursday; its week starts Mon 21 Sep

  it('counts done tasks, all time and this week, by when they were done', () => {
    const s = profileStats({
      today,
      timeZone: 'Europe/Moscow',
      tasks: [
        { status: 'DONE', completedAt: '2026-09-22T10:00:00Z' },
        { status: 'DONE', completedAt: '2026-09-10T10:00:00Z' },
        // 20 Sep 22:30Z is Monday 21 Sep 01:30 in Moscow: this week.
        { status: 'DONE', completedAt: '2026-09-20T22:30:00Z' },
        { status: 'TODO' },
      ],
      days: [],
      reflections: 0,
    })
    expect(s.tasksDone).toBe(3)
    expect(s.tasksDoneThisWeek).toBe(2)
  })

  it('counts taken and full day-task days, and the streak of taken days up to today', () => {
    const s = profileStats({
      today,
      timeZone: 'Europe/Moscow',
      tasks: [],
      days: [
        day('2026-09-20', true, 5),
        day('2026-09-21', false),
        day('2026-09-22', true, 3),
        day('2026-09-23', true, 5),
        day('2026-09-24', true, 1),
      ],
      reflections: 4,
    })
    expect(s.daysTaken).toBe(4)
    expect(s.daysFull).toBe(2)
    expect(s.takenStreak).toBe(3)
    expect(s.reflections).toBe(4)
  })

  it('keeps the streak alive through today while today is not taken yet', () => {
    const s = profileStats({
      today,
      timeZone: 'Europe/Moscow',
      tasks: [],
      days: [day('2026-09-22', true, 2), day('2026-09-23', true, 2), day('2026-09-24', false)],
      reflections: 0,
    })
    expect(s.takenStreak).toBe(2)
  })

  it('is all zeros for a new account', () => {
    expect(profileStats({ today, timeZone: 'UTC', tasks: [], days: [], reflections: 0 })).toEqual({
      tasksDone: 0, tasksDoneThisWeek: 0, daysTaken: 0, daysFull: 0, takenStreak: 0, reflections: 0,
    })
  })
})
