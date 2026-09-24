/**
 * Day tasks: the day's colour (spec 2026-09-22 §3, §11; web port 2026-09-24).
 *
 * A port of bot/internal/handlers/daycolour.go with the same test table.
 * The server computes `level` and `before_start`; this only decides which
 * days get a square, so the bot and the web cannot paint one day twice.
 */

export interface DayItem {
  task_id: string
  title: string
  done: boolean
}

/** One day of GET /api/day-tasks, as the server sends it. */
export interface Day {
  day: string
  target: number
  confirmed: boolean
  items: DayItem[]
  done: number
  level: number
  before_start: boolean
}

// Denis's own table: 0 nothing or not taken · 5 all done.
const SQUARES = ['⬛', '🟫', '🟥', '🟧', '🟨', '🟩'] as const

/** The square for a server level; anything out of range is ⬛, as in the bot. */
export function dayLevelSquare(level: number): string {
  return SQUARES[level] ?? SQUARES[0]
}

/**
 * The square each day gets, keyed YYYY-MM-DD; a day with none is absent.
 * The future never, today only once taken, a day before the start only if
 * the person chose so. After the start an untaken day is ⬛.
 */
export function dayColours(days: Day[], today: string, paintBefore: boolean): Record<string, string> {
  const out: Record<string, string> = {}
  for (const d of days) {
    if (d.day > today) continue
    if (d.day === today && !d.confirmed) continue
    if (d.before_start && !paintBefore) continue
    out[d.day] = dayLevelSquare(d.level)
  }
  return out
}

/**
 * Today as YYYY-MM-DD in the person's zone, not the browser's
 * (learning-the-right-time-in-the-wrong-zone). An unknown zone falls back to
 * Moscow, the server's default.
 */
export function todayInZone(now: Date, timeZone: string): string {
  const format = (tz: string) =>
    new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(now)
  try {
    return format(timeZone)
  } catch {
    return format('Europe/Moscow')
  }
}
