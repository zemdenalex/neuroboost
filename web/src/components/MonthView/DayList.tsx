import { useTranslation } from 'react-i18next'
import { resolveEventColor } from '../../lib/calendar/eventColor'
import type { MonthItem } from '../../lib/calendar/monthCells'
import type { Day } from '../../lib/dayTasks/dayColour'

const NEUTRAL = '#71717a'

interface Props {
  day: string
  locale: string
  items: MonthItem[]
  square?: string
  dayTasks?: Day
  timeFormat: Intl.DateTimeFormat
  calendarColors: Record<string, string | null>
  onOpenWeek: () => void
}

/** Variant D's lower half: the chosen day as a list, like the bot's calendar → day. */
export function DayList({ day, locale, items, square, dayTasks, timeFormat, calendarColors, onOpenWeek }: Props) {
  const { t } = useTranslation('calendar')
  const title = new Date(day + 'T12:00:00Z').toLocaleDateString(locale, {
    weekday: 'short',
    day: 'numeric',
    month: 'long',
    timeZone: 'UTC',
  })
  return (
    <div data-testid="month-day-list" className="border-t border-zinc-700 bg-zinc-950 p-3 overflow-y-auto max-h-[45%]">
      <div className="flex items-center gap-2 mb-2">
        <h3 className="text-sm font-semibold capitalize">{title}</h3>
        {square && <span className="text-xs">{square}</span>}
        {dayTasks?.confirmed && (
          <span className="text-xs text-zinc-400 tabular-nums">
            📌 {dayTasks.done}/{dayTasks.target}
          </span>
        )}
        <button
          type="button"
          onClick={onOpenWeek}
          className="ml-auto px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700"
        >
          {t('month.openWeek')}
        </button>
      </div>
      {items.length === 0 && <p className="text-xs text-zinc-500">{t('month.free')}</p>}
      <ul className="flex flex-col">
        {items.map(({ event, first, last }) => (
          <li key={event.id} className="flex gap-3 py-1.5 border-b border-zinc-800 text-sm">
            <span className="w-20 shrink-0 whitespace-nowrap text-zinc-500 tabular-nums">
              {event.allDay
                ? t('allDay')
                : first
                  ? timeFormat.format(new Date(event.startsAt))
                  : last
                    ? `→ ${timeFormat.format(new Date(event.endsAt))}`
                    : '…'}
            </span>
            <span
              className="pl-2 border-l-[3px] truncate"
              style={{ borderLeftColor: resolveEventColor(event, calendarColors) ?? NEUTRAL }}
            >
              {event.title}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
