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

/** XP from real actions (Denis 25.09, «Day tasks drive it»). */
export function xpOf(s: Pick<ProfileStats, 'tasksDone' | 'daysFull' | 'reflections'>): number {
  return s.tasksDone * 10 + s.daysFull * 25 + s.reflections * 5
}

const XP_PER_LEVEL = 250

/** Level every 250 XP, from 1; `into` of `need` toward the next. */
export function levelOf(xp: number): { level: number; into: number; need: number } {
  return { level: Math.floor(xp / XP_PER_LEVEL) + 1, into: xp % XP_PER_LEVEL, need: XP_PER_LEVEL }
}

const CHUNK_DAYS = 60
const MAX_CHUNKS = 13 // about two years

/**
 * Every day-tasks day up to today. The API answers at most 62 days per call,
 * and XP must not shrink as old days leave a window, so this reads back chunk
 * by chunk until a chunk lies wholly before the start (or two years).
 */
export async function loadAllDays(today: string, list: (from: string, to: string) => Promise<Day[]>): Promise<Day[]> {
  const out: Day[] = []
  let to = today
  for (let i = 0; i < MAX_CHUNKS; i++) {
    const from = shift(to, -(CHUNK_DAYS - 1))
    const chunk = await list(from, to)
    out.push(...chunk)
    if (chunk.length > 0 && chunk.every((d) => d.before_start)) break
    to = shift(from, -1)
  }
  return out
}
