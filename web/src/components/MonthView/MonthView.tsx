import { useCallback, useEffect, useMemo, useState, type ComponentType, type MouseEvent } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { dateLocale } from '../../utils/date'
import { monthGrid } from '../../lib/calendar/monthGrid'
import { eventsByDay } from '../../lib/calendar/monthCells'
import { createDayClick, routeCellClick } from '../../lib/calendar/monthClick'
import { todayInZone } from '../../lib/dayTasks/dayColour'
import { useDayTasks } from '../../lib/dayTasks/loadDayColours'
import { ListCell } from './cells/ListCell'
import { ClassicCell } from './cells/ClassicCell'
import { HeatCell } from './cells/HeatCell'
import { DotCell } from './cells/DotCell'
import { CommitCell } from './cells/CommitCell'
import { DayList } from './DayList'
import { squareColour } from './squareColour'
import { useMonthDrag } from './useMonthDrag'
import type { CellProps, MonthViewProps } from './monthview.types'
import type { MonthVariant } from '../../lib/calendar/monthVariant'

const CELLS: Record<MonthVariant, ComponentType<CellProps>> = {
  list: ListCell,
  classic: ClassicCell,
  heat: HeatCell,
  split: DotCell,
  commit: CommitCell,
}

/** Where each variant shows the day-task colour: a corner square or a tint of the cell. */
const SQUARE_IN_CORNER: Record<MonthVariant, boolean> = {
  list: true,
  classic: false,
  heat: false,
  split: true,
  commit: false,
}

const WEEKDAYS = [0, 1, 2, 3, 4, 5, 6]

/**
 * The web month (spec docs/team/architecture/V003-20260924-arc-web-month-view.md).
 *
 * The shell is shared by all five variants: 6×7 days from a Monday, the
 * header, click and double click, dragging. Only the cell differs.
 */
