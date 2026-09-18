import type { NbEvent } from '../../types'
import type { Task } from '../../types'

/**
 * Grouping for the agenda: "what is coming", not "what does a day look like".
 *
 * 🔴 This is the one view the product has nowhere — not in the web, not in the
 * bot (docs/razbor-mobilnyy-kalendar-2026-08-19.md §3, option B). The month and
 * the day both answer "how is this period shaped"; neither answers "what is
 * next", which is the question a phone is usually taken out to ask.
 *
 * Pure functions on plain data, tested without a browser: the grouping is where
 * the bugs live (timezones, all-day boundaries, tasks with no time), and none of
 * them need React to reproduce.
 */

export interface AgendaItem {
  id: string
  kind: 'event' | 'task'
  title: string
  /** Start instant for an event; the due instant for a task. */
  at: Date
  allDay: boolean
  /** End instant, events only. */
  until?: Date
  colour?: string
  done?: boolean
}

export interface AgendaDay {
  /** Local midnight of the day, as a Date. */
  day: Date
  /** ISO yyyy-mm-dd in the viewer's zone — a stable React key. */
  key: string
  items: AgendaItem[]
}

/** localDayKey is the calendar day an instant falls on, in a named zone. */
export function localDayKey(at: Date, timezone: string): string {
  // 🔴 Formatted in the target zone rather than sliced off an ISO string.
  // toISOString() is UTC, so anything after 21:00 Moscow would be filed under
  // tomorrow — the exact mistake made reading a due_date on 17.09.
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(at)
  return parts
}

/**
 * Builds the agenda: every event and every dated task from `from` onward,
 * grouped by local day, each day's items in time order.
 *
 * Undated tasks are left out on purpose. The agenda answers "what is next", and
 * something with no date has no place in that order; it lives in the task list,
 * which is the screen for exactly that.
 */
export function buildAgenda(
  events: NbEvent[],
  tasks: Task[],
  timezone: string,
  from: Date,
  days: number,
): AgendaDay[] {
  const items: AgendaItem[] = []

  for (const e of events) {
    const at = new Date(e.startsAt)
    if (Number.isNaN(at.getTime())) continue
    items.push({
      id: e.id,
      kind: 'event',
      title: e.title,
      at,
      allDay: Boolean(e.allDay),
      until: e.endsAt ? new Date(e.endsAt) : undefined,
      colour: e.color ?? undefined,
    })
  }

  for (const t of tasks) {
    if (!t.dueDate) continue
    const at = new Date(t.dueDate)
    if (Number.isNaN(at.getTime())) continue
    items.push({
      id: t.id,
      kind: 'task',
      title: t.title,
      at,
      // A task's due date is a day, not an appointment: it is shown with the
      // day's all-day items rather than pretending to a time nobody chose.
      allDay: true,
      done: t.status === 'DONE',
    })
  }

  // The window, in local days.
  const wanted: string[] = []
  const cursor = new Date(from)
  for (let i = 0; i < days; i++) {
    wanted.push(localDayKey(cursor, timezone))
    cursor.setDate(cursor.getDate() + 1)
  }
  const inWindow = new Set(wanted)

  const byDay = new Map<string, AgendaItem[]>()
  for (const item of items) {
    const key = localDayKey(item.at, timezone)
    if (!inWindow.has(key)) continue
    const list = byDay.get(key)
    if (list) list.push(item)
    else byDay.set(key, [item])
  }

  const out: AgendaDay[] = []
  for (const key of wanted) {
    const list = byDay.get(key)
    if (!list || list.length === 0) continue
    list.sort((a, b) => {
      // All-day first: it frames the day rather than sitting inside it.
      if (a.allDay !== b.allDay) return a.allDay ? -1 : 1
      return a.at.getTime() - b.at.getTime()
    })
    out.push({ day: new Date(`${key}T00:00:00`), key, items: list })
  }
  return out
}
