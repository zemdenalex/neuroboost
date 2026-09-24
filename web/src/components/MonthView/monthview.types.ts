import type { PointerEvent as ReactPointerEvent, ReactNode } from 'react'
import type { NbEvent } from '../../types'
import type { MonthItem } from '../../lib/calendar/monthCells'
import type { MonthVariant } from '../../lib/calendar/monthVariant'
import type { Day } from '../../lib/dayTasks/dayColour'

/** What every cell renderer receives for its day. */
export interface CellProps {
  day: string
  items: MonthItem[]
  inMonth: boolean
  isToday: boolean
  /** The day-task square for this day, if it has one. */
  square?: string
  /** The server's day-tasks record for this day, if taken. */
  dayTasks?: Day
  dayTasksEnabled: boolean
  timeFormat: Intl.DateTimeFormat
  timezone: string
  calendarColors: Record<string, string | null>
  /** Starts dragging an event; only the variants that draw events pass it on. */
  onItemPointerDown?: (event: NbEvent, e: ReactPointerEvent) => void
}

export interface MonthViewProps {
  year: number
  month: number
  variant: MonthVariant
  /** How long a click waits for a second one before it opens the week. */
  clickWaitMs: number
  events: NbEvent[]
  timezone: string
  calendarColors: Record<string, string | null>
  onPrev: () => void
  onNext: () => void
  onToday: () => void
  /** A single click: open the day's week. */
  onOpenDay: (day: string) => void
  /** A double click: create an event on the day. */
  onCreateOnDay: (day: string) => void
  /** An event dropped on another day; the shift is toDay minus fromDay. */
  onMoveToDay: (event: NbEvent, fromDay: string, toDay: string) => void
  /** Controls for the right of the header: the view switch, the filter. */
  headerExtra?: ReactNode
}
