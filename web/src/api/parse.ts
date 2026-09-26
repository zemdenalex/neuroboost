import { api } from './client'

/**
 * POST /api/parse: one typed line, read exactly as the bot reads it (gap list
 * row 1, Denis 26.09: «one parser, API endpoint»). The server owns the
 * vocabulary: the user's zone, calendars, own words and reminder presets.
 *
 * `kind` says what to do: `task` is saved at once (the bot's quick save),
 * `event` is confirmed first, `ask` is a line the bot would ask about
 * (`missing` names what); the web keeps its old behaviour for those.
 */
export interface ParsedLine {
  kind: 'task' | 'event' | 'ask'
  title: string
  /** The zone the line was read in; wall times are printed in it. */
  timezone: string
  missing?: string
  tags: string[]
  rrule: string | null
  // Task fields. null = not stated; priority 0 is Buffer, a real answer.
  priority: number | null
  due_date: string | null
  estimated_minutes: number | null
  // Event fields, in the shape POST /api/events takes.
  starts_at: string | null
  ends_at: string | null
  all_day: boolean
  color: string | null
  calendar_id: string | null
  calendar_name: string | null
  /** null = the user's preset; [] = stated as none (silent). */
  reminder_offsets: number[] | null
  /** The word «задача» was used: a task plus an event bound to it. */
  is_task: boolean
  uncertain: string[]
}

/** How long the row waits for the parser before it saves the line as typed. */
export const PARSE_TIMEOUT_MS = 3000

/**
 * The parsed line, or null when the parser cannot be reached in time: a 404
 * from an API without the route, a 5xx, a network error, a timeout. The
 * caller then does exactly what it did before the parser existed.
 */
export async function parseLine(text: string, timeoutMs = PARSE_TIMEOUT_MS): Promise<ParsedLine | null> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const timeout = new Promise<null>(resolve => {
    timer = setTimeout(() => resolve(null), timeoutMs)
  })
  try {
    return await Promise.race([
      api.post<ParsedLine>('/parse', { text }).then(p => (p && typeof p.kind === 'string' ? p : null)),
      timeout,
    ])
  } catch {
    return null
  } finally {
    clearTimeout(timer)
  }
}
