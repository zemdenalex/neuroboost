import { useTranslation } from 'react-i18next'
import { cellRows, type MonthItem } from '../../../lib/calendar/monthCells'
import { resolveEventColor } from '../../../lib/calendar/eventColor'
import type { CellProps } from '../monthview.types'

const ROWS = 3
const NEUTRAL = '#71717a'

/**
 * Variant A, "list": the week's look in a month cell. A calendar-colour stripe,
 * the time in grey on the day the event starts, the title; "+N more" past three.
 */
export function ListCell({ items, timeFormat, calendarColors, onItemPointerDown }: CellProps) {
  const { t } = useTranslation('calendar')
  const { shown, more } = cellRows(items, ROWS)
  return (
    <div className="flex flex-col gap-0.5 min-w-0">
      {shown.map(({ event, first }: MonthItem) => (
        <div
          key={event.id}
          data-testid="month-item"
          data-event-id={event.id}
          onPointerDown={onItemPointerDown ? (e) => onItemPointerDown(event, e) : undefined}
          className="flex items-baseline gap-1 min-w-0 pl-1 border-l-[3px] text-[11px] leading-tight cursor-grab"
          style={{ borderLeftColor: resolveEventColor(event, calendarColors) ?? NEUTRAL }}
          title={event.title}
        >
          {first && !event.allDay && (
            <span className="text-zinc-500 tabular-nums shrink-0">{timeFormat.format(new Date(event.startsAt))}</span>
          )}
          <span className="truncate text-zinc-200">{event.title}</span>
        </div>
      ))}
      {more > 0 && <div className="text-[11px] text-zinc-500">{t('month.more', { count: more })}</div>}
    </div>
  )
}
