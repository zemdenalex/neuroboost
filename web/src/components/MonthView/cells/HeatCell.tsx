import { busyShare } from '../../../lib/calendar/busyShare'
import type { CellProps } from '../monthview.types'
import { squareColour } from '../squareColour'

const NO_COLOUR = '#52525b'

/**
 * Variant C, "heat": no text. The fill rises with how full the day was, in the
 * day-task colour; "events·tasks" in the corner; titles in the tooltip.
 */
export function HeatCell({ day, items, square, dayTasks, timezone }: CellProps) {
  const share = busyShare(
    items.map((i) => i.event),
    day,
    timezone,
  )
  const titles = items.map((i) => i.event.title).join(' · ')
  return (
    <div className="relative flex-1 min-h-6 rounded-sm bg-zinc-900 overflow-hidden" title={titles || undefined} data-testid="month-heat">
      <div
        className="absolute inset-x-0 bottom-0 opacity-80"
        style={{ height: `${Math.round(share * 100)}%`, backgroundColor: squareColour(square) ?? NO_COLOUR }}
      />
      <span className="absolute bottom-0.5 right-1 text-[10px] text-zinc-200 tabular-nums">
        {items.length}·{dayTasks ? dayTasks.done : 0}
      </span>
    </div>
  )
}
