import type { CellProps } from '../monthview.types'
import { squareColour } from '../squareColour'
import { DotCell } from './DotCell'

/**
 * Variant E, "commit": the day's day tasks, after Atrioc's calendar (Denis,
 * 21.09: «calendar with 5 things every day… as you do them, they fill up the
 * day»). Ticks, a bar filling done/target, the block tinted by the result.
 *
 * - A taken day shows its tasks; a future day shows what is already taken.
 * - A past day left untaken after the start gets the ⬛ square from the shared
 *   rule, and shows an empty bar: a missed day, not a blank one.
 * - Events are dots underneath (Denis, 24.09: «at least show dots for events,
 *   it feels too empty otherwise»).
 */
export function CommitCell(props: CellProps) {
  const { dayTasks, square } = props
  const taken = Boolean(dayTasks?.confirmed)
  const missed = !taken && square !== undefined
  const colour = squareColour(square)
  const target = Math.max(dayTasks?.target ?? 5, dayTasks?.items.length ?? 0, 1)
  const done = taken ? (dayTasks?.done ?? 0) : 0

  return (
    <div className="flex flex-col gap-0.5 min-w-0">
      {(taken || missed) && (
        <div
          data-testid="month-commit"
          className="flex flex-col gap-0.5 min-w-0 rounded-sm px-0.5"
          style={colour && taken ? { backgroundColor: `${colour}26` } : undefined}
        >
          <div className="flex items-center gap-1">
            <div className="h-1.5 flex-1 rounded-sm bg-zinc-800 overflow-hidden">
              <div className="h-full" style={{ width: `${(done / target) * 100}%`, backgroundColor: colour ?? '#22c55e' }} />
            </div>
            <span className="text-[10px] text-zinc-400 tabular-nums">
              {done}/{target}
            </span>
          </div>
          {taken &&
            dayTasks!.items.map((item) => (
              <div
                key={item.task_id}
                className={`truncate text-[10px] leading-tight ${item.done ? 'text-zinc-300 line-through decoration-zinc-500' : 'text-zinc-100'}`}
                title={item.title}
              >
                {item.done ? '☑' : '☐'} {item.title}
              </div>
            ))}
        </div>
      )}
      <DotCell {...props} />
    </div>
  )
}
