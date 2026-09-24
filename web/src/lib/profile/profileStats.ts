import type { Day } from '../dayTasks/dayColour'
import { localDayKey } from '../calendar/agenda'

/**
 * The Profile page's numbers, all from real data (Denis 25.09: «real numbers,
 * remove mock data»). Nothing here is invented: every figure is a count over
 * what the API returned.
 */
export interface ProfileStats {
  tasksDone: number
  tasksDoneThisWeek: number
  /** Days with day tasks taken, in the loaded range. */
  daysTaken: number
  /** Taken days where every task was done. */
  daysFull: number
  /** Consecutive taken days ending today (or yesterday while today is not taken yet). */
  takenStreak: number
  reflections: number
}

interface Input {
  today: string
  timeZone: string
  tasks: Array<{ status: string; completedAt?: string }>
  days: Day[]
  reflections: number
}

const DAY_MS = 24 * 60 * 60 * 1000

function shift(day: string, by: number): string {
  return new Date(Date.parse(day + 'T00:00:00Z') + by * DAY_MS).toISOString().slice(0, 10)
}

function mondayOf(day: string): string {
  const d = new Date(day + 'T00:00:00Z')
  return shift(day, -((d.getUTCDay() + 6) % 7))
}

export function profileStats({ today, timeZone, tasks, days, reflections }: Input): ProfileStats {
  const done = tasks.filter((t) => t.status === 'DONE')
  const weekStart = mondayOf(today)
  const tasksDoneThisWeek = done.filter((t) => {
    if (!t.completedAt) return false
    const at = Date.parse(t.completedAt)
    if (Number.isNaN(at)) return false
    const d = localDayKey(new Date(at), timeZone)
    return d >= weekStart && d <= today
  }).length

  const taken = new Map(days.filter((d) => d.confirmed).map((d) => [d.day, d]))
  const daysFull = [...taken.values()].filter((d) => d.target > 0 && d.done >= d.target).length

  let cursor = taken.has(today) ? today : shift(today, -1)
  let takenStreak = 0
  while (taken.has(cursor)) {
    takenStreak++
    cursor = shift(cursor, -1)
  }

  return {
    tasksDone: done.length,
    tasksDoneThisWeek,
    daysTaken: taken.size,
    daysFull,
    takenStreak,
    reflections,
  }
}
