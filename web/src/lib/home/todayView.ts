import { answeredToday, type Task } from '../../types'
import { dayLevelSquare, type Day } from '../dayTasks/dayColour'

/**
 * Home as the bot's «📅 Сегодня» (gap list row 8, Denis 26.09; bot today.go):
 * open tasks, most urgent first, five and «и ещё N»; the day-tasks line.
 *
 * Two differences from the bot, both on purpose: priority 0 (Buffer) sorts
 * LAST — it is the lowest level (gotcha 4), and the bot's plain ascending sort
 * put it first — and a repeating task already answered today is left out, as
 * the web's task list does.
 */
const rank = (p: number) => (p === 0 ? 99 : p)

export function homeTasks(tasks: Task[], limit = 5): { shown: Task[]; more: number } {
  const open = tasks
    .filter((t) => (t.status === 'TODO' || t.status === 'SCHEDULED') && !answeredToday(t))
    .sort((a, b) => rank(a.priority) - rank(b.priority))
  return { shown: open.slice(0, limit), more: Math.max(0, open.length - limit) }
}

export type HomeDayLine = { kind: 'notTaken' } | { kind: 'taken'; square: string; done: number; target: number }

/** The «📌 🟧 3 из 5» line; null when there is no day to show (off, or not loaded). */
export function homeDayLine(day: Day | undefined): HomeDayLine | null {
  if (!day) return null
  if (!day.confirmed) return { kind: 'notTaken' }
  return { kind: 'taken', square: dayLevelSquare(day.level), done: day.done, target: day.target }
}
