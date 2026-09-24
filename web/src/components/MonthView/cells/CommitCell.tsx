import type { CellProps } from '../monthview.types'
import { squareColour } from '../squareColour'

/**
 * Variant E, "commit": the day's day tasks, after Atrioc's calendar (Denis,
 * 21.09: «calendar with 5 things every day… as you do them, they fill up the
 * day»). Ticks, a bar filling done/target, the cell tinted by the result.
 * A future day shows what is already taken; an untaken day shows nothing.
 */
export function CommitCell({ dayTasks, square }: CellProps) {
  if (!dayTasks || !dayTasks.confirmed) return null
  const colour = squareColour(square)
  const target = Math.max(dayTasks.target, dayTasks.items.length, 1)
  return (
    <div
      data-testid="month-commit"
      className="flex flex-col gap-0.5 min-w-0 rounded-sm px-0.5"
      style={colour ? { backgroundColor: `${colour}26` } : undefined}
    >
      <div className="flex items-center gap-1">
        <div className="h-1.5 flex-1 rounded-sm bg-zinc-800 overflow-hidden">
          <div
            className="h-full"
            style={{ width: `${(dayTasks.done / target) * 100}%`, backgroundColor: colour ?? '#22c55e' }}
          />
        </div>
        <span className="text-[10px] text-zinc-400 tabular-nums">
          {dayTasks.done}/{target}
        </span>
      </div>
      {dayTasks.items.map((item) => (
        <div
          key={item.task_id}
          className={`truncate text-[10px] leading-tight ${item.done ? 'text-zinc-300 line-through decoration-zinc-500' : 'text-zinc-100'}`}
          title={item.title}
        >
          {item.done ? '☑' : '☐'} {item.title}
        </div>
      ))}
    </div>
  )
}
