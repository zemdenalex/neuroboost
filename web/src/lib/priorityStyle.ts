import { priorityMeta } from './priority'

/**
 * How a priority is drawn, the bot's own setting (settings.bot.priority_style,
 * bot/internal/format/priority.go). Настя, 18.09: the colours were «из разных
 * цветов, нет одного стиля»; Denis 21.09 chose three styles for the bot and on
 * 26.09 said the web follows the same setting (gap list row 16).
 */
export type PriorityStyle = 'circles' | 'dot' | 'dash'
export const PRIORITY_STYLES: PriorityStyle[] = ['circles', 'dot', 'dash']

export function readPriorityStyle(settings: object | undefined | null): PriorityStyle {
  const bot = (settings as { bot?: unknown } | undefined | null)?.bot
  const v = bot && typeof bot === 'object' ? (bot as { priority_style?: unknown }).priority_style : undefined
  return v === 'dot' || v === 'dash' ? v : 'circles'
}

export type PriorityMark = { kind: 'dot'; className: string } | { kind: 'text'; text: string }

export function priorityMark(style: PriorityStyle, p: number): PriorityMark {
  if (style === 'dot') return { kind: 'text', text: p === 0 ? '·0' : (p <= 2 ? '●' : '○') + p }
  if (style === 'dash') return { kind: 'text', text: '—' }
  return { kind: 'dot', className: priorityMeta(p).dotClass }
}
