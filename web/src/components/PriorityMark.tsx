import { useAuthContext } from '../contexts/AuthContext'
import { priorityMark, readPriorityStyle } from '../lib/priorityStyle'

/**
 * A task's priority as the person chose to see it, in the web as in the bot
 * (settings.bot.priority_style, gap list row 16): a coloured dot, «●1 ○3», or
 * «—». Decorative next to a label, so hidden from screen readers.
 */
export function PriorityMark({
  priority,
  size = 'sm',
  circleClass,
  className = '',
}: {
  priority: number
  size?: 'sm' | 'md'
  /** Overrides the dot colour in the circles style (Eisenhower's quadrant fallback). */
  circleClass?: string
  className?: string
}) {
  const { user } = useAuthContext()
  const mark = priorityMark(readPriorityStyle(user?.settings), priority)
  if (mark.kind === 'dot') {
    const box = size === 'md' ? 'w-3 h-3' : 'w-2 h-2'
    return <span aria-hidden className={`inline-block rounded-full flex-shrink-0 ${box} ${circleClass ?? mark.className} ${className}`} />
  }
  return (
    <span aria-hidden data-testid="priority-mark-text" className={`flex-shrink-0 font-mono text-xs leading-none text-zinc-400 ${className}`}>
      {mark.text}
    </span>
  )
}