export function MonthView(props: MonthViewProps) {
  const { year, month, variant, events, timezone, calendarColors, headerExtra } = props
  const { onPrev, onNext, onToday, onOpenDay, onCreateOnDay, onMoveToDay } = props
  const { t, i18n } = useTranslation('calendar')
  const locale = dateLocale(i18n.language)

  const days = useMemo(() => monthGrid(year, month), [year, month])
  const byDay = useMemo(() => eventsByDay(events, days, timezone), [events, days, timezone])
  const dayTasks = useDayTasks(days[0], days[days.length - 1])
  const dayRecords = useMemo(() => Object.fromEntries(dayTasks.days.map((d) => [d.day, d])), [dayTasks.days])
  const today = todayInZone(new Date(), timezone)
  const monthPrefix = `${year}-${String(month).padStart(2, '0')}`

  const timeFormat = useMemo(
    () => new Intl.DateTimeFormat(locale, { hour: '2-digit', minute: '2-digit', timeZone: timezone }),
    [locale, timezone],
  )
  const title = new Date(Date.UTC(year, month - 1, 15)).toLocaleDateString(locale, {
    month: 'long',
    year: 'numeric',
    timeZone: 'UTC',
  })
  const weekdayNames = useMemo(
    () =>
      // 2026-09-21 is a Monday.
      WEEKDAYS.map((i) =>
        new Date(Date.UTC(2026, 8, 21 + i)).toLocaleDateString(locale, { weekday: 'short', timeZone: 'UTC' }),
      ),
    [locale],
  )

  const dayClick = useMemo(() => createDayClick(onOpenDay, onCreateOnDay), [onOpenDay, onCreateOnDay])
  useEffect(() => () => dayClick.cancel(), [dayClick])

  // Variant D: a click chooses the day for the list below (it has its own
  // "open week"); a double click still creates. Ruling, spec §5 R10.
  const [chosen, setChosen] = useState<string | null>(null)
  const chosenDay = chosen && days.includes(chosen) ? chosen : days.includes(today) ? today : `${monthPrefix}-01`
  const splitClick = useMemo(() => createDayClick(setChosen, onCreateOnDay), [onCreateOnDay])
  useEffect(() => () => splitClick.cancel(), [splitClick])

  const cancelClicks = useCallback(() => {
    dayClick.cancel()
    splitClick.cancel()
  }, [dayClick, splitClick])
  const drag = useMonthDrag(onMoveToDay, cancelClicks)

  const onCellClick = (day: string, e: MouseEvent) =>
    routeCellClick(variant === 'split', day, e.detail, drag.wasDrag(), { click: dayClick, splitClick, choose: setChosen })

  const Cell = CELLS[variant]
  const drags = variant === 'list' || variant === 'classic'

  return (
    <div data-testid="month-view" data-variant={variant} className="flex flex-col h-full min-h-0">
      <div className="px-2 py-2 border-b border-zinc-700 bg-zinc-900 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onPrev}
            aria-label={t('month.prev')}
            className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700"
          >
            ←
          </button>
          <button
            type="button"
            onClick={onNext}
            aria-label={t('month.next')}
            className="px-2 py-1 text-xs rounded bg-zinc-800 border border-zinc-700 hover:bg-zinc-700"
          >
            →
          </button>
          <h2 data-testid="month-title" className="font-semibold text-sm md:text-lg capitalize">
            {title}
          </h2>
        </div>
        <div className="flex items-center gap-2">
          {!today.startsWith(monthPrefix) && (
            <button
              type="button"
              onClick={onToday}
              className="px-2 py-1 text-xs rounded bg-zinc-700 hover:bg-zinc-600 text-white"
            >
              {t('today')}
            </button>
          )}
          {headerExtra}
        </div>
      </div>

      <div className="grid grid-cols-7 text-[11px] text-zinc-500 text-center border-b border-zinc-800">
        {weekdayNames.map((name) => (
          <div key={name} className="py-1 capitalize">
            {name}
          </div>
        ))}
      </div>

      {variant === 'commit' && !dayTasks.enabled && (
        <div className="px-3 py-2 text-xs text-zinc-400 border-b border-zinc-800 bg-zinc-950">
          {t('month.dayTasksOff')}.{' '}
          <Link to="/settings" className="text-blue-400 hover:underline">
            {t('month.openSettings')}
          </Link>
        </div>
      )}

      <div
        className={`grid grid-cols-7 grid-rows-6 min-h-0 ${variant === 'split' ? 'flex-[0_0_auto] h-[45%]' : 'flex-1'}`}
        data-testid="month-grid"
      >
        {days.map((day) => {
          const inMonth = day.startsWith(monthPrefix)
          const isToday = day === today
          const cellProps: CellProps = {
            day,
            items: byDay[day] ?? [],
            inMonth,
            isToday,
            square: dayTasks.colours[day],
            dayTasks: dayRecords[day],
            dayTasksEnabled: dayTasks.enabled,
            timeFormat,
            timezone,
            calendarColors,
            onItemPointerDown: drags ? drag.onItemPointerDown : undefined,
          }
          // Only the classic month tints the whole cell (R11); E tints its own block.
          const tint = variant === 'classic' ? squareColour(cellProps.square) : undefined
          return (
            <div
              key={day}
              data-day={day}
              data-testid="month-day"
              onClick={(e) => onCellClick(day, e)}
              className={[
                'border-r border-b border-zinc-800 p-1 min-h-0 overflow-hidden flex flex-col gap-0.5 cursor-pointer select-none',
                inMonth ? 'bg-black' : 'bg-zinc-950 opacity-50',
                drag.over === day ? 'ring-2 ring-inset ring-blue-500' : '',
                variant === 'split' && day === chosenDay ? 'ring-2 ring-inset ring-zinc-400' : '',
              ].join(' ')}
              style={tint ? { backgroundColor: `${tint}1f` } : undefined}
            >
              <div className="flex items-center justify-between">
                <span
                  className={
                    isToday
                      ? 'text-[11px] font-semibold px-1 rounded bg-blue-600 text-white tabular-nums'
                      : 'text-[11px] font-semibold text-zinc-300 tabular-nums'
                  }
                >
                  {Number(day.slice(8))}
                </span>
                {cellProps.square && SQUARE_IN_CORNER[variant] && (
                  <span data-testid="month-day-square" className="text-[10px] leading-none">
                    {cellProps.square}
                  </span>
                )}
              </div>
              <Cell {...cellProps} />
            </div>
          )
        })}
      </div>

      {variant === 'split' && (
        <DayList
          day={chosenDay}
          locale={locale}
          items={byDay[chosenDay] ?? []}
          square={dayTasks.colours[chosenDay]}
          dayTasks={dayRecords[chosenDay]}
          timeFormat={timeFormat}
          calendarColors={calendarColors}
          onOpenWeek={() => onOpenDay(chosenDay)}
        />
      )}
    </div>
  )
}

export default MonthView
