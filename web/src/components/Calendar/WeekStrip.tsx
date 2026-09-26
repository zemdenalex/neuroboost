import { squareColour } from '../MonthView/squareColour';

export interface WeekStripDay {
  /** YYYY-MM-DD in the user's zone. */
  day: string;
  /** Short weekday + date, already localised ("пн 21"). */
  label: string;
  /** 0..1 — how full the day is (busyShare), the heatmap's own measure. */
  share: number;
  /** Day-task square, when the day has one. */
  square?: string;
}

interface Props {
  days: WeekStripDay[];
  /** The day the grid below shows. */
  shown: string;
  today: string;
  onPick: (index: number) => void;
}

const NO_COLOUR = '#3b82f6';

/**
 * Phone month variant C, "week strip" (Denis 26.09: «A + C + D», page
 * https://claude.ai/artifact/NRbFMBYbLHj6yPkGMXmheo). Not a month: the seven
 * days of the week over the one-day grid, each with a bar for how full it is,
 * in the day-task colour when the day has one. A tap shows that day below.
 */
export function WeekStrip({ days, shown, today, onPick }: Props) {
  return (
    <div data-testid="week-strip" className="grid grid-cols-7 gap-1 px-2 py-1.5 border-b border-zinc-800 bg-zinc-950">
      {days.map((d, i) => {
        const on = d.day === shown;
        return (
          <button
            key={d.day}
            type="button"
            data-testid="week-strip-day"
            data-day={d.day}
            aria-pressed={on}
            onClick={() => onPick(i)}
            className={[
              'flex flex-col items-center rounded py-1 text-[10px] leading-tight tabular-nums',
              on ? 'bg-zinc-800 text-white ring-1 ring-blue-500' : d.day === today ? 'text-blue-400' : 'text-zinc-400',
            ].join(' ')}
          >
            <span className="capitalize whitespace-nowrap">{d.label}</span>
            <span className="mt-1 h-1 w-[80%] rounded-sm bg-zinc-800 overflow-hidden">
              <span
                className="block h-full"
                style={{ width: `${Math.round(d.share * 100)}%`, backgroundColor: squareColour(d.square) ?? NO_COLOUR }}
              />
            </span>
          </button>
        );
      })}
    </div>
  );
}
