import { resolveEventColor } from '../../../lib/calendar/eventColor'
import type { CellProps } from '../monthview.types'

const DOTS = 4
const NEUTRAL = '#71717a'

/** Variant D's compact cell: up to four dots in the calendar colours. */
export function DotCell({ items, calendarColors }: CellProps) {
  return (
    <div className="flex items-center justify-center gap-0.5 h-2" data-testid="month-dots">
      {items.slice(0, DOTS).map(({ event }) => (
        <span
          key={event.id}
          className="w-1.5 h-1.5 rounded-full"
          style={{ backgroundColor: resolveEventColor(event, calendarColors) ?? NEUTRAL }}
        />
      ))}
    </div>
  )
}
