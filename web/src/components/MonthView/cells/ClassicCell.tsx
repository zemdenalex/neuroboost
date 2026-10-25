import { useTranslation } from 'react-i18next'
import { cellRows } from '../../../lib/calendar/monthCells'
import { resolveEventColor } from '../../../lib/calendar/eventColor'
import type { CellProps } from '../monthview.types'

const ROWS = 2
const NEUTRAL = '#52525b'

/** Variant B, "classic": chips filled with the calendar colour, no times, "N more". */
export function ClassicCell({ items, calendarColors, onItemPointerDown }: CellProps) {
  const { t } = useTranslation('calendar')
  const { shown, more } = cellRows(items, ROWS)
  return (
    <div className="flex flex-col gap-0.5 min-w-0">
      {shown.map(({ event }) => (
        <div
          key={event.id}
          data-testid="month-item"
          data-event-id={event.id}
          onPointerDown={onItemPointerDown ? (e) => onItemPointerDown(event, e) : undefined}
          className="truncate rounded-sm px-1 text-[11px] leading-tight text-white cursor-grab"
          style={{ backgroundColor: resolveEventColor(event, calendarColors) ?? NEUTRAL }}
          title={event.title}
        >
          {event.title}
        </div>
      ))}
      {more > 0 && <div className="text-[11px] text-zinc-500">{t('month.more', { count: more })}</div>}
    </div>
  )
}
